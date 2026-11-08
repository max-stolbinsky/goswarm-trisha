package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"goswarm-trisha/sim/engine"
	"goswarm-trisha/sim/runner"
)

func main() {
	mode := flag.String("mode", "deathmatch", "classic|deathmatch|ctf")
	seedStr := flag.String("seed", "138234", "map seed")
	bot0 := flag.String("bot0", "trisha-v1", "bot for side 0")
	bot1 := flag.String("bot1", "starter", "bot for side 1")
	cpu := flag.Bool("cpu", true, "enforce CPU limits")
	scale := flag.Float64("scale", 0.001, "CPU scale (µs to server ms)")
	side := flag.Int("side", 0, "which side bot0 plays (0 or 1)")
	flag.Parse()

	seed, _ := strconv.Atoi(*seedStr)
	simRoot, _ := os.Getwd()
	previews, err := engine.LoadPreviews(filepath.Join(simRoot, "maps", "previews.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "load previews: %v\n", err)
		os.Exit(1)
	}
	pm, ok := previews[*mode]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown mode %q\n", *mode)
		os.Exit(1)
	}
	pv, ok := pm[*seedStr]
	if !ok {
		fmt.Fprintf(os.Stderr, "unknown seed %q for mode %q\n", *seedStr, *mode)
		os.Exit(1)
	}

	mods0 := engine.PlayerState{}
	if *bot0 == "trisha" {
		mods0 = engine.PlayerState{HasDrone: true, HasShield: true, HasRadar: true, HasBoost: true}
	}
	g, err := engine.NewGameFromPreview(pv, seed, false, mods0, engine.PlayerState{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "new game: %v\n", err)
		os.Exit(1)
	}
	g.EnableCPU = *cpu
	g.CPUScale = *scale

	src0, err := runner.LoadBotSource(simRoot, *bot0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bot0: %v\n", err)
		os.Exit(1)
	}
	src1, err := runner.LoadBotSource(simRoot, *bot1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bot1: %v\n", err)
		os.Exit(1)
	}
	b0, err := runner.NewBot(src0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile bot0: %v\n", err)
		os.Exit(1)
	}
	b1, err := runner.NewBot(src1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "compile bot1: %v\n", err)
		os.Exit(1)
	}

	run := runner.AttachBots(g, b0, b1, *side)
	g.Play(run)

	fmt.Printf("outcome=%s message=%q steps=%d\n", g.Outcome, g.Message, g.Tick)
	fmt.Printf("stop0: tick=%d reason=%q total_cpu=%.3f\n", g.Players[0].StopTick, g.Players[0].StopReason, g.Players[0].TotalCPU)
	fmt.Printf("stop1: tick=%d reason=%q total_cpu=%.3f\n", g.Players[1].StopTick, g.Players[1].StopReason, g.Players[1].TotalCPU)
}
