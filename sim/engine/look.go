package engine

// LookForward returns the first obstacle in the unit's facing direction.
func (g *Game) LookForward(u *Unit) (what string, dist int) {
	return g.lookRay(u.X, u.Y, u.Dir, u)
}

// LookSide looks left (left=true) or right without turning.
func (g *Game) LookSide(u *Unit, left bool) (what string, dist int) {
	d := u.Dir
	if left {
		d = (d + 3) % 4
	} else {
		d = (d + 1) % 4
	}
	if g.Players[u.Owner].HasRadar {
		return g.lookRay(u.X, u.Y, d, u)
	}
	return "", 0
}

func (g *Game) lookRay(x, y, dir int, viewer *Unit) (string, int) {
	for step := 1; ; step++ {
		nx, ny := x+DX[dir]*step, y+DY[dir]*step
		if !g.InBounds(nx, ny) {
			return "wall", step
		}
		if g.Walls[Key(nx, ny)] {
			return "wall", step
		}
		if w := g.builtWallAt(nx, ny); w != nil {
			return "built_wall", step
		}
		if c := g.constructionAt(nx, ny); c != nil {
			return "construction", step
		}
		for _, b := range g.Bullets {
			if !b.Alive || b.X != nx || b.Y != ny {
				continue
			}
			if b.Owner == viewer.Owner {
				continue
			}
			return "bullet", step
		}
		if u := g.unitAt(nx, ny); u != nil {
			if u.Owner == viewer.Owner {
				if g.isMyFlagCell(nx, ny, viewer.Owner) {
					return "my_flag", step
				}
				return "ally", step
			}
			return "enemy", step
		}
		if g.Mode == ModeClassic {
			for _, f := range g.Flags {
				if f.X == nx && f.Y == ny {
					return "flag", step
				}
			}
		}
		if g.Mode == ModeCTF {
			for _, f := range g.Flags {
				if f.X == nx && f.Y == ny && f.Owner >= 0 && f.Owner != viewer.Owner {
					return "enemy_flag", step
				}
			}
		}
	}
}

// CellLabel is used by drone(): what look_forward would call a cell from a viewer.
func (g *Game) CellLabel(x, y, viewerOwner int) string {
	if !g.InBounds(x, y) {
		return "wall"
	}
	if g.Walls[Key(x, y)] {
		return "wall"
	}
	if w := g.builtWallAt(x, y); w != nil {
		return "built_wall"
	}
	if c := g.constructionAt(x, y); c != nil {
		return "construction"
	}
	for _, b := range g.Bullets {
		if b.Alive && b.X == x && b.Y == y && b.Owner != viewerOwner {
			return "bullet"
		}
	}
	if u := g.unitAt(x, y); u != nil {
		if u.Owner == viewerOwner {
			if g.isMyFlagCell(x, y, viewerOwner) {
				return "my_flag"
			}
			return "ally"
		}
		return "enemy"
	}
	if g.Mode == ModeClassic {
		for _, f := range g.Flags {
			if f.X == x && f.Y == y {
				return "flag"
			}
		}
	}
	if g.Mode == ModeCTF {
		for _, f := range g.Flags {
			if f.X == x && f.Y == y && f.Owner >= 0 && f.Owner != viewerOwner {
				return "enemy_flag"
			}
		}
	}
	return "empty"
}

func (g *Game) isMyFlagCell(x, y, owner int) bool {
	if g.Mode != ModeCTF {
		return false
	}
	for _, f := range g.Flags {
		if f.Owner == owner && f.X == x && f.Y == y {
			return true
		}
	}
	return false
}
