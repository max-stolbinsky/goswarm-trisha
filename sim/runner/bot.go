package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"

	"goswarm-trisha/sim/engine"
	"goswarm-trisha/sim/runner/api"
)

// Bot runs a yaegi robot for one player.
type Bot struct {
	Player int
	loop   func()
}

// LoadBotSource resolves bot name to source text.
func LoadBotSource(simRoot, name string) (string, error) {
	switch name {
	case "trisha":
		p := filepath.Join(simRoot, "..", "Robot.go")
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		return stripPackage(string(b)), nil
	default:
		p := filepath.Join(simRoot, "bots", name+".go.txt")
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}

func stripPackage(src string) string {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "package ") {
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// NewBot compiles bot source.
func NewBot(src string) (*Bot, error) {
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)
	i.Use(apiSymbols())

	wrapped := yaegiPreamble + src + `
func __robot_loop() { loop() }
`

	if _, err := i.Eval(wrapped); err != nil {
		return nil, fmt.Errorf("compile bot: %w", err)
	}

	v, err := i.Eval("main.__robot_loop")
	if err != nil {
		return nil, err
	}
	fn, ok := v.Interface().(func())
	if !ok {
		return nil, fmt.Errorf("loop is not func()")
	}
	return &Bot{loop: fn}, nil
}

// RunLoop executes loop() with CPU timing for the current unit.
func (b *Bot) RunLoop(g *engine.Game, u *engine.Unit) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	start := time.Now()
	api.WithUnit(g, u, b.loop)
	if g.EnableCPU {
		elapsed := float64(time.Since(start).Microseconds()) * g.CPUScale
		if elapsed < 0 {
			elapsed = 0
		}
		g.AddTickCPU(u.Owner, elapsed)
	}
	return nil
}

// AttachBots wires two bots into a game tick runner. bot0Side is 0 or 1.
func AttachBots(g *engine.Game, b0, b1 *Bot, bot0Side int) engine.UnitRunner {
	if bot0Side != 0 && bot0Side != 1 {
		bot0Side = 0
	}
	return func(player int, u *engine.Unit) error {
		var bot *Bot
		if player == bot0Side {
			bot = b0
		} else {
			bot = b1
		}
		if bot == nil {
			return nil
		}
		return bot.RunLoop(g, u)
	}
}
