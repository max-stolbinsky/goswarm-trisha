package engine

import "sort"

// RunTick advances one game tick. runners[0] and runners[1] invoke bot loops.
type UnitRunner func(player int, u *Unit) error

// RunTick executes bullet flight, unit actions, construction, wrecks, and win checks.
func (g *Game) RunTick(run UnitRunner) {
	if g.Done {
		return
	}
	g.Tick++
	g.advanceBullets()
	g.tickCooldowns()
	for _, u := range g.Units {
		if u.Alive {
			u.Acted = false
		}
	}

	first := g.pickFirstPlayer()
	order := []int{first, 1 - first}

	for _, p := range order {
		if g.Players[p].Stopped {
			continue
		}
		units := g.livingUnits(p)
		for _, u := range units {
			g.Current = u
			if run != nil {
				if err := run(p, u); err != nil {
					g.stopPlayer(p, "runtime error: "+err.Error())
				}
			}
		}
	}

	g.advanceConstructions()
	g.clearWrecks()
	g.checkEnd()
	if g.Mode == ModeClassic {
		g.FirstPlayer = 1 - g.FirstPlayer
	}
}

// AddTickCPU records interpreter time for a player this tick (ms after scale).
func (g *Game) AddTickCPU(player int, ms float64) {
	if !g.EnableCPU {
		return
	}
	g.Players[player].TickCPU += ms
	g.Players[player].TotalCPU += ms
}

// FinalizeTickCPU applies per-tick limits after all units ran.
func (g *Game) FinalizeTickCPU() {
	if !g.EnableCPU {
		return
	}
	for p := 0; p < 2; p++ {
		if g.Players[p].Stopped {
			continue
		}
		if g.Players[p].TickCPU > g.TickLimitMS {
			g.stopPlayer(p, "tick cpu limit")
		} else if g.Players[p].TotalCPU > g.GameLimitMS {
			g.stopPlayer(p, "game cpu limit")
		}
		g.Players[p].TickCPU = 0
	}
}

func (g *Game) stopPlayer(p int, reason string) {
	g.Players[p].Stopped = true
	if g.Players[p].StopTick == 0 {
		g.Players[p].StopTick = g.Tick
	}
	g.Players[p].StopReason = reason
}

func (g *Game) pickFirstPlayer() int {
	if g.Mode == ModeClassic {
		return g.FirstPlayer
	}
	return g.RNG.Intn(2)
}

func (g *Game) livingUnits(owner int) []*Unit {
	var out []*Unit
	for _, u := range g.Units {
		if u.Alive && u.Owner == owner {
			out = append(out, u)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (g *Game) tickCooldowns() {
	for _, u := range g.Units {
		if u.BoostCD > 0 {
			u.BoostCD--
		}
	}
	for p := 0; p < 2; p++ {
		if g.Players[p].DroneCD > 0 {
			g.Players[p].DroneCD--
		}
	}
}

func (g *Game) advanceBullets() {
	big := [2]bool{g.Players[0].HasBigGun, g.Players[1].HasBigGun}
	var next []*Bullet
	for _, b := range g.Bullets {
		if !b.Alive {
			continue
		}
		steps := 1
		if big[b.Owner] {
			steps = 2
		}
		alive := true
		for s := 0; s < steps && alive; s++ {
			nx, ny := b.X+DX[b.Dir], b.Y+DY[b.Dir]
			if !g.InBounds(nx, ny) || g.Walls[Key(nx, ny)] {
				alive = false
				break
			}
			if bw := g.builtWallAt(nx, ny); bw != nil {
				bw.HP--
				if bw.HP <= 0 {
					g.removeBuiltWall(bw)
				}
				alive = false
				break
			}
			if u := g.unitAt(nx, ny); u != nil {
				if u.Owner != b.Owner || g.FriendlyFire {
					g.damageUnit(u, b.Owner)
				}
				alive = false
				break
			}
			b.X, b.Y = nx, ny
		}
		if alive {
			next = append(next, b)
		}
	}
	g.Bullets = next
}

func (g *Game) advanceConstructions() {
	var remain []*Construction
	for _, c := range g.Constructions {
		if c.Left <= 0 {
			continue
		}
		builder := g.findUnit(c.BuilderID)
		c.Left--
		if builder != nil && builder.Alive {
			builder.Busy = c.Left
		}
		if c.Left > 0 {
			remain = append(remain, c)
			continue
		}
		if c.IsWall {
			g.Built = append(g.Built, &BuiltWall{X: c.X, Y: c.Y, HP: 3, Owner: c.Owner})
		} else {
			id := g.NextUnitID
			g.NextUnitID++
			nu := &Unit{
				ID: id, Owner: c.Owner, X: c.X, Y: c.Y, Dir: c.Dir,
				Kind: KindWarrior, Alive: true,
			}
			if g.Players[c.Owner].HasShield {
				nu.Shielded = true
			}
			g.Units = append(g.Units, nu)
		}
		if builder != nil {
			builder.Busy = 0
		}
	}
	g.Constructions = remain
}

func (g *Game) findUnit(id int) *Unit {
	for _, u := range g.Units {
		if u.ID == id {
			return u
		}
	}
	return nil
}

func (g *Game) clearWrecks() {
	for k, left := range g.Wrecks {
		left--
		if left <= 0 {
			delete(g.Wrecks, k)
		} else {
			g.Wrecks[k] = left
		}
	}
}

func (g *Game) aliveCount(owner int) int {
	n := 0
	for _, u := range g.Units {
		if u.Alive && u.Owner == owner {
			n++
		}
	}
	return n
}

func (g *Game) checkEnd() {
	if g.Done {
		return
	}
	a0, a1 := g.aliveCount(0), g.aliveCount(1)
	if g.Mode == ModeDeathmatch || g.Mode == ModeCTF {
		if a0 == 0 && a1 > 0 {
			g.Outcome, g.Message, g.Done = Win1, "Eliminated all enemies", true
			return
		}
		if a1 == 0 && a0 > 0 {
			g.Outcome, g.Message, g.Done = Win0, "Eliminated all enemies", true
			return
		}
	}
	if g.Players[0].Stopped && g.Players[1].Stopped {
		g.Outcome = Draw
		g.Message = "All players crashed/finished"
		g.Done = true
		return
	}
	if g.Tick >= g.MaxSteps {
		g.finishBySurvivors(a0, a1)
	}
}

func (g *Game) finishBySurvivors(a0, a1 int) {
	g.Done = true
	g.Message = "Step limit"
	if a0 > a1 {
		g.Outcome = Win0
	} else if a1 > a0 {
		g.Outcome = Win1
	} else {
		g.Outcome = Draw
	}
}

// Play runs until done.
func (g *Game) Play(run UnitRunner) {
	for !g.Done && g.Tick < g.MaxSteps {
		g.RunTick(run)
		g.FinalizeTickCPU()
	}
}
