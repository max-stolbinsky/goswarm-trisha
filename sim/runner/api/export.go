package api

// Exported aliases for yaegi (bots call lowercase via import wrapper).

func MoveForward() bool  { return move_forward() }
func TurnLeft() bool     { return turn_left() }
func TurnRight() bool    { return turn_right() }
func Shoot() bool        { return shoot() }
func BuildWall() bool    { return build_wall() }
func BuildWarrior() bool { return build_warrior() }
func LookForward() (string, int) {
	return look_forward()
}
func LookLeft() (string, int)  { return look_left() }
func LookRight() (string, int) { return look_right() }
func GetPos() (int, int)       { return get_pos() }
func GetDir() int              { return get_dir() }
func GetGridSize() int         { return get_grid_size() }
func MyFlag() (int, int)       { return my_flag() }
func GameMode() string         { return game_mode() }
func FriendlyFire() bool       { return friendly_fire() }
func MyKind() string           { return my_kind() }
func MyID() int                { return my_id() }
func MyUnits() int             { return my_units() }
func Busy() int                { return busy() }
func HasModule(name string) bool {
	return has_module(name)
}
func Drone(x, y int) ([][]string, bool) { return drone(x, y) }
func DroneReady() int                   { return drone_ready() }
func Boost() bool                       { return boost() }
func BoostReady() int                   { return boost_ready() }
func Shielded() bool                    { return shielded() }
func RandInt(n int) int                 { return randInt(n) }
