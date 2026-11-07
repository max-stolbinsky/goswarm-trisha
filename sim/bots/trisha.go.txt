// Trisha v3: ladder-safe CPU. Three mode strategies in separate loops.
// Classic: drone + center-box sweep.
// Deathmatch: hold, then pack hunt.
// CTF: guard+walls on approaches, camp-beacon raid.

const (
	Up = iota
	Right
	Down
	Left
)

type Unit struct {
	tx, ty, budget, phase int
	moves, scan, dodge    int
	detour, detourDir     int
	stuck                 int // consecutive failed progress
	lx, ly                int // last position (stuck detect)
	horiz                 bool
}

var units = map[int]*Unit{}
var seen [40][40]bool
var tick, lastID int
var flagKnown bool
var flagX, flagY int
var enemyX, enemyY, enemyAge int
var alarm bool // DM: contact seen — leave the corner
var modsReady bool
var hasRadar, hasDrone, hasShield bool
var droneDone bool // one drone launch attempt this ready-window
var droneShot int  // how many launches this game (cap)
var walled [4]bool // CTF: four sides around our flag
var wallTries [4]int
var warriorRank = map[int]int{} // birth order among warriors
var nextWarriorRank int
var dx = [4]int{0, 1, 0, -1}
var dy = [4]int{-1, 0, 1, 0}

const (
	dmHoldTicks  = 900 // long enough vs rushers; alarm cuts it short
	droneShotMax = 4   // cover the flag box, then rely on foot search
	ctfGuards    = 1    // one approach guard (sweep: best overall with rear walls)
	ctfRearWalls = true // seal rear sides after army is full
)

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func me() *Unit {
	id := my_id()
	u, ok := units[id]
	if !ok {
		x, y := get_pos()
		u = &Unit{lx: x, ly: y}
		units[id] = u
	}
	return u
}

func initMods() {
	if modsReady {
		return
	}
	hasRadar = has_module("radar")
	hasDrone = has_module("drone")
	hasShield = has_module("shield")
	modsReady = true
}

func turnTo(dir int) {
	if (dir-get_dir()+4)%4 == 3 {
		turn_left()
	} else {
		turn_right()
	}
}

func noteHit(what string, dist int) {
	if dist < 1 {
		return
	}
	x, y := get_pos()
	d := get_dir()
	ex, ey := x+dx[d]*dist, y+dy[d]*dist
	if what == "flag" || what == "enemy_flag" {
		flagKnown = true
		flagX, flagY = ex, ey
	}
	if what == "enemy" {
		enemyX, enemyY, enemyAge = ex, ey, tick
		alarm = true
	}
	if what == "bullet" {
		alarm = true
	}
}

func markHere() {
	x, y := get_pos()
	if x >= 0 && y >= 0 && x < 40 && y < 40 {
		seen[y][x] = true
	}
}

func markCell(x, y int) {
	if x >= 0 && y >= 0 && x < 40 && y < 40 {
		seen[y][x] = true
	}
}

func flagBox() (lo, hi int) {
	g := get_grid_size()
	if game_mode() == "ctf" {
		home := g / 4
		if home > 8 {
			home = 8
		}
		fx, _ := my_flag()
		lo, hi = g-2-home, g-3
		if fx >= g/2 {
			lo, hi = 2, 1+home
		}
		return
	}
	// Classic: flag is in the center 10×10.
	lo, hi = g/2-5, g/2+4
	if lo < 0 {
		lo = 0
	}
	if hi >= g {
		hi = g - 1
	}
	return
}

// droneAim: first shot on the camp beacon, then cover the box corners.
func droneAim() (int, int) {
	lo, hi := flagBox()
	if droneShot == 0 && game_mode() == "ctf" {
		return campBeacon()
	}
	// 9×9 covers ±4; offset centers span the home box.
	points := [4][2]int{
		{lo + 4, lo + 4},
		{hi - 4, lo + 4},
		{lo + 4, hi - 4},
		{hi - 4, hi - 4},
	}
	i := droneShot % 4
	cx, cy := clamp(points[i][0], lo, hi), clamp(points[i][1], lo, hi)
	return cx, cy
}

