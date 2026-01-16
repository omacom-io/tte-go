package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type PrintConfig struct {
	PrintHeadReturnSpeed float64
	PrintSpeed           int
	PrintHeadEasing      utils.EasingFunction
	FinalGradientStops   []utils.Color
	FinalGradientSteps   []int
	FinalGradientDir     utils.GradientDirection
}

type Print struct {
	base             *BaseEffect
	config           PrintConfig
	pendingRows      []*printRow
	processedRows    []*printRow
	currentRow       *printRow
	typingHead       *engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}
	typing           bool
	lastColumn       int
	finalColors      map[*engine.EffectCharacter]utils.Color
}

type printRow struct {
	untyped []*engine.EffectCharacter
	typed   []*engine.EffectCharacter
}

func NewPrint(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	print := &Print{
		base:             base,
		config:           defaultPrintConfig(),
		activeCharacters: map[*engine.EffectCharacter]struct{}{},
		finalColors:      map[*engine.EffectCharacter]utils.Color{},
	}
	print.build()
	return print
}

func defaultPrintConfig() PrintConfig {
	return PrintConfig{
		PrintHeadReturnSpeed: 1.5,
		PrintSpeed:           2,
		PrintHeadEasing:      utils.InOutQuad,
		FinalGradientStops:   mustColors("02b8bd", "c1f0e3", "00ffa0"),
		FinalGradientSteps:   []int{12},
		FinalGradientDir:     utils.GradientDiagonal,
	}
}

func (p *Print) build() {
	finalGradient, _ := utils.NewGradient(p.config.FinalGradientStops, p.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		p.base.Canvas.TextBottom,
		p.base.Canvas.TextTop,
		p.base.Canvas.TextLeft,
		p.base.Canvas.TextRight,
		p.config.FinalGradientDir,
	)
	for _, character := range p.base.Terminal.GetCharacters(true, true, true, false, engine.TopToBottomLeftToRight) {
		color, ok := finalMapping[character.InputCoord]
		if !ok {
			color = utils.Color{R: 255, G: 255, B: 255}
		}
		p.finalColors[character] = color
	}

	rows := p.base.Terminal.GetCharactersGrouped(engine.RowTopToBottom, true, true, true, false)
	for _, row := range rows {
		p.pendingRows = append(p.pendingRows, newPrintRow(row, p.finalColors, utils.Color{R: 255, G: 255, B: 255}))
	}
	if len(p.pendingRows) == 0 {
		p.typing = false
		return
	}
	p.currentRow = p.pendingRows[0]
	p.pendingRows = p.pendingRows[1:]
	p.typing = true
	p.lastColumn = 0
	p.typingHead = newTypingHead(p.base, utils.Coord{Row: 1, Col: 1})
	p.base.Terminal.AddCharacter(p.typingHead)
	p.base.Terminal.SetCharacterVisibility(p.typingHead, false)
}

func (p *Print) Next() (string, bool) {
	if len(p.activeCharacters) == 0 && !p.typing {
		return "", false
	}
	if p.typingHead != nil && p.typingHead.Motion != nil && p.typingHead.Motion.ActivePath != nil {
		// wait for carriage return
	} else if p.currentRow != nil && len(p.currentRow.untyped) > 0 {
		count := p.config.PrintSpeed
		if count > len(p.currentRow.untyped) {
			count = len(p.currentRow.untyped)
		}
		for i := 0; i < count; i++ {
			if next := p.currentRow.typeChar(); next != nil {
				p.base.Terminal.SetCharacterVisibility(next, true)
				p.activeCharacters[next] = struct{}{}
				p.lastColumn = next.InputCoord.Col
			}
		}
	} else {
		p.processedRows = append(p.processedRows, p.currentRow)
		if len(p.pendingRows) > 0 {
			for _, row := range p.processedRows {
				row.moveUp()
			}
			p.currentRow = p.pendingRows[0]
			p.pendingRows = p.pendingRows[1:]
			if !allFill(p.processedRows[len(p.processedRows)-1].typed) && !allFill(p.currentRow.untyped) {
				leftExtent := leftMostNonFill(p.currentRow.untyped)
				filtered := []*engine.EffectCharacter{}
				for _, char := range p.currentRow.untyped {
					if char.InputCoord.Col >= leftExtent && char.InputCoord.Col <= p.base.Canvas.TextRight {
						filtered = append(filtered, char)
					}
				}
				p.currentRow.untyped = filtered
			}
			p.typingHead.Motion.SetCoordinate(utils.Coord{Row: 1, Col: p.lastColumn})
			p.base.Terminal.SetCharacterVisibility(p.typingHead, true)
			p.typingHead.Motion.Paths = map[string]*engine.Path{}
			path, _ := p.typingHead.Motion.NewPath(p.config.PrintHeadReturnSpeed, p.config.PrintHeadEasing, nil, 0, false, "carriage_return_path")
			path.AddWaypoint(utils.Coord{Row: 1, Col: p.currentRow.untyped[0].InputCoord.Col})
			p.typingHead.Motion.ActivatePath(path)
			_ = p.typingHead.EventHandler.RegisterEvent(engine.EventPathComplete, path, engine.ActionCallback, engine.Callback{
				Fn: func(_ *engine.EffectCharacter, args ...any) {
					p.base.Terminal.SetCharacterVisibility(p.typingHead, false)
				},
			})
			p.activeCharacters[p.typingHead] = struct{}{}
		} else {
			p.typing = false
		}
	}

	for character := range p.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			delete(p.activeCharacters, character)
		}
	}
	return p.base.Terminal.GetFormattedOutputString(), true
}

