package api

import (
	"sync"

	"goswarm-trisha/sim/engine"
)

var (
	mu sync.Mutex
	g  *engine.Game
	u  *engine.Unit
)

// Bind sets the game and running unit for API calls.
func Bind(game *engine.Game, unit *engine.Unit) {
	mu.Lock()
	g, u = game, unit
}

// Unbind releases the API lock after a unit loop.
func Unbind() {
	g, u = nil, nil
	mu.Unlock()
}

// WithUnit runs fn with game/unit bound (serialized for parallel sims).
func WithUnit(game *engine.Game, unit *engine.Unit, fn func()) {
	Bind(game, unit)
	defer Unbind()
	fn()
}

func move_forward() bool {
	if g == nil || u == nil {
		return false
	}
	return g.MoveForward(u)
}

func turn_left() bool {
	if g == nil || u == nil {
		return false
	}
	return g.TurnLeft(u)
}

func turn_right() bool {
	if g == nil || u == nil {
		return false
	}
	return g.TurnRight(u)
}

func shoot() bool {
	if g == nil || u == nil {
		return false
	}
	return g.Shoot(u)
}

func build_wall() bool {
	if g == nil || u == nil {
		return false
	}
	return g.BuildWall(u)
}

func build_warrior() bool {
	if g == nil || u == nil {
		return false
	}
	return g.BuildWarrior(u)
}

func look_forward() (string, int) {
	if g == nil || u == nil {
		return "", 0
	}
	return g.LookForward(u)
}

func look_left() (string, int) {
	if g == nil || u == nil {
		return "", 0
	}
	return g.LookSide(u, true)
}

func look_right() (string, int) {
	if g == nil || u == nil {
		return "", 0
	}
	return g.LookSide(u, false)
}

func get_pos() (int, int) {
	if u == nil {
		return 0, 0
	}
	return u.X, u.Y
}

func get_dir() int {
	if u == nil {
		return 0
	}
	return u.Dir
}

func get_grid_size() int {
	if g == nil {
		return 40
	}
	return g.Grid
}

func my_flag() (int, int) {
	if g == nil || u == nil {
		return -1, -1
	}
	return g.MyFlag(u.Owner)
}

func game_mode() string {
	if g == nil {
		return "classic"
	}
	return string(g.Mode)
}

func friendly_fire() bool {
	if g == nil {
		return false
	}
	return g.FriendlyFire
}

func my_kind() string {
	if u == nil {
		return "warrior"
	}
	return u.Kind.String()
}

func my_id() int {
	if u == nil {
		return 0
	}
	return u.ID
}

func my_units() int {
	if g == nil || u == nil {
		return 0
	}
	return g.MyUnits(u.Owner)
}

func busy() int {
	if g == nil || u == nil {
		return 0
	}
	return g.Busy(u)
}

func has_module(name string) bool {
	if g == nil || u == nil {
		return false
	}
	p := g.Players[u.Owner]
	switch name {
	case "big_gun":
		return p.HasBigGun
	case "drone":
		return p.HasDrone
	case "shield":
		return p.HasShield
	case "radar":
		return p.HasRadar
	case "fast_builder":
		return p.HasFastBuild
	case "boost":
		return p.HasBoost
	default:
		return false
	}
}

func drone(x, y int) ([][]string, bool) {
	if g == nil || u == nil {
		return nil, false
	}
	return g.Drone(u.Owner, x, y)
}

func drone_ready() int {
	if g == nil || u == nil {
		return -1
	}
	return g.DroneReady(u.Owner)
}

func boost() bool {
	if g == nil || u == nil {
		return false
	}
	return g.Boost(u)
}

func boost_ready() int {
	if g == nil || u == nil {
		return -1
	}
	return g.BoostReady(u)
}

func shielded() bool {
	if u == nil {
		return false
	}
	return u.Shielded
}

func randInt(n int) int {
	if g == nil || u == nil {
		return 0
	}
	return g.RandInt(u.Owner, n)
}
