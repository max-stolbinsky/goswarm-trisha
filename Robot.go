// Trisha v3: ladder-safe CPU. Modules used sparingly.
// DM: hold facing the enemy half; break hold on contact; then hunt.
// CTF/Classic: cheap unseen search + capped drone.

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
var dx = [4]int{0, 1, 0, -1}
var dy = [4]int{-1, 0, 1, 0}

const (
	dmHoldTicks  = 900 // long enough vs rushers; alarm cuts it short
	droneShotMax = 4   // cover the flag box, then rely on foot search
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
		u = &Unit{}
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

// droneAim: walk a few points so 9×9 shots cover the whole flag box
// (a single center shot misses the rim — e.g. flag at 15,15 vs aim 20,20).
func droneAim() (int, int) {
	lo, hi := flagBox()
	// 9×9 covers ±4; two offset centers span a 10×10 box.
	points := [4][2]int{
		{lo + 4, lo + 4},
		{hi - 4, lo + 4},
		{lo + 4, hi - 4},
		{hi - 4, hi - 4},
	}
	i := droneShot % 4
	cx, cy := points[i][0], points[i][1]
	if cx < lo {
		cx = lo
	}
	if cy < lo {
		cy = lo
	}
	if cx > hi {
		cx = hi
	}
	if cy > hi {
		cy = hi
	}
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
	cx, cy := get_pos()
	u.tx, u.ty = x, y
	u.budget = 3*(abs(x-cx)+abs(y-cy)) + 20
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
		horiz := abs(ddx) >= abs(ddy)
		if zigzag && ddx != 0 && ddy != 0 {
			horiz = !u.horiz
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
		shoot()
		return
	}
	if move_forward() {
		u.moves++
		u.horiz = want == Left || want == Right
		markHere()
		return
	}
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
	// Starter-style box sweep: nearest corner → zigzag to opposite → roam.
	switch u.phase {
	case 0:
		cx, cy := lo, lo
		if abs(x-hi) < abs(x-lo) {
			cx = hi
		}
		if abs(y-hi) < abs(y-lo) {
			cy = hi
		}
		target(u, cx, cy)
		u.phase = 1
	case 1:
		if near(u) || u.budget <= 0 {
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
	if u.phase != 2 && u.moves >= 8 {
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

func hunt(u *Unit) {
	if holding() {
		dmHold(u)
		return
	}
	x, y := get_pos()
	g := get_grid_size()
	// Chase last known contact first.
	if enemyAge > 0 && tick-enemyAge < 120 {
		if u.tx != enemyX || u.ty != enemyY || u.budget <= 0 {
			target(u, enemyX, enemyY)
		}
		step(u, false)
		return
	}
	if u.budget <= 0 || near(u) {
		if u.phase == 0 {
			target(u, g-1-x, g-1-y)
			u.phase = 1
		} else {
			target(u, 2+randInt(g-4), 2+randInt(g-4))
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

func builder() {
	if busy() > 0 {
		return
	}
	if my_units() < 6 {
		if !build_warrior() {
			turn_right()
		}
		return
	}
	turn_right()
}

func loop() {
	id := my_id()
	if id <= lastID {
		tick++
		droneDone = false // new tick: allow one drone try again if ready
	}
	lastID = id
	initMods()
	u := me()

	what, dist := look_forward()
	noteHit(what, dist)

	mode := game_mode()
	// Drone only in CTF/classic while flag unknown — never during DM hold.
	if mode != "deathmatch" && !flagKnown {
		maybeDrone()
	}

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
		// Under fire: leave DM hold even if we tank with the shield.
		alarm = true
		if !(hasShield && shielded() && dist > 2) {
			dodgeAway()
			u.dodge = 1
			return
		}
	}
	// Shield just broken → treat as contact.
	if hasShield && !shielded() {
		alarm = true
	}

	// Radar in all modes (incl. DM hold), one unit per tick.
	if radarSnap(u) {
		return
	}

	if my_kind() == "builder" {
		builder()
		return
	}
	if u.scan > 0 {
		u.scan--
		turn_right()
		return
	}
	if mode == "deathmatch" {
		hunt(u)
		return
	}
	// Light coverage for the flag box (bounded — not a full map paint).
	markHere()
	if dist >= 1 {
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
	search(u)
}