func newPrintRow(chars []*engine.EffectCharacter, colorMap map[*engine.EffectCharacter]utils.Color, typingHeadColor utils.Color) *printRow {
	row := &printRow{}
	if len(chars) == 0 {
		return row
	}
	allSpaces := true
	for _, character := range chars {
		if character.Symbol != " " {
			allSpaces = false
			break
		}
	}
	if allSpaces {
		chars = chars[:1]
	} else {
		rightExtent := 0
		for _, character := range chars {
			if !character.IsFillCharacter && character.InputCoord.Col > rightExtent {
				rightExtent = character.InputCoord.Col
			}
		}
		filtered := []*engine.EffectCharacter{}
		for _, character := range chars {
			if character.InputCoord.Col <= rightExtent {
				filtered = append(filtered, character)
			}
		}
		chars = filtered
	}
	for _, character := range chars {
		character.Motion.SetCoordinate(utils.Coord{Row: 1, Col: character.InputCoord.Col})
		colorGradient, _ := utils.NewGradient([]utils.Color{typingHeadColor, colorMap[character]}, []int{5}, false)
		scene := character.Animation.NewScene("typed")
		_ = scene.ApplyGradientToSymbols([]string{"█", "▓", "▒", "░", character.Symbol}, 3, colorGradient, nil)
		character.Animation.ActivateScene("typed")
		row.untyped = append(row.untyped, character)
	}
	return row
}

func (r *printRow) moveUp() {
	for _, character := range r.typed {
		current := character.Motion.CurrentCoord
		character.Motion.SetCoordinate(utils.Coord{Row: current.Row + 1, Col: current.Col})
	}
}

func (r *printRow) typeChar() *engine.EffectCharacter {
	if len(r.untyped) == 0 {
		return nil
	}
	next := r.untyped[0]
	r.untyped = r.untyped[1:]
	r.typed = append(r.typed, next)
	return next
}

func newTypingHead(base *BaseEffect, coord utils.Coord) *engine.EffectCharacter {
	character := engine.NewEffectCharacter("█", coord)
	character.ExistingColorHandling = base.Config.ExistingColorHandling
	character.UseXterm = base.Config.XtermColors
	character.NoColor = base.Config.NoColor
	if character.Animation != nil {
		character.Animation.UseXterm = base.Config.XtermColors
		character.Animation.NoColor = base.Config.NoColor
	}
	character.InputCoord = coord
	character.Coord = coord
	return character
}

func allFill(chars []*engine.EffectCharacter) bool {
	if len(chars) == 0 {
		return true
	}
	for _, char := range chars {
		if !char.IsFillCharacter {
			return false
		}
	}
	return true
}

func leftMostNonFill(chars []*engine.EffectCharacter) int {
	min := 0
	for _, char := range chars {
		if char.IsFillCharacter {
			continue
		}
		if min == 0 || char.InputCoord.Col < min {
			min = char.InputCoord.Col
		}
	}
	return min
}

func (p *Print) CanvasHeight() int {
	return p.base.CanvasHeight()
}
