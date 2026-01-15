package cli

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"tte-go/internal/config"
	"tte-go/internal/effects"
	"tte-go/internal/engine"
	"tte-go/internal/terminal"
	"tte-go/internal/utils"
)

const version = "dev"

var (
	errNoInput      = errors.New("no input provided")
	errNoEffect     = errors.New("no effect specified")
	errNoCandidates = errors.New("no effects available for random selection")
)

func Run(args []string) error {
	cfg := config.DefaultTerminalConfig()
	fs := flag.NewFlagSet("tte", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)

	inputFile := fs.String("input-file", "", "File to read input from")
	fs.StringVar(inputFile, "i", "", "File to read input from (shorthand)")
	showVersion := fs.Bool("version", false, "Show program version")
	fs.BoolVar(showVersion, "v", false, "Show program version (shorthand)")

	randomEffect := fs.Bool("random-effect", false, "Randomly select an effect to apply")
	fs.BoolVar(randomEffect, "R", false, "Randomly select an effect to apply (shorthand)")

	includeEffects := &utils.StringSlice{}
	excludeEffects := &utils.StringSlice{}
	fs.Var(includeEffects, "include-effects", "Space-separated list of effects to include when randomly selecting an effect")
	fs.Var(excludeEffects, "exclude-effects", "Space-separated list of effects to exclude when randomly selecting an effect")

	cfg.BindFlags(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}
	effectArgs := fs.Args()
	if *showVersion {
		fmt.Printf("TerminalTextEffects (Go) %s\n", version)
		return nil
	}

	availableEffects := effects.EffectNames()
	effectName := ""
	if *randomEffect {
		candidates := filterEffects(availableEffects, includeEffects.Values(), excludeEffects.Values())
		if len(candidates) == 0 {
			return errNoCandidates
		}
		rand.New(rand.NewSource(time.Now().UnixNano()))
		effectName = candidates[rand.Intn(len(candidates))]
	} else if len(effectArgs) > 0 {
		effectName = effectArgs[0]
		effectArgs = effectArgs[1:]
	} else {
		return errNoEffect
	}

	if effectName == "" {
		return errNoEffect
	}

	if hasHelpFlag(effectArgs) {
		effects.PrintEffectHelp(effectName)
		return nil
	}

	inputData, err := utils.ReadInput(*inputFile)
	if err != nil {
		return err
	}
	if strings.TrimSpace(inputData) == "" {
		return errNoInput
	}

	effectSpec, ok := effects.Get(effectName)
	if !ok {
		return fmt.Errorf("unknown effect: %s", effectName)
	}

	term := terminal.New(cfg)
	eng := engine.New(term)
	return eng.Run(effectSpec.Factory(inputData, cfg))
}

func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	nonFlagIndex := findFirstNonFlag(args)
	parseArgs := args
	if nonFlagIndex >= 0 {
		parseArgs = args[:nonFlagIndex]
	}
	if err := fs.Parse(parseArgs); err != nil {
		return nil, err
	}
	if nonFlagIndex >= 0 {
		return args[nonFlagIndex:], nil
	}
	return []string{}, nil
}

func findFirstNonFlag(args []string) int {
	for i, arg := range args {
		if arg == "--" {
			if i+1 < len(args) {
				return i + 1
			}
			return -1
		}
		if !strings.HasPrefix(arg, "-") {
			return i
		}
	}
	return -1
}

func filterEffects(all []string, include []string, exclude []string) []string {
	if len(include) == 0 && len(exclude) == 0 {
		return all
	}

	set := make(map[string]struct{}, len(all))
	for _, name := range all {
		set[name] = struct{}{}
	}

	if len(include) > 0 {
		filtered := []string{}
		for _, name := range include {
			if _, ok := set[name]; ok {
				filtered = append(filtered, name)
			}
		}
		return filtered
	}

	filtered := []string{}
	excluded := make(map[string]struct{}, len(exclude))
	for _, name := range exclude {
		excluded[name] = struct{}{}
	}
	for _, name := range all {
		if _, ok := excluded[name]; !ok {
			filtered = append(filtered, name)
		}
	}
	return filtered
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func PrintError(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
}
