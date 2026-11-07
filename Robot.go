// Trisha v2: lean CPU + Drone / Shield / Radar modules.
// Long DM hold, uncleared-cell search for Classic/CTF.

const (
	Up = iota
	Right
	Down
	Left
)

type Unit struct {
	tx, ty, budget, phase int
	moves, scan, dodge    int
	visits                map[int]int
}

var units = map[int]*Unit{}
var seen [40][40]bool
var wallAt [40][40]bool
var tick, lastID int
var flagKnown bool
var flagX, flagY int
var enemyX, enemyY, enemyAge int
var hasRadar, hasDrone, hasShield = -1, -1, -1 // -1 unk, 0 no, 1 yes
var droneTick int                     // last tick we launched the drone
var dx = [4]int{0, 1, 0, -1}
var dy = [4]int{-1, 0, 1, 0}

const dmHoldTicks = 1400

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func key(x, y int) int { return x*1000 + y }

func me() *Unit {
	id := my_id()
	u, ok := units[id]
	if !ok {
		u = &Unit{visits: map[int]int{}}
		units[id] = u
	}
	return u
}

func ensureMods() {
	if hasRadar < 0 {
		if has_module("radar") {
			hasRadar = 1
		} else {
			hasRadar = 0
		}
	}
	if hasDrone < 0 {
		if has_module("drone") {
			hasDrone = 1
		} else {
			hasDrone = 0
		}
	}
	if hasShield < 0 {
		if has_module("shield") {
			hasShield = 1
		} else {
			hasShield = 0
		}
	}
}

func turnTo(dir int) {
	if (dir-get_dir()+4)%4 == 3 {
		turn_left()
	} else {
		turn_right()
	}
}

func markSeen(x, y int) {
	if x >= 0 && y >= 0 && x < 40 && y < 40 {
		seen[y][x] = true
	}
}

func markWall(x, y int) {
	if x >= 0 && y >= 0 && x < 40 && y < 40 {
		wallAt[y][x] = true
		seen[y][x] = true
	}
}

func isWallXY(x, y int) bool {
	if x < 0 || y < 0 || x >= 40 || y >= 40 {
		return true
	}
	return wallAt[y][x]
}

func observeDir(what string, dist, dir int) {
	if dist < 1 {
		return
	}
	x, y := get_pos()
	// Full ray only in Classic/CTF search; DM just needs the hit cell.
	if game_mode() != "deathmatch" {
		lim := dist
		if lim > 10 {
			lim = 10
		}
		for i := 1; i < lim; i++ {
			markSeen(x+dx[dir]*i, y+dy[dir]*i)
		}
	}
	ex, ey := x+dx[dir]*dist, y+dy[dir]*dist
	if what == "wall" || what == "my_flag" {
		markWall(ex, ey)
		return
	}
	markSeen(ex, ey)
	if what == "flag" || what == "enemy_flag" {
		flagKnown = true
		flagX, flagY = ex, ey
	}
	if what == "enemy" {
		enemyX, enemyY, enemyAge = ex, ey, tick
	}
}

func observe(what string, dist int) {
	markSeen(get_pos())
	x, y := get_pos()
	_ = x
	_ = y
	observeDir(what, dist, get_dir())
}

func ingestCell(label string, cx, cy int) {
	if cx < 0 || cy < 0 || cx >= 40 || cy >= 40 {
		return
	}
	switch label {
	case "wall", "my_flag":
		markWall(cx, cy)
	case "empty", "ally", "enemy", "flag", "enemy_flag", "built_wall", "construction", "bullet":
		markSeen(cx, cy)
		if label == "flag" || label == "enemy_flag" {
			flagKnown = true
			flagX, flagY = cx, cy
		}
		if label == "enemy" {
			enemyX, enemyY, enemyAge = cx, cy, tick
		}
	default:
		if label != "" {
			markSeen(cx, cy)
		}
	}
}

// tryDrone scouts the flag zone (CTF/classic) or enemy half (DM). Shared CD.
func tryDrone() bool {
	ensureMods()
	if hasDrone != 1 || drone_ready() != 0 {
		return false
	}
	// One launch per ready window (first unit that reaches here).
	if tick == droneTick {
		return false
	}
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
		if flagKnown {
			cx, cy = flagX, flagY
		}
	} else if game_mode() == "deathmatch" {
		x, y := get_pos()
		cx, cy = g-1-x, g-1-y
		if enemyAge > 0 && tick-enemyAge < 200 {
			cx, cy = enemyX, enemyY
		}
	} else if flagKnown {
		cx, cy = flagX, flagY
	}
	cells, ok := drone(cx, cy)
	if !ok {
		return false
	}
	droneTick = tick
	// cells[r][c] covers (cx-4+c, cy-4+r); center is [4][4].
	for r := 0; r < 9; r++ {
		if r >= len(cells) {
			break
		}
		row := cells[r]
		for c := 0; c < 9; c++ {
			if c >= len(row) {
				break
			}
			ingestCell(row[c], cx-4+c, cy-4+r)
		}
	}
	return true
}

// radarReact: side looks. Turn toward enemy/flag/bullet — costs the action.
func radarReact(u *Unit) bool {
	ensureMods()
	if hasRadar != 1 {
		return false
	}
	wl, dl := look_left()
	observeDir(wl, dl, (get_dir()+3)%4)
	wr, dr := look_right()
	observeDir(wr, dr, (get_dir()+1)%4)

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
	// Side bullet: step out if we still have a shield, else turn away.
	if wl == "bullet" && dl <= 4 {
		if hasShield == 1 && shielded() {
			return false
		}
		turn_right()
		u.dodge = 1
		return true
	}
	if wr == "bullet" && dr <= 4 {
		if hasShield == 1 && shielded() {
			return false
		}
		turn_left()
		u.dodge = 1
		return true
	}
	return false
}