// Drone: sparse scouting of the flag box; stop after a few shots or when found.
func maybeDrone() {
	if !hasDrone || flagKnown || droneShot >= droneShotMax || drone_ready() != 0 {
		return
	}
	if droneDone {
		return
	}
	droneDone = true
	cx, cy := droneAim()
	cells, ok := drone(cx, cy)
	if !ok {
		return
	}
	droneShot++
	for r := 0; r < len(cells) && r < 9; r++ {
		row := cells[r]
		for c := 0; c < len(row) && c < 9; c++ {
			lab := row[c]
			if lab == "flag" || lab == "enemy_flag" {
				flagKnown = true
				flagX, flagY = cx-4+c, cy-4+r
				return
			}
		}
	}
}

// Radar at most once per tick for the whole squad.
func radarSnap(u *Unit) bool {
	if !hasRadar {
		return false
	}
	if my_id()%5 != tick%5 {
		return false
	}
	wl, dl := look_left()
	wr, dr := look_right()
	if wl == "enemy" || wl == "bullet" || wl == "flag" || wl == "enemy_flag" {
		noteSide(wl, dl, true)
	}
	if wr == "enemy" || wr == "bullet" || wr == "flag" || wr == "enemy_flag" {
		noteSide(wr, dr, false)
	}
	if my_kind() != "builder" {
		if wl == "enemy" {
			turn_left()
			return true
		}
		if wr == "enemy" {
			turn_right()
			return true
		}
	}
	if wl == "flag" || wl == "enemy_flag" {
		turn_left()
		return true
	}
	if wr == "flag" || wr == "enemy_flag" {
		turn_right()
		return true
	}
	if (wl == "bullet" || wr == "bullet") && !(hasShield && shielded()) {
		if wl == "bullet" {
			turn_right()
		} else {
			turn_left()
		}
		u.dodge = 1
		return true
	}
	return false
}

func noteSide(what string, dist int, left bool) {
	if dist < 1 {
		return
	}
	x, y := get_pos()
	d := get_dir()
	if left {
		d = (d + 3) % 4
	} else {
		d = (d + 1) % 4
	}
	ex, ey := x+dx[d]*dist, y+dy[d]*dist
	if what == "enemy" || what == "bullet" {
		enemyX, enemyY, enemyAge = ex, ey, tick
		alarm = true
	}
	if what == "flag" || what == "enemy_flag" {
		flagKnown = true
		flagX, flagY = ex, ey
	}
}

// Preferred facing during DM hold: toward the mirrored enemy half.
func holdDir() int {
	x, y := get_pos()
	g := get_grid_size()
	// Bias by unit id so the squad does not share one blind cone.
	id := my_id()
	if id%2 == 0 {
		if x < g/2 {
			return Right
		}
		return Left
	}
	if y < g/2 {
		return Down
	}
	return Up
}

// Dodge toward the map interior, not into the map edge.
func dodgeAway() {
	x, y := get_pos()
	g := get_grid_size()
	d := get_dir()
	// Prefer a side step toward center.
	right := (d + 1) % 4
	left := (d + 3) % 4
	rx, ry := x+dx[right], y+dy[right]
	lx, ly := x+dx[left], y+dy[left]
	scoreR := abs(rx-g/2) + abs(ry-g/2)
	scoreL := abs(lx-g/2) + abs(ly-g/2)
	if rx < 0 || ry < 0 || rx >= g || ry >= g {
		scoreR = 1000
	}
	if lx < 0 || ly < 0 || lx >= g || ly >= g {
		scoreL = 1000
	}
	if scoreL < scoreR {
		turn_left()
	} else {
		turn_right()
	}
}

func target(u *Unit, x, y int) {
	targetAt(u, inland(x), inland(y))
}

