// Trisha v3: ladder-safe CPU. Modules used sparingly.
// DM: hold-and-spin (Starter-style). CTF/Classic: cheap unseen search.
// Hot path avoids maps, ray paints, and per-tick radar/drone work.

const (
	Up = iota
	Right
	Down
	Left
)

type Unit struct {
	tx, ty, budget, phase int
	moves, scan, dodge    int
	horiz                 bool
}

var units = map[int]*Unit{}
var seen [40][40]bool
var tick, lastID int
var flagKnown bool
var flagX, flagY int
var modsReady bool
var hasRadar, hasDrone, hasShield bool
var droneDone bool // one drone launch attempt this ready-window
var dx = [4]int{0, 1, 0, -1}
var dy = [4]int{-1, 0, 1, 0}

const dmHoldTicks = 1400

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
	if what == "flag" || what == "enemy_flag" {
		x, y := get_pos()
		d := get_dir()
		flagKnown = true
		flagX, flagY = x+dx[d]*dist, y+dy[d]*dist
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

// Drone: only until flag found; scan 9×9 only for flag/enemy labels (no full paint).
func maybeDrone() {
	if !hasDrone || flagKnown || drone_ready() != 0 {
		return
	}
	if droneDone {
		return
	}
	droneDone = true
	g := get_grid_size()
	cx, cy := g/2, g/2
	if game_mode() == "ctf" {
		home := g / 4
		if home > 8 {
			home = 8
		}
		fx, _ := my_flag()
		lo, hi := g-2-home, g-3
		if fx >= g/2 {
			lo, hi = 2, 1+home
		}
		cx, cy = (lo+hi)/2, (lo+hi)/2
	}
	cells, ok := drone(cx, cy)
	if !ok {
		return
	}
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

// Radar at most once per tick for the whole squad (unit id == tick%8 style).
func radarSnap(u *Unit) bool {
	if !hasRadar {
		return false
	}
	if my_id()%5 != tick%5 {
		return false
	}
	wl, _ := look_left()
	wr, _ := look_right()
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

// Starter-like step: no visit maps, one look when facing target.
func step(u *Unit, zigzag bool) {
	u.budget--
	x, y := get_pos()
	ddx, ddy := u.tx-x, u.ty-y
	horiz := abs(ddx) >= abs(ddy)
	if zigzag && ddx != 0 && ddy != 0 {
		horiz = !u.horiz
	}
	want := toward(ddx, ddy, horiz)
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
	x, y := get_pos()
	g := get_grid_size()
	if flagKnown {
		if u.tx != flagX || u.ty != flagY {
			target(u, flagX, flagY)
		}
		step(u, false)
		return
	}
	lo, hi := g/2-5, g/2+4
	if lo < 0 {
		lo = 0
	}
	if hi >= g {
		hi = g - 1
	}
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
	}
	if u.budget <= 0 || (x == u.tx && y == u.ty) {
		pickUnseen(u, lo, hi)
	}
	if u.moves >= 8 {
		u.moves = 0
		u.scan = 3
		turn_right()
		return
	}
	step(u, u.phase == 2)
}

func hunt(u *Unit) {
	if tick < dmHoldTicks {
		turn_right()
		return
	}
	x, y := get_pos()
	g := get_grid_size()
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
		if !(hasShield && shielded() && dist > 2) {
			turn_right()
			u.dodge = 1
			return
		}
	}

	// Cheap DM hold: no radar, no maps — just spin and shoot (above).
	if mode == "deathmatch" && tick < dmHoldTicks {
		turn_right()
		return
	}

	// Radar only after hold / in CTF-classic, and only 1 unit per tick.
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
	// Mark stood cell cheaply for search coverage.
	markHere()
	if dist >= 1 {
		x, y := get_pos()
		d := get_dir()
		markCell(x+dx[d]*dist, y+dy[d]*dist)
	}
	search(u)
}