func target(u *Unit, x, y int) {
	u.tx, u.ty = x, y
	u.budget = 120
	u.visits = map[int]int{}
}

func near(u *Unit) bool {
	x, y := get_pos()
	return abs(u.tx-x) + abs(u.ty-y) <= 1
}

func navigate(u *Unit) {
	x, y := get_pos()
	g := get_grid_size()
	dir := get_dir()
	best, want := 1000000, dir
	for i := 0; i < 4; i++ {
		d := (dir + i) % 4
		nx, ny := x+dx[d], y+dy[d]
		if nx < 0 || ny < 0 || nx >= g || ny >= g || isWallXY(nx, ny) {
			continue
		}
		score := 4*(abs(nx-u.tx)+abs(ny-u.ty)) + u.visits[key(nx, ny)]*9
		turn := (d - dir + 4) % 4
		if turn == 3 {
			turn = 1
		}
		score += turn
		if score < best {
			best = score
			want = d
		}
	}
	if get_dir() != want {
		turnTo(want)
		return
	}
	what, dist := look_forward()
	observe(what, dist)
	if dist == 1 && what == "built_wall" && my_kind() != "builder" {
		shoot()
		return
	}
	if dist == 1 && (what == "ally" || what == "construction" || what == "bullet") {
		u.visits[key(x, y)]++
		turn_right()
		return
	}
	if move_forward() {
		u.moves++
		nx, ny := get_pos()
		u.visits[key(nx, ny)]++
		markSeen(nx, ny)
	} else {
		u.visits[key(x, y)]++
	}
	u.budget--
}

func pickUnseen(u *Unit, lo, hi int) {
	x, y := get_pos()
	best, bx, by := 1000000, lo, lo
	for a := lo; a <= hi; a++ {
		for b := lo; b <= hi; b++ {
			if a < 0 || b < 0 || a >= 40 || b >= 40 {
				continue
			}
			if seen[b][a] || wallAt[b][a] {
				continue
			}
			score := abs(x-a) + abs(y-b)
			if game_mode() == "ctf" {
				score += ((a + b + my_id()) % 3) * 2
			}
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
		navigate(u)
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
	needNew := u.budget <= 0 || (x == u.tx && y == u.ty)
	if !needNew && u.tx >= 0 && u.ty >= 0 && u.tx < 40 && u.ty < 40 && seen[u.ty][u.tx] {
		needNew = true
	}
	if needNew {
		pickUnseen(u, lo, hi)
	}
	if u.moves >= 6 && (game_mode() != "classic" || (x >= lo && x <= hi && y >= lo && y <= hi)) {
		u.moves = 0
		u.scan = 3
		turn_right()
		return
	}
	navigate(u)
}

func hunt(u *Unit) {
	if tick < dmHoldTicks {
		turn_right()
		return
	}
	if enemyAge > 0 && tick-enemyAge < 80 {
		target(u, enemyX, enemyY)
		navigate(u)
		return
	}
	x, y := get_pos()
	g := get_grid_size()
	if u.budget <= 0 || (x == u.tx && y == u.ty) {
		if u.phase == 0 {
			target(u, g-1-x, g-1-y)
			u.phase = 1
		} else {
			target(u, 2+randInt(g-4), 2+randInt(g-4))
		}
	}
	if u.moves >= 4 {
		u.moves = 0
		u.scan = 3
		turn_right()
		return
	}
	navigate(u)
}

func builder(u *Unit, what string, dist int) {
	_ = u
	_ = what
	_ = dist
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
	}
	lastID = id
	u := me()
	ensureMods()
	markSeen(get_pos())

	what, dist := look_forward()
	observe(what, dist)

	// Drone once per cooldown; skip during early DM hold (spinning is enough).
	if !(game_mode() == "deathmatch" && tick < dmHoldTicks) {
		tryDrone()
	}

	if busy() > 0 {
		return
	}
	if u.dodge > 0 {
		u.dodge = 0
		if dist > 1 || what == "flag" || what == "enemy_flag" {
			move_forward()
		} else {
			turn_right()
		}
		return
	}
	// Front bullet: dodge unless shield still up and the shot is far.
	if what == "bullet" && dist <= 8 {
		canTank := hasShield == 1 && shielded() && dist > 2
		if !canTank {
			x, y := get_pos()
			d := (get_dir() + 1) % 4
			g := get_grid_size()
			nx, ny := x+dx[d], y+dy[d]
			if nx < 0 || ny < 0 || nx >= g || ny >= g || isWallXY(nx, ny) {
				turn_left()
			} else {
				turn_right()
			}
			u.dodge = 1
			return
		}
	}
	if what == "enemy" && my_kind() != "builder" {
		shoot()
		return
	}
	if what == "flag" || what == "enemy_flag" {
		u.scan = 0
		move_forward()
		return
	}
	if radarReact(u) {
		return
	}
	if my_kind() == "builder" {
		builder(u, what, dist)
		return
	}
	if u.scan > 0 {
		u.scan--
		turn_right()
		return
	}
	if game_mode() == "deathmatch" {
		hunt(u)
		return
	}
	search(u)
}