func targetAt(u *Unit, x, y int) {
	cx, cy := get_pos()
	u.tx, u.ty = x, y
	u.budget = 3*(abs(u.tx-cx)+abs(u.ty-cy)) + 20
	u.stuck = 0
	u.detour = 0
	// Prefer the longer axis first → one bend (L-path), not a staircase.
	u.horiz = abs(u.tx-cx) >= abs(u.ty-cy)
}

// Keep waypoints off the map rim — rim pacing is a common soft-stall.
func inland(v int) int {
	g := get_grid_size()
	return clamp(v, 4, g-5)
}

func near(u *Unit) bool {
	x, y := get_pos()
	return abs(u.tx-x)+abs(u.ty-y) <= 1
}

func toward(ddx, ddy int, horiz bool) int {
	if (horiz && ddx != 0) || ddy == 0 {
		if ddx > 0 {
			return Right
		}
		if ddx < 0 {
			return Left
		}
		return get_dir()
	}
	if ddy > 0 {
		return Down
	}
	return Up
}

// Starter-like step with a short detour so a wall on a straight line
// cannot soft-lock the unit (flip horiz alone does nothing when dy==0).
func step(u *Unit, zigzag bool) {
	u.budget--
	x, y := get_pos()
	ddx, ddy := u.tx-x, u.ty-y
	want := u.detourDir
	if u.detour > 0 {
		u.detour--
	} else {
		horiz := u.horiz
		if zigzag && ddx != 0 && ddy != 0 {
			horiz = !u.horiz
		} else if !zigzag {
			// Hold the chosen axis until cleared (L-path).
			if u.horiz {
				horiz = ddx != 0
			} else {
				horiz = ddx != 0 && ddy == 0
			}
		} else {
			horiz = abs(ddx) >= abs(ddy)
		}
		want = toward(ddx, ddy, horiz)
	}
	if get_dir() != want {
		turnTo(want)
		return
	}
	what, dist := look_forward()
	noteHit(what, dist)
	if what == "enemy" && my_kind() != "builder" {
		shoot()
		return
	}
	if what == "built_wall" && dist == 1 && my_kind() != "builder" {
		fx, fy := my_flag()
		// Near home these are almost always ours — shooting them opens the flag.
		if abs(x-fx)+abs(y-fy) > 4 {
			shoot()
			return
		}
		u.detourDir = (want + 1 + 2*(my_id()%2)) % 4
		u.detour = 2
		turnTo(u.detourDir)
		return
	}
	// Ally ahead: step aside instead of queuing up.
	if what == "ally" && dist == 1 {
		u.detourDir = (want + 1 + 2*(my_id()%2)) % 4
		u.detour = 2 + my_id()%3
		turnTo(u.detourDir)
		return
	}
	if move_forward() {
		u.moves++
		// Keep L-path axis sticky; only flip after a real axis change intent in zigzag.
		if zigzag {
			u.horiz = want == Left || want == Right
		}
		markHere()
		nx, ny := get_pos()
		if nx == u.lx && ny == u.ly {
			u.stuck++
		} else {
			u.stuck = 0
			u.lx, u.ly = nx, ny
		}
		return
	}
	u.stuck++
	// Blocked: peel off along the other axis for a few ticks.
	altHoriz := want == Up || want == Down
	u.detourDir = toward(ddx, ddy, altHoriz)
	if u.detourDir == want || randInt(4) == 0 {
		u.detourDir = (want + 1 + 2*randInt(2)) % 4
	}
	u.detour = 2 + randInt(4)
	u.horiz = !u.horiz
}

func pickUnseen(u *Unit, lo, hi int) {
	x, y := get_pos()
	best, bx, by := 1000000, lo, lo
	for a := lo; a <= hi; a++ {
		for b := lo; b <= hi; b++ {
			if a < 0 || b < 0 || a >= 40 || b >= 40 || seen[b][a] {
				continue
			}
			score := abs(x-a) + abs(y-b)
			if score < best {
				best = score
				bx, by = a, b
			}
		}
	}
	if best == 1000000 {
		w := hi - lo + 1
		if w < 1 {
			w = 1
		}
		bx = lo + randInt(w)
		by = lo + randInt(w)
	}
	target(u, bx, by)
}

