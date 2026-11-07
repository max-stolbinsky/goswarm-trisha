package runner

// yaegiPreamble defines lowercase bot API names for interpreted robots.
const yaegiPreamble = `
package main

import (
	a "goswarm-trisha/sim/runner/api"
	"fmt"
)

func move_forward() bool { return a.MoveForward() }
func turn_left() bool { return a.TurnLeft() }
func turn_right() bool { return a.TurnRight() }
func shoot() bool { return a.Shoot() }
func build_wall() bool { return a.BuildWall() }
func build_warrior() bool { return a.BuildWarrior() }
func look_forward() (string, int) { return a.LookForward() }
func look_left() (string, int) { return a.LookLeft() }
func look_right() (string, int) { return a.LookRight() }
func get_pos() (int, int) { return a.GetPos() }
func get_dir() int { return a.GetDir() }
func get_grid_size() int { return a.GetGridSize() }
func my_flag() (int, int) { return a.MyFlag() }
func game_mode() string { return a.GameMode() }
func friendly_fire() bool { return a.FriendlyFire() }
func my_kind() string { return a.MyKind() }
func my_id() int { return a.MyID() }
func my_units() int { return a.MyUnits() }
func busy() int { return a.Busy() }
func has_module(name string) bool { return a.HasModule(name) }
func drone(x, y int) ([][]string, bool) { return a.Drone(x, y) }
func drone_ready() int { return a.DroneReady() }
func boost() bool { return a.Boost() }
func boost_ready() int { return a.BoostReady() }
func shielded() bool { return a.Shielded() }
func randInt(n int) int { return a.RandInt(n) }

var Println = fmt.Println

`
