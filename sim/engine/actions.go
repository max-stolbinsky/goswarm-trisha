package engine

// --- unit helpers ---

func (g *Game) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < g.Grid && y < g.Grid
}

func (g *Game) unitAt(x, y int) *Unit {
	for _, u := range g.Units {
		if u.Alive && u.X == x && u.Y == y {
			return u
		}
	}
	return nil
}

func (g *Game) builtWallAt(x, y int) *BuiltWall {
	for _, w := range g.Built {
		if w.X == x && w.Y == y && w.HP > 0 {
			return w
		}
	}
	return nil
}

func (g *Game) constructionAt(x, y int) *Construction {
	for _, c := range g.Constructions {
		if c.X == x && c.Y == y && c.Left > 0 {
			return c
		}
	}
	return nil
}

func (g *Game) cellBlockedForMove(x, y int, u *Unit) bool {
	if !g.InBounds(x, y) || g.Walls[Key(x, y)] {
		return true
	}
	if g.builtWallAt(x, y) != nil {
		return true
	}
	if g.constructionAt(x, y) != nil {
		return true
	}
	if other := g.unitAt(x, y); other != nil && other.ID != u.ID {
		return true
	}
	if g.isMyFlagCell(x, y, u.Owner) {
		return true
	}
	return false
}

func (g *Game) canAct(u *Unit) bool {
	if !u.Alive || u.Acted {
		return false
	}
	if u.Kind == KindBuilder && u.Busy > 0 {
		return false
	}
	return true
}

func (g *Game) markActed(u *Unit) {
	u.Acted = true
}

// MoveForward moves one cell forward.
func (g *Game) MoveForward(u *Unit) bool {
	if !g.canAct(u) {
		return false
	}
	nx, ny := u.X+DX[u.Dir], u.Y+DY[u.Dir]
	if g.cellBlockedForMove(nx, ny, u) {
		return false
	}
	if g.walkIntoBullet(u, nx, ny) {
		g.markActed(u)
		return false
	}
	u.X, u.Y = nx, ny
	g.checkFlagTouch(u)
	g.markActed(u)
	return true
}

// Boost is explicit boost action (2 cells).
func (g *Game) Boost(u *Unit) bool {
	if !g.Players[u.Owner].HasBoost || u.BoostCD > 0 {
		return false
	}
	if !g.canAct(u) {
		return false
	}
	steps := 2
	_ = steps
	for i := 1; i <= 2; i++ {
		nx, ny := u.X+DX[u.Dir]*i, u.Y+DY[u.Dir]*i
		if !g.InBounds(nx, ny) || g.Walls[Key(nx, ny)] {
			return false
		}
		if g.builtWallAt(nx, ny) != nil || g.constructionAt(nx, ny) != nil {
			return false
		}
		if i == 1 && g.isMyFlagCell(nx, ny, u.Owner) {
			return false
		}
		if other := g.unitAt(nx, ny); other != nil && other.ID != u.ID {
			return false
		}
	}
	nx, ny := u.X+DX[u.Dir]*2, u.Y+DY[u.Dir]*2
	if g.walkIntoBullet(u, nx, ny) {
		g.markActed(u)
		return false
	}
	u.X, u.Y = nx, ny
	u.BoostCD = 20
	g.checkFlagTouch(u)
	g.markActed(u)
	return true
}

func (g *Game) TurnLeft(u *Unit) bool {
	if !u.Alive || u.Acted {
		return false
	}
	if u.Kind == KindBuilder && u.Busy > 0 {
		return false
	}
	u.Dir = (u.Dir + 3) % 4
	g.markActed(u)
	return true
}

func (g *Game) TurnRight(u *Unit) bool {
	if !u.Alive || u.Acted {
		return false
	}
	if u.Kind == KindBuilder && u.Busy > 0 {
		return false
	}
	u.Dir = (u.Dir + 1) % 4
	g.markActed(u)
	return true
}

func (g *Game) Shoot(u *Unit) bool {
	if !g.canAct(u) || u.Kind == KindBuilder {
		return false
	}
	fx, fy := u.X+DX[u.Dir], u.Y+DY[u.Dir]
	if !g.InBounds(fx, fy) || g.Walls[Key(fx, fy)] {
		g.markActed(u)
		return false
	}
	if target := g.unitAt(fx, fy); target != nil && target.Alive {
		if target.Owner != u.Owner || g.FriendlyFire {
			g.damageUnit(target, u.Owner)
			g.markActed(u)
			return true
		}
	}
	if bw := g.builtWallAt(fx, fy); bw != nil {
		bw.HP--
		if bw.HP <= 0 {
			g.removeBuiltWall(bw)
		}
		g.markActed(u)
		return true
	}
	g.Bullets = append(g.Bullets, &Bullet{X: fx, Y: fy, Dir: u.Dir, Owner: u.Owner, Alive: true})
	g.markActed(u)
	return true
}

func (g *Game) BuildWall(u *Unit) bool {
	if u.Kind != KindBuilder || u.Busy > 0 || u.Acted {
		return false
	}
	if g.Mode == ModeClassic || g.Mode == ModeDeathmatch {
		return false
	}
	if g.wallCount(u.Owner) >= 30 {
		return false
	}
	fx, fy := u.X+DX[u.Dir], u.Y+DY[u.Dir]
	if !g.cellFreeForBuild(fx, fy) {
		return false
	}
	ticks := 3
	if g.Players[u.Owner].HasFastBuild {
		ticks = 2
	}
	g.Constructions = append(g.Constructions, &Construction{
		X: fx, Y: fy, Owner: u.Owner, BuilderID: u.ID,
		Left: ticks, IsWall: true, Dir: u.Dir,
	})
	u.Busy = ticks
	g.markActed(u)
	return true
}

