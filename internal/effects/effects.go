package effects

import (
	"fmt"
	"sort"

	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type EffectFactory func(input string, cfg config.TerminalConfig) engine.Effect

type EffectSpec struct {
	Name        string
	Description string
	Factory     EffectFactory
}

var registry = map[string]EffectSpec{}

func Register(spec EffectSpec) {
	registry[spec.Name] = spec
}

func registerAlias(name, description string, factory EffectFactory) {
	Register(EffectSpec{
		Name:        name,
		Description: description,
		Factory:     factory,
	})
}

func Get(name string) (EffectSpec, bool) {
	spec, ok := registry[name]
	return spec, ok
}

func EffectNames() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func PrintEffectHelp(name string) {
	spec, ok := registry[name]
	if !ok {
		fmt.Printf("Unknown effect: %s\n", name)
		return
	}
	fmt.Printf("%s\n\n%s\n\n", spec.Name, spec.Description)
	fmt.Println("No effect-specific options are implemented yet in the Go port.")
}

func init() {
	registerAlias("beams", "Create beams which travel over the canvas illuminating characters behind them.", NewBeams)
	registerAlias("binarypath", "Binary representations of each character move towards the home coordinate of the character.", NewBinaryPath)
	registerAlias("blackhole", "Characters are consumed by a black hole and explode outwards.", NewBlackhole)
	registerAlias("bouncyballs", "Characters are bouncy balls falling from the top of the canvas.", NewBouncyBalls)
	registerAlias("bubbles", "Characters are formed into bubbles that float down and pop.", NewBubbles)
	registerAlias("burn", "Burns vertically in the canvas.", NewBurn)
	registerAlias("colorshift", "Display a gradient that shifts colors across the terminal.", NewColorShift)
	registerAlias("crumble", "Characters lose color and crumble into dust, vacuumed up, and reformed.", NewCrumble)
	registerAlias("decrypt", "Display a movie style decryption effect.", NewDecrypt)
	registerAlias("errorcorrect", "Some characters start in the wrong position and are corrected in sequence.", NewErrorCorrect)
	Register(EffectSpec{
		Name:        "expand",
		Description: "Expands the text from a single point.",
		Factory:     NewExpand,
	})
	registerAlias("fireworks", "Characters launch and explode like fireworks and fall into place.", NewFireworks)
	registerAlias("highlight", "Run a specular highlight across the text.", NewHighlight)
	registerAlias("laseretch", "A laser etches characters onto the terminal.", NewLaserEtch)
	registerAlias("matrix", "Matrix digital rain effect.", NewMatrix)
	Register(EffectSpec{
		Name:        "middleout",
		Description: "Text expands in a single row or column in the middle of the canvas then out.",
		Factory:     NewMiddleOut,
	})
	registerAlias("orbittingvolley", "Four launchers orbit the canvas firing volleys of characters inward.", NewOrbittingVolley)
	registerAlias("overflow", "Input text overflows and scrolls the terminal until ordered.", NewOverflow)
	registerAlias("pour", "Pours the characters into position from the given direction.", NewPour)
	Register(EffectSpec{
		Name:        "print",
		Description: "Lines are printed one at a time following a print head.",
		Factory:     NewPrint,
	})
	Register(EffectSpec{
		Name:        "rain",
		Description: "Rain characters from the top of the canvas.",
		Factory:     NewRain,
	})
	Register(EffectSpec{
		Name:        "randomsequence",
		Description: "Prints the input data in a random sequence.",
		Factory:     NewRandomSequence,
	})
	registerAlias("rings", "Characters are dispersed and form into spinning rings.", NewRings)
	Register(EffectSpec{
		Name:        "scattered",
		Description: "Text is scattered across the canvas and moves into position.",
		Factory:     NewScattered,
	})
	Register(EffectSpec{
		Name:        "slice",
		Description: "Slices the input in half and slides it into place from opposite directions.",
		Factory:     NewSlice,
	})
	Register(EffectSpec{
		Name:        "slide",
		Description: "Slide characters into view from outside the terminal.",
		Factory:     NewSlide,
	})
	registerAlias("smoke", "Smoke floods the canvas colorizing characters it crosses.", NewSmoke)
	registerAlias("spotlights", "Spotlights search the text area, illuminating characters, then converge.", NewSpotlights)
	registerAlias("spray", "Draws characters spawning at varying rates from a single point.", NewSpray)
	Register(EffectSpec{
		Name:        "swarm",
		Description: "Characters are grouped into swarms and move around before settling.",
		Factory:     NewSwarm,
	})
	Register(EffectSpec{
		Name:        "sweep",
		Description: "Sweep across the canvas to reveal uncolored text.",
		Factory:     NewSweep,
	})
	registerAlias("synthgrid", "Create a grid which fills with characters dissolving into final text.", NewSynthGrid)
	registerAlias("thunderstorm", "Create a thunderstorm in the terminal.", NewThunderstorm)
	registerAlias("unstable", "Spawn characters jumbled, explode, then reassemble.", NewUnstable)
	registerAlias("vhstape", "Lines of characters glitch left and right like an old VHS tape.", NewVHSTape)
	registerAlias("waves", "Waves travel across the terminal leaving characters behind.", NewWaves)
	Register(EffectSpec{
		Name:        "wipe",
		Description: "Wipes the text across the terminal to reveal characters.",
		Factory:     NewWipe,
	})
}
