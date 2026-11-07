package runner

import (
	"fmt"
	"reflect"

	"goswarm-trisha/sim/runner/api"
)

func apiSymbols() map[string]map[string]reflect.Value {
	return map[string]map[string]reflect.Value{
		"goswarm-trisha/sim/runner/api/api": {
			"MoveForward":  reflect.ValueOf(api.MoveForward),
			"TurnLeft":     reflect.ValueOf(api.TurnLeft),
			"TurnRight":    reflect.ValueOf(api.TurnRight),
			"Shoot":        reflect.ValueOf(api.Shoot),
			"BuildWall":    reflect.ValueOf(api.BuildWall),
			"BuildWarrior": reflect.ValueOf(api.BuildWarrior),
			"LookForward":  reflect.ValueOf(api.LookForward),
			"LookLeft":     reflect.ValueOf(api.LookLeft),
			"LookRight":    reflect.ValueOf(api.LookRight),
			"GetPos":       reflect.ValueOf(api.GetPos),
			"GetDir":       reflect.ValueOf(api.GetDir),
			"GetGridSize":  reflect.ValueOf(api.GetGridSize),
			"MyFlag":       reflect.ValueOf(api.MyFlag),
			"GameMode":     reflect.ValueOf(api.GameMode),
			"FriendlyFire": reflect.ValueOf(api.FriendlyFire),
			"MyKind":       reflect.ValueOf(api.MyKind),
			"MyID":         reflect.ValueOf(api.MyID),
			"MyUnits":      reflect.ValueOf(api.MyUnits),
			"Busy":         reflect.ValueOf(api.Busy),
			"HasModule":    reflect.ValueOf(api.HasModule),
			"Drone":        reflect.ValueOf(api.Drone),
			"DroneReady":   reflect.ValueOf(api.DroneReady),
			"Boost":        reflect.ValueOf(api.Boost),
			"BoostReady":   reflect.ValueOf(api.BoostReady),
			"Shielded":     reflect.ValueOf(api.Shielded),
			"RandInt":      reflect.ValueOf(api.RandInt),
		},
		"fmt/fmt": {
			"Println": reflect.ValueOf(fmt.Println),
		},
	}
}
