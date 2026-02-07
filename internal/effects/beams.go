package effects

import (
	"fmt"
	"sort"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/terminal"
	"tte-go/internal/utils"
)

type BeamsConfig struct {
	BeamRowSymbols       []string
	BeamColumnSymbols    []string
	BeamDelay            int
	BeamRowSpeedRange    [2]int
	BeamColumnSpeedRange [2]int
	BeamGradientStops    []utils.Color
	BeamGradientSteps    []int
	BeamGradientFrames   int
	FinalGradientStops   []utils.Color
	FinalGradientSteps   []int
	FinalGradientFrames  int
	FinalGradientDir     utils.GradientDirection
	FinalWipeSpeed       int
}

type Beams struct {
	base             *BaseEffect
	config           BeamsConfig
	pendingGroups    []*beamsGroup
	activeGroups     []*beamsGroup
	activeCharacters map[*engine.EffectCharacter]struct{}
	finalWipeGroups  [][]*engine.EffectCharacter
	delay            int
	phase            string
	finalColorMap    map[*engine.EffectCharacter]*utils.ColorPair
}

type beamsGroup struct {
	characters           []*engine.EffectCharacter
	direction            string
	speed                float64
	nextCharacterCounter float64
	terminal             *engine.TerminalState
}

func NewBeams(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	beams := &Beams{
		base:             base,
		config:           defaultBeamsConfig(),
		activeCharacters: map[*engine.EffectCharacter]struct{}{},
		phase:            "beams",
		finalColorMap:    map[*engine.EffectCharacter]*utils.ColorPair{},
	}
	beams.finalWipeGroups = base.Terminal.GetCharactersGrouped(engine.DiagonalTopLeftToBottomRight, true, true, true, true)
	beams.build()
	return beams
}

func defaultBeamsConfig() BeamsConfig {
	return BeamsConfig{
		BeamRowSymbols:       []string{"▂", "▁", "_"},
		BeamColumnSymbols:    []string{"▌", "▍", "▎", "▏"},
		BeamDelay:            6,
		BeamRowSpeedRange:    [2]int{15, 60},
		BeamColumnSpeedRange: [2]int{9, 15},
		BeamGradientStops:    mustColors("ffffff", "00D1FF", "8A008A"),
		BeamGradientSteps:    []int{2, 6},
		BeamGradientFrames:   2,
		FinalGradientStops:   mustColors("8A008A", "00D1FF", "ffffff"),
		FinalGradientSteps:   []int{12},
		FinalGradientFrames:  4,
		FinalGradientDir:     utils.GradientVertical,
		FinalWipeSpeed:       3,
	}
}