// Enemy camp beacon: mirror of our flag, clamped into their home box.
// Exact flag is rarely on the mirror (~5.5 cells off), but the camp always is.
func campBeacon() (int, int) {
	lo, hi := flagBox()
	fx, fy := my_flag()
	g := get_grid_size()
	return clamp(g-1-fx, lo, hi), clamp(g-1-fy, lo, hi)
}

func search(u *Unit) {
	if flagKnown {
		if u.tx != flagX || u.ty != flagY {
			target(u, flagX, flagY)
		}
		step(u, false)
		return
	}
	lo, hi := flagBox()
	x, y := get_pos()
	// CTF: L-rush to camp beacon, then sweep the box. Classic: corner sweep.
	switch u.phase {
	case 0:
		if game_mode() == "ctf" {
			bx, by := campBeacon()
			target(u, bx, by)
		} else {
			cx, cy := lo, lo
			if abs(x-hi) < abs(x-lo) {
				cx = hi
			}
			if abs(y-hi) < abs(y-lo) {
				cy = hi
			}
			target(u, cx, cy)
		}
		u.phase = 1
	case 1:
		if near(u) || u.budget <= 0 {
			// Sweep to the opposite corner of the box.
			target(u, lo+hi-u.tx, lo+hi-u.ty)
			u.phase = 2
		}
	case 2:
		if near(u) || u.budget <= 0 {
			u.phase = 3
		}
	default:
		need := near(u) || u.budget <= 0
		if !need && u.tx >= 0 && u.ty >= 0 && u.tx < 40 && u.ty < 40 && seen[u.ty][u.tx] {
			need = true
		}
		if need {
			pickUnseen(u, lo, hi)
			u.scan = 3
			turn_right()
			return
		}
	}
	// Don't spin during the camp rush — only after we arrive.
	if u.phase >= 3 && u.moves >= 8 {
		u.moves = 0
		u.scan = 3
		turn_right()
		return
	}
	step(u, u.phase == 2)
}

func holding() bool {
	return !alarm && tick < dmHoldTicks
}

func dmHold(u *Unit) {
	// Rare glance left/right via radar (one unit/tick) already ran in loop.
	// Mostly face the enemy half; every ~7 ticks peek with a turn.
	if tick%7 == my_id()%7 {
		turn_right()
		return
	}
	want := holdDir()
	if get_dir() != want {
		turnTo(want)
		return
	}
	// Already facing out: idle this tick (look already done). Saves CPU.
}

