package engine

const (
	Up = iota
	Right
	Down
	Left
)

var DX = [4]int{0, 1, 0, -1}
var DY = [4]int{-1, 0, 1, 0}

type Kind int

const (
	KindTank Kind = iota
	KindBuilder
	KindWarrior
)

func (k Kind) String() string {
	switch k {
	case KindTank:
		return "tank"
	case KindBuilder:
		return "builder"
	default:
		return "warrior"
	}
}

type Mode string

const (
	ModeClassic    Mode = "classic"
	ModeDeathmatch Mode = "deathmatch"
	ModeCTF        Mode = "ctf"
)

type Outcome string

const (
	Win0 Outcome = "win0"
	Win1 Outcome = "win1"
	Draw Outcome = "draw"
)

type Unit struct {
	ID, Owner, X, Y, Dir int
	Kind                 Kind
	Alive                bool
	Busy                 int
	BoostCD              int
	Shielded             bool
	Acted                bool
}

type Bullet struct {
	X, Y, Dir, Owner int
	Alive            bool
}

type BuiltWall struct {
	X, Y, HP, Owner int
}

type Construction struct {
	X, Y, Owner, BuilderID, Left int
	Kind                         Kind // wall=builder sentinel? use KindBuilder for wall, KindWarrior for warrior
	IsWall                       bool
	Dir                          int
}

type Flag struct {
	X, Y, Owner int // Owner -1 for classic shared flag
}

type PlayerState struct {
	Stopped      bool
	TickCPU      float64 // ms this tick
	TotalCPU     float64 // ms whole game
	StopTick     int
	StopReason   string
	HasBoost     bool
	HasShield    bool
	HasRadar     bool
	HasDrone     bool
	HasBigGun    bool
	HasFastBuild bool
	DroneCD      int
	WallCount    int
}

type Game struct {
	Mode          Mode
	Grid          int
	Seed          int
	FriendlyFire  bool
	MaxSteps      int
	Tick          int
	Walls         map[int]bool // key x*1000+y map walls
	Units         []*Unit
	Bullets       []*Bullet
	Built         []*BuiltWall
	Constructions []*Construction
	Flags         []Flag // classic: one with Owner=-1; ctf: two
	Players       [2]PlayerState
	NextUnitID    int
	Wrecks        map[int]int // key -> ticks left
	FirstPlayer   int         // who acts first this tick
	RNG           *RNG
	Message       string
	Outcome       Outcome
	Done          bool

	// CPU model
	CPUScale       float64 // local_ns * scale -> server ms
	TickLimitMS    float64
	GameLimitMS    float64
	EnableCPU      bool
	Current        *Unit
	Log            []string
}

func Key(x, y int) int { return x*1000 + y }

type RNG struct {
	state uint64
}

func NewRNG(seed int) *RNG {
	s := uint64(seed)
	if s == 0 {
		s = 1
	}
	return &RNG{state: s}
}

// Simple LCG; not identical to server, used only for bot randInt when map is fixed.
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	r.state = r.state*6364136223846793005 + 1
	return int((r.state >> 33) % uint64(n))
}