func (b *Beams) build() {
	finalGradient, _ := utils.NewGradient(b.config.FinalGradientStops, b.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		b.base.Canvas.TextBottom,
		b.base.Canvas.TextTop,
		b.base.Canvas.TextLeft,
		b.base.Canvas.TextRight,
		b.config.FinalGradientDir,
	)

	preexistingColorsPresent := false
	for _, character := range b.base.Terminal.GetCharacters(true, true, true, true, engine.TopToBottomLeftToRight) {
		if character.InputColors != nil && (character.InputColors.FG != nil || character.InputColors.BG != nil) {
			preexistingColorsPresent = true
			break
		}
	}

	for _, character := range b.base.Terminal.GetCharacters(true, true, true, true, engine.TopToBottomLeftToRight) {
		b.base.Terminal.SetCharacterVisibility(character, false)
		if character.IsFillCharacter {
			black := utils.Color{R: 0, G: 0, B: 0}
			b.finalColorMap[character] = &utils.ColorPair{FG: &black}
			if !withinTextBounds(b.base.Canvas, character.InputCoord) {
				continue
			}
		}
		if b.base.Config.ExistingColorHandling == "dynamic" && preexistingColorsPresent {
			fg := utils.Color{R: 255, G: 255, B: 255}
			var bg *utils.Color
			if character.InputColors != nil && character.InputColors.FG != nil {
				fg = *character.InputColors.FG
			}
			if character.InputColors != nil && character.InputColors.BG != nil {
				bg = character.InputColors.BG
			}
			b.finalColorMap[character] = &utils.ColorPair{FG: &fg, BG: bg}
		} else {
			color := finalMapping[character.InputCoord]
			b.finalColorMap[character] = &utils.ColorPair{FG: &color}
		}
	}

	beamGradient, _ := utils.NewGradient(b.config.BeamGradientStops, b.config.BeamGradientSteps, false)
	groups := []*beamsGroup{}

	rows := b.base.Terminal.GetCharactersGrouped(engine.RowTopToBottom, true, true, true, true)
	for _, row := range rows {
		row = filterTextBound(row, b.base.Canvas)
		if len(row) > 0 {
			groups = append(groups, newBeamsGroup(row, "row", b.base.Terminal, b.config))
		}
	}
	cols := b.base.Terminal.GetCharactersGrouped(engine.ColumnLeftToRight, true, true, true, true)
	for _, col := range cols {
		col = filterTextBound(col, b.base.Canvas)
		if len(col) > 0 {
			groups = append(groups, newBeamsGroup(col, "column", b.base.Terminal, b.config))
		}
	}

	for _, group := range groups {
		for _, character := range group.characters {
			beamRowScene := character.Animation.NewScene("beam_row")
			beamColumnScene := character.Animation.NewScene("beam_column")
			brightenScene := character.Animation.NewScene("brighten")

			_ = beamRowScene.ApplyGradientToSymbols(b.config.BeamRowSymbols, b.config.BeamGradientFrames, beamGradient, nil)
			_ = beamColumnScene.ApplyGradientToSymbols(b.config.BeamColumnSymbols, b.config.BeamGradientFrames, beamGradient, nil)

			charColors := b.finalColorMap[character]
			var fgFade *utils.Gradient
			var fgBrighten *utils.Gradient
			var bgFade *utils.Gradient
			var bgBrighten *utils.Gradient
			if charColors != nil && charColors.FG != nil {
				faded := character.Animation.AdjustColorBrightness(*charColors.FG, 0.3)
				fgFade, _ = utils.NewGradient([]utils.Color{*charColors.FG, faded}, []int{10}, false)
				fgBrighten, _ = utils.NewGradient([]utils.Color{faded, *charColors.FG}, []int{10}, false)
			}
			if charColors != nil && charColors.BG != nil {
				faded := character.Animation.AdjustColorBrightness(*charColors.BG, 0.3)
				bgFade, _ = utils.NewGradient([]utils.Color{*charColors.BG, faded}, []int{10}, false)
				bgBrighten, _ = utils.NewGradient([]utils.Color{faded, *charColors.BG}, []int{10}, false)
			}
			_ = beamRowScene.ApplyGradientToSymbols([]string{character.Symbol}, 2, fgFade, bgFade)
			_ = beamColumnScene.ApplyGradientToSymbols([]string{character.Symbol}, 2, fgFade, bgFade)
			_ = brightenScene.ApplyGradientToSymbols([]string{character.Symbol}, b.config.FinalGradientFrames, fgBrighten, bgBrighten)
		}
	}

	b.pendingGroups = groups
	utils.Shuffle(len(b.pendingGroups), func(i, j int) { b.pendingGroups[i], b.pendingGroups[j] = b.pendingGroups[j], b.pendingGroups[i] })
}

