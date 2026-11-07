package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"

	"goswarm-trisha/sim/engine"
	"goswarm-trisha/sim/runner"
)

func main() {
	mode := flag.String("mode", "deathmatch", "classic|deathmatch|ctf")
	bot0 := flag.String("bot0", "trisha-v1", "bot for side 0")
	bot1 := flag.String("bot1", "starter", "bot for side 1")
	limit := flag.Int("limit", 0, "max seeds (0 = all)")
	cpu := flag.Bool("cpu", false, "enforce CPU limits")
	scale := flag.Float64("scale", 0.001, "CPU scale")
	par := flag.Int("parallel", 8, "parallel fights")
	flag.Parse()

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

	seeds := make([]string, 0, len(pm))
	for s := range pm {
		seeds = append(seeds, s)
	}
	if *limit > 0 && *limit < len(seeds) {
		seeds = seeds[:*limit]
	}

	var w0, w1, draws, timeouts int64
	sem := make(chan struct{}, *par)
	var wg sync.WaitGroup

	for _, seedStr := range seeds {
		seedStr := seedStr
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			pv := pm[seedStr]
			seed, _ := strconv.Atoi(seedStr)
			mods0 := engine.PlayerState{}
			if *bot0 == "trisha" {
				mods0 = engine.PlayerState{HasDrone: true, HasShield: true, HasRadar: true}
			}
			g, err := engine.NewGameFromPreview(pv, seed, false, mods0, engine.PlayerState{})
			if err != nil {
				return
			}
			g.EnableCPU = *cpu
			g.CPUScale = *scale

			b0, err := runner.NewBot(src0)
			if err != nil {
				return
			}
			b1, err := runner.NewBot(src1)
			if err != nil {
				return
			}
			g.Play(runner.AttachBots(g, b0, b1, 0))

			switch g.Outcome {
			case engine.Win0:
				atomic.AddInt64(&w0, 1)
			case engine.Win1:
				atomic.AddInt64(&w1, 1)
			default:
				atomic.AddInt64(&draws, 1)
			}
			if g.Players[0].Stopped || g.Players[1].Stopped {
				atomic.AddInt64(&timeouts, 1)
			}
		}()
	}
	wg.Wait()

	fmt.Printf("mode=%s bot0=%s bot1=%s games=%d\n", *mode, *bot0, *bot1, len(seeds))
	fmt.Printf("W/L/D for bot0: %d / %d / %d\n", w0, w1, draws)
	fmt.Printf("timeout_or_stop: %d (%.1f%%)\n", timeouts, 100*float64(timeouts)/float64(len(seeds)))
}
