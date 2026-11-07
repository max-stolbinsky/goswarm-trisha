package engine

import (
	"encoding/json"
	"fmt"
	"os"
)

type PreviewUnit struct {
	ID    int    `json:"id"`
	Owner int    `json:"owner"`
	Kind  string `json:"kind"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Dir   int    `json:"dir"`
	Alive bool   `json:"alive"`
}

type PreviewFlag struct {
	Owner int `json:"owner"`
	X     int `json:"x"`
	Y     int `json:"y"`
}

type PreviewWall struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Preview struct {
	Mode     string         `json:"mode"`
	GridSize int            `json:"grid_size"`
	Units    []PreviewUnit  `json:"units"`
	Walls    []PreviewWall  `json:"walls"`
	Flag     *PreviewFlag   `json:"flag"`
	Flags    []PreviewFlag  `json:"flags"`
}

// LoadPreviews reads sim/maps/previews.json: map[mode]map[seed]Preview
func LoadPreviews(path string) (map[string]map[string]Preview, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw map[string]map[string]Preview
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func NewGameFromPreview(p Preview, seed int, ff bool, modules0, modules1 PlayerState) (*Game, error) {
	g := &Game{
		Mode:         Mode(p.Mode),
		Grid:         p.GridSize,
		Seed:         seed,
		FriendlyFire: ff,
		MaxSteps:     3000,
		Walls:        map[int]bool{},
		Wrecks:       map[int]int{},
		RNG:          NewRNG(seed),
		Players:      [2]PlayerState{modules0, modules1},
		TickLimitMS:  50,
		GameLimitMS:  2500,
		CPUScale:     1e-6, // placeholder; calibrated later
		EnableCPU:    true,
	}
	if g.Grid == 0 {
		g.Grid = 40
	}
	for _, w := range p.Walls {
		g.Walls[Key(w.X, w.Y)] = true
	}
	for _, u := range p.Units {
		kind := KindWarrior
		switch u.Kind {
		case "tank":
			kind = KindTank
		case "builder":
			kind = KindBuilder
		case "warrior":
			kind = KindWarrior
		default:
			return nil, fmt.Errorf("unknown kind %q", u.Kind)
		}
		nu := &Unit{
			ID: u.ID, Owner: u.Owner, X: u.X, Y: u.Y, Dir: u.Dir,
			Kind: kind, Alive: true,
		}
		if modules0.HasShield && u.Owner == 0 {
			nu.Shielded = true
		}
		if modules1.HasShield && u.Owner == 1 {
			nu.Shielded = true
		}
		g.Units = append(g.Units, nu)
		if u.ID >= g.NextUnitID {
			g.NextUnitID = u.ID + 1
		}
	}
	if p.Flag != nil {
		g.Flags = append(g.Flags, Flag{X: p.Flag.X, Y: p.Flag.Y, Owner: -1})
	}
	for _, f := range p.Flags {
		g.Flags = append(g.Flags, Flag{X: f.X, Y: f.Y, Owner: f.Owner})
	}
	return g, nil
}