func (b *Beams) Next() (string, bool) {
	if b.phase != "complete" || len(b.activeCharacters) > 0 {
		switch b.phase {
		case "beams":
			if b.delay == 0 {
				if len(b.pendingGroups) > 0 {
					for i := 0; i < utils.RandIntn(5)+1 && len(b.pendingGroups) > 0; i++ {
						b.activeGroups = append(b.activeGroups, b.pendingGroups[0])
						b.pendingGroups = b.pendingGroups[1:]
					}
				}
				b.delay = b.config.BeamDelay
			} else {
				b.delay--
			}
			for _, group := range b.activeGroups {
				group.incrementNextCharacterCounter()
				if int(group.nextCharacterCounter) > 1 {
					for i := 0; i < int(group.nextCharacterCounter); i++ {
						if !group.complete() {
							if nextChar := group.getNextCharacter(); nextChar != nil {
								b.activeCharacters[nextChar] = struct{}{}
							}
						}
					}
				}
			}
			remaining := []*beamsGroup{}
			for _, group := range b.activeGroups {
				if !group.complete() {
					remaining = append(remaining, group)
				}
			}
			b.activeGroups = remaining
			if len(b.pendingGroups) == 0 && len(b.activeGroups) == 0 && len(b.activeCharacters) == 0 {
				b.phase = "final_wipe"
			}
		case "final_wipe":
			if len(b.finalWipeGroups) > 0 {
				for i := 0; i < b.config.FinalWipeSpeed && len(b.finalWipeGroups) > 0; i++ {
					group := b.finalWipeGroups[0]
					b.finalWipeGroups = b.finalWipeGroups[1:]
					for _, character := range group {
						character.Animation.ActivateScene("brighten")
						b.base.Terminal.SetCharacterVisibility(character, true)
						b.activeCharacters[character] = struct{}{}
					}
				}
			} else {
				b.phase = "complete"
			}
		}

		b.update()
		return b.base.Terminal.GetFormattedOutputString(), true
	}
	return "", false
}

func (b *Beams) update() {
	for character := range b.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			delete(b.activeCharacters, character)
		}
	}
}

func newBeamsGroup(chars []*engine.EffectCharacter, direction string, terminal *engine.TerminalState, cfg BeamsConfig) *beamsGroup {
	group := &beamsGroup{
		characters: chars,
		direction:  direction,
		terminal:   terminal,
	}
	speedRange := cfg.BeamRowSpeedRange
	if direction == "column" {
		speedRange = cfg.BeamColumnSpeedRange
	}
	speed := utils.RandIntn(speedRange[1]-speedRange[0]+1) + speedRange[0]
	group.speed = float64(speed) * 0.1
	if direction == "row" {
		sort.Slice(group.characters, func(i, j int) bool {
			return group.characters[i].InputCoord.Col < group.characters[j].InputCoord.Col
		})
	} else {
		sort.Slice(group.characters, func(i, j int) bool {
			return group.characters[i].InputCoord.Row < group.characters[j].InputCoord.Row
		})
	}
	if utils.RandIntn(2) == 0 {
		reverseEffectCharacters(group.characters)
	}
	return group
}

func (g *beamsGroup) incrementNextCharacterCounter() {
	g.nextCharacterCounter += g.speed
}

func (g *beamsGroup) getNextCharacter() *engine.EffectCharacter {
	g.nextCharacterCounter -= 1
	next := g.characters[0]
	g.characters = g.characters[1:]
	if next.Animation.ActiveScene != nil {
		next.Animation.ActiveScene.ResetScene()
		return nil
	}
	g.terminal.SetCharacterVisibility(next, true)
	if g.direction == "row" {
		next.Animation.ActivateScene("beam_row")
	} else {
		next.Animation.ActivateScene("beam_column")
	}
	return next
}

func (g *beamsGroup) complete() bool {
	return len(g.characters) == 0
}

func mustColors(values ...string) []utils.Color {
	colors := make([]utils.Color, 0, len(values))
	for _, value := range values {
		color, err := utils.ParseHexColor(value)
		if err != nil {
			panic(fmt.Sprintf("invalid color: %s", value))
		}
		colors = append(colors, *color)
	}
	return colors
}

func reverseEffectCharacters(chars []*engine.EffectCharacter) {
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
}

func withinTextBounds(canvas *terminal.Canvas, coord utils.Coord) bool {
	if canvas == nil {
		return false
	}
	return coord.Col >= canvas.TextLeft && coord.Col <= canvas.TextRight && coord.Row >= canvas.TextBottom && coord.Row <= canvas.TextTop
}

func filterTextBound(chars []*engine.EffectCharacter, canvas *terminal.Canvas) []*engine.EffectCharacter {
	filtered := make([]*engine.EffectCharacter, 0, len(chars))
	for _, character := range chars {
		if withinTextBounds(canvas, character.InputCoord) {
			filtered = append(filtered, character)
		}
	}
	return filtered
}

func (b *Beams) CanvasHeight() int {
	return b.base.CanvasHeight()
}

func (b *Beams) CanvasWidth() int {
	return b.base.CanvasWidth()
}