// Per-unit offset on a 2D grid so hunters fan out.
func packOffset() (int, int) {
	lane := my_id() % 5
	return (lane%3 - 1) * 3, (lane/3 - 1) * 3 // roughly -3..3 on both axes
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func freshRoam(u *Unit) {
	g := get_grid_size()
	ox, oy := packOffset()
	// Prefer the far half + private offset; never the rim.
	tx := inland(g/4 + randInt(g/2) + ox)
	ty := inland(g/4 + randInt(g/2) + oy)
	// Push opposite to current edge if hugging the rim.
	x, y := get_pos()
	if x <= 3 {
		tx = inland(g/2 + randInt(g/2-4))
	}
	if x >= g-4 {
		tx = inland(4 + randInt(g/2-4))
	}
	if y <= 3 {
		ty = inland(g/2 + randInt(g/2-4))
	}
	if y >= g-4 {
		ty = inland(4 + randInt(g/2-4))
	}
	target(u, tx, ty)
	u.phase = 2
	enemyAge = 0 // drop a stale chase that pinned us to a wall
}

func hunt(u *Unit) {
	if holding() {
		dmHold(u)
		return
	}
	x, y := get_pos()
	g := get_grid_size()
	ox, oy := packOffset()

	// Stuck on a few tiles (edge pacing): abandon target and roam inland.
	if u.stuck >= 5 || (u.budget <= 0 && abs(x-u.lx)+abs(y-u.ly) <= 1 && u.moves > 0) {
		freshRoam(u)
		turn_right()
		return
	}

	// Short chase window; offsets stay inland so we don't pin to the rim.
	if enemyAge > 0 && tick-enemyAge < 50 && u.stuck < 3 {
		tx := inland(enemyX + ox)
		ty := inland(enemyY + oy)
		if u.tx != tx || u.ty != ty || u.budget <= 0 {
			target(u, tx, ty)
		}
		step(u, false)
		return
	}
	if u.budget <= 0 || near(u) {
		if u.phase == 0 {
			tx := inland(g - 1 - x + ox)
			ty := inland(g - 1 - y + oy)
			target(u, tx, ty)
			u.phase = 1
		} else {
			freshRoam(u)
		}
	}
	if u.moves >= 5 {
		u.moves = 0
		u.scan = 3
		turn_right()
		return
	}
	step(u, false)
}

func makeWarrior() {
	if !build_warrior() {
		turn_right()
	}
}

func warriorOrder() int {
	id := my_id()
	if r, ok := warriorRank[id]; ok {
		return r
	}
	r := nextWarriorRank
	nextWarriorRank++
	warriorRank[id] = r
	return r
}

func isGuard() bool {
	if ctfGuards <= 0 || my_kind() != "warrior" {
		return false
	}
	r := warriorOrder()
	// Spawn warrior (rank 0) always raids; next ctfGuards are posted on approaches.
	return r >= 1 && r < 1+ctfGuards
}

// Sides of our flag that face the map center (enemy approaches).
func approachSides() [4]int {
	fx, fy := my_flag()
	mid := get_grid_size() / 2
	n := 0
	var out [4]int
	for side := Up; side <= Left; side++ {
		nx, ny := fx+dx[side], fy+dy[side]
		if abs(nx-mid)+abs(ny-mid) < abs(fx-mid)+abs(fy-mid) {
			out[n] = side
			n++
		}
	}
	// Pack unused as -1 so callers know length via first -1... use n via helper.
	for i := n; i < 4; i++ {
		out[i] = -1
	}
	return out
}

func approachCount(sides [4]int) int {
	n := 0
	for i := 0; i < 4 && sides[i] >= 0; i++ {
		n++
	}
	return n
}

func tryWallSide(u *Unit, side int, what string, dist int) bool {
	if walled[side] {
		return false
	}
	fx, fy := my_flag()
	sx, sy := fx+2*dx[side], fy+2*dy[side]
	g := get_grid_size()
	if sx < 0 || sy < 0 || sx >= g || sy >= g {
		walled[side] = true
		return false
	}
	x, y := get_pos()
	if x != sx || y != sy {
		if u.tx != sx || u.ty != sy {
			targetAt(u, sx, sy)
		}
		if u.budget <= 0 {
			walled[side] = true
			return false
		}
		step(u, false)
		return true
	}
	face := (side + 2) % 4
	if get_dir() != face {
		turnTo(face)
		return true
	}
	if (what == "built_wall" && dist == 1) || build_wall() || wallTries[side] > 5 {
		walled[side] = true
	}
	wallTries[side]++
	return true
}

// Hold the approach post outside the wall; face outward; peek with radar.
func guard(u *Unit) {
	gi := warriorOrder() - 1
	sides := approachSides()
	n := approachCount(sides)
	if n == 0 {
		search(u)
		return
	}
	if gi < 0 {
		gi = 0
	}
	side := sides[gi%n]
	fx, fy := my_flag()
	px, py := fx+2*dx[side], fy+2*dy[side]
	g := get_grid_size()
	if px < 0 || py < 0 || px >= g || py >= g {
		search(u)
		return
	}
	x, y := get_pos()
	if x != px || y != py {
		if u.tx != px || u.ty != py || u.budget <= 0 {
			targetAt(u, px, py)
		}
		step(u, false)
		return
	}
	if get_dir() != side {
		turnTo(side)
		return
	}
	// Side radar every tick (guards are few; skip the shared id%5 queue).
	if hasRadar {
		wl, _ := look_left()
		wr, _ := look_right()
		if wl == "enemy" {
			turn_left()
			return
		}
		if wr == "enemy" {
			turn_right()
			return
		}
	}
}

func dmBuilder() {
	if busy() > 0 {
		return
	}
	if my_units() < 6 {
		makeWarrior()
		return
	}
	turn_right()
}

// CTF: spawn raider out; guard+wall on each approach; fill army; optional rear walls.
func ctfBuilder(u *Unit, what string, dist int) {
	if busy() > 0 {
		return
	}
	front := approachSides()
	fn := approachCount(front)
	for i := 0; i < fn; i++ {
		side := front[i]
		if i < ctfGuards {
			need := 2 + (i + 1) // builder + spawn raider + guards so far
			if my_units() < need {
				makeWarrior()
				return
			}
		}
		if tryWallSide(u, side, what, dist) {
			return
		}
	}
	if my_units() < 6 {
		makeWarrior()
		return
	}
	if ctfRearWalls {
		for side := Up; side <= Left; side++ {
			if tryWallSide(u, side, what, dist) {
				return
			}
		}
	}
	turn_right()
}

func markSight(dist int) {
	markHere()
	if dist < 1 {
		return
	}
	x, y := get_pos()
	d := get_dir()
	lim := dist
	if lim > 8 {
		lim = 8
	}
	for i := 1; i <= lim; i++ {
		markCell(x+dx[d]*i, y+dy[d]*i)
	}
}

func finishScan(u *Unit) bool {
	if u.scan <= 0 {
		return false
	}
	u.scan--
	turn_right()
	return true
}

// Classic: find the center flag. No walls, no army beyond spawn.
func classicLoop(u *Unit, what string, dist int) {
	if !flagKnown {
		maybeDrone()
	}
	if my_kind() == "builder" {
		turn_right()
		return
	}
	if radarSnap(u) || finishScan(u) {
		return
	}
	markSight(dist)
	search(u)
}

// Deathmatch: hold the spawn facing out, then hunt as a pack.
func dmLoop(u *Unit, what string, dist int) {
	if hasShield && !shielded() {
		alarm = true
	}
	if my_kind() == "builder" {
		dmBuilder()
		return
	}
	if radarSnap(u) || finishScan(u) {
		return
	}
	hunt(u)
}

// CTF: wall+guard the flag, rush the enemy camp beacon, then box-sweep.
func ctfLoop(u *Unit, what string, dist int) {
	if !flagKnown {
		maybeDrone()
	}
	if my_kind() == "builder" {
		ctfBuilder(u, what, dist)
		return
	}
	warriorOrder()
	if isGuard() {
		if what == "built_wall" && dist == 1 {
			fx, fy := my_flag()
			x, y := get_pos()
			if abs(x-fx)+abs(y-fy) <= 4 {
				turn_right()
				return
			}
		}
		guard(u)
		return
	}
	if radarSnap(u) || finishScan(u) {
		return
	}
	markSight(dist)
	search(u)
}

func loop() {
	id := my_id()
	if id <= lastID {
		tick++
		droneDone = false
	}
	lastID = id
	initMods()
	u := me()

	what, dist := look_forward()
	noteHit(what, dist)

	if busy() > 0 {
		return
	}
	if u.dodge > 0 {
		u.dodge = 0
		move_forward()
		return
	}
	if what == "flag" || what == "enemy_flag" {
		u.scan = 0
		move_forward()
		return
	}
	if what == "enemy" && my_kind() != "builder" {
		shoot()
		return
	}
	if what == "bullet" && dist <= 6 {
		alarm = true
		if !(hasShield && shielded() && dist > 2) {
			dodgeAway()
			u.dodge = 1
			return
		}
	}

	switch game_mode() {
	case "deathmatch":
		dmLoop(u, what, dist)
	case "ctf":
		ctfLoop(u, what, dist)
	default:
		classicLoop(u, what, dist)
	}
}