func (g *Game) BuildWarrior(u *Unit) bool {
	if u.Kind != KindBuilder || u.Busy > 0 || u.Acted {
		return false
	}
	if g.Mode == ModeClassic {
		return false
	}
	if g.warriorCount(u.Owner) >= 5 {
		return false
	}
	fx, fy := u.X+DX[u.Dir], u.Y+DY[u.Dir]
	if !g.cellFreeForBuild(fx, fy) {
		return false
	}
	ticks := 10
	if g.Players[u.Owner].HasFastBuild {
		ticks = 8
	}
	g.Constructions = append(g.Constructions, &Construction{
		X: fx, Y: fy, Owner: u.Owner, BuilderID: u.ID,
		Left: ticks, IsWall: false, Kind: KindWarrior, Dir: u.Dir,
	})
	u.Busy = ticks
	g.markActed(u)
	return true
}

func (g *Game) cellFreeForBuild(x, y int) bool {
	if !g.InBounds(x, y) || g.Walls[Key(x, y)] {
		return false
	}
	if g.builtWallAt(x, y) != nil || g.constructionAt(x, y) != nil {
		return false
	}
	if g.unitAt(x, y) != nil {
		return false
	}
	for _, b := range g.Bullets {
		if b.Alive && b.X == x && b.Y == y {
			return false
		}
	}
	for _, f := range g.Flags {
		if f.X == x && f.Y == y {
			return false
		}
	}
	return true
}

func (g *Game) wallCount(owner int) int {
	n := 0
	for _, w := range g.Built {
		if w.Owner == owner && w.HP > 0 {
			n++
		}
	}
	for _, c := range g.Constructions {
		if c.Owner == owner && c.IsWall && c.Left > 0 {
			n++
		}
	}
	return n
}

func (g *Game) warriorCount(owner int) int {
	n := 0
	for _, u := range g.Units {
		if u.Alive && u.Owner == owner && u.Kind == KindWarrior {
			n++
		}
	}
	for _, c := range g.Constructions {
		if c.Owner == owner && !c.IsWall && c.Left > 0 {
			n++
		}
	}
	return n
}

func (g *Game) walkIntoBullet(u *Unit, x, y int) bool {
	for _, b := range g.Bullets {
		if !b.Alive || b.X != x || b.Y != y {
			continue
		}
		if b.Owner == u.Owner && !g.FriendlyFire {
			continue
		}
		g.killUnit(u)
		return true
	}
	return false
}

func (g *Game) damageUnit(u *Unit, attackerOwner int) {
	if u.Shielded {
		u.Shielded = false
		return
	}
	g.killUnit(u)
	_ = attackerOwner
}

func (g *Game) killUnit(u *Unit) {
	if !u.Alive {
		return
	}
	u.Alive = false
	g.Wrecks[Key(u.X, u.Y)] = 10
	g.removeConstructionsForBuilder(u.ID)
	u.Busy = 0
}

func (g *Game) removeBuiltWall(w *BuiltWall) {
	for i, bw := range g.Built {
		if bw == w {
			g.Built = append(g.Built[:i], g.Built[i+1:]...)
			return
		}
	}
}

func (g *Game) removeConstructionsForBuilder(builderID int) {
	out := g.Constructions[:0]
	for _, c := range g.Constructions {
		if c.BuilderID != builderID {
			out = append(out, c)
		}
	}
	g.Constructions = out
}

func (g *Game) checkFlagTouch(u *Unit) {
	if g.Mode == ModeClassic {
		for _, f := range g.Flags {
			if u.X == f.X && u.Y == f.Y {
				g.Outcome = Win0
				if u.Owner == 1 {
					g.Outcome = Win1
				}
				g.Message = "Flag captured"
				g.Done = true
				return
			}
		}
	}
	if g.Mode == ModeCTF {
		for _, f := range g.Flags {
			if f.Owner >= 0 && f.Owner != u.Owner && u.X == f.X && u.Y == f.Y {
				if u.Owner == 0 {
					g.Outcome = Win0
				} else {
					g.Outcome = Win1
				}
				g.Message = "Enemy flag touched"
				g.Done = true
				return
			}
		}
	}
}

// MyUnits returns alive unit count for owner.
func (g *Game) MyUnits(owner int) int {
	n := 0
	for _, u := range g.Units {
		if u.Alive && u.Owner == owner {
			n++
		}
	}
	return n
}

func (g *Game) Busy(u *Unit) int {
	if u.Kind != KindBuilder {
		return 0
	}
	return u.Busy
}

func (g *Game) BoostReady(u *Unit) int {
	if !g.Players[u.Owner].HasBoost {
		return -1
	}
	return u.BoostCD
}

func (g *Game) DroneReady(owner int) int {
	if !g.Players[owner].HasDrone {
		return -1
	}
	return g.Players[owner].DroneCD
}

func (g *Game) Drone(owner int, x, y int) ([][]string, bool) {
	if !g.Players[owner].HasDrone || g.Players[owner].DroneCD > 0 {
		return nil, false
	}
	cells := make([][]string, 9)
	for r := 0; r < 9; r++ {
		cells[r] = make([]string, 9)
		for c := 0; c < 9; c++ {
			cells[r][c] = g.CellLabel(x-4+c, y-4+r, owner)
		}
	}
	g.Players[owner].DroneCD = 50
	return cells, true
}

func (g *Game) MyFlag(owner int) (int, int) {
	if g.Mode != ModeCTF {
		return -1, -1
	}
	for _, f := range g.Flags {
		if f.Owner == owner {
			return f.X, f.Y
		}
	}
	return -1, -1
}

func (g *Game) RandInt(owner int, n int) int {
	_ = owner
	return g.RNG.Intn(n)
}
