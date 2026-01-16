package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type RainConfig struct {
	RainColors         []utils.Color
	MovementSpeed      [2]float64
	RainSymbols        []string
	FinalGradientStops []utils.Color
	FinalGradientSteps []int
	FinalGradientDir   utils.GradientDirection
	MovementEasing     utils.EasingFunction
}

type Rain struct {
	base             *BaseEffect
	config           RainConfig
	pendingChars     []*engine.EffectCharacter
	groupByRow       map[int][]*engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}
	finalColorMap    map[*engine.EffectCharacter]utils.Color
}

func NewRain(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	effect := &Rain{
		base:             base,
		config:           defaultRainConfig(),
		groupByRow:       map[int][]*engine.EffectCharacter{},
		activeCharacters: map[*engine.EffectCharacter]struct{}{},
		finalColorMap:    map[*engine.EffectCharacter]utils.Color{},
	}
	effect.build()
	return effect
}

func defaultRainConfig() RainConfig {
	return RainConfig{
		RainColors:         mustColors("00315C", "004C8F", "0075DB", "3F91D9", "78B9F2", "9AC8F5", "B8D8F8", "E3EFFC"),
		MovementSpeed:      [2]float64{0.33, 0.57},
		RainSymbols:        []string{"o", ".", ",", "*", "|"},
		FinalGradientStops: mustColors("488bff", "b2e7de", "57eaf7"),
		FinalGradientSteps: []int{12},
		FinalGradientDir:   utils.GradientDiagonal,
		MovementEasing:     utils.InQuart,
	}
}

func (r *Rain) build() {
	finalGradient, _ := utils.NewGradient(r.config.FinalGradientStops, r.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		r.base.Canvas.TextBottom,
		r.base.Canvas.TextTop,
		r.base.Canvas.TextLeft,
		r.base.Canvas.TextRight,
		r.config.FinalGradientDir,
	)
	for _, character := range r.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		r.finalColorMap[character] = finalMapping[character.InputCoord]
	}

	for _, character := range r.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		rainColor := r.config.RainColors[utils.RandIntn(len(r.config.RainColors))]
		rainScene := character.Animation.NewScene("rain")
		_ = rainScene.AddFrame(r.config.RainSymbols[utils.RandIntn(len(r.config.RainSymbols))], 1, &utils.ColorPair{FG: &rainColor})
		rainScene.Preexisting = nil
		fadeScene := character.Animation.NewScene("fade")
		fadeGradient, _ := utils.NewGradient([]utils.Color{rainColor, r.finalColorMap[character]}, []int{7}, false)
		_ = fadeScene.ApplyGradientToSymbols([]string{character.Symbol}, 3, fadeGradient, nil)
		character.Animation.ActivateScene("rain")
		character.Motion.SetCoordinate(utils.Coord{Row: r.base.Canvas.Top, Col: character.InputCoord.Col})
		path, _ := character.Motion.NewPath(randomRange(r.config.MovementSpeed[0], r.config.MovementSpeed[1]), r.config.MovementEasing, nil, 0, false, "")
		path.AddWaypoint(character.InputCoord)
		_ = character.EventHandler.RegisterEvent(engine.EventPathComplete, path, engine.ActionActivateScene, fadeScene)
		character.Motion.ActivatePath(path)
		r.pendingChars = append(r.pendingChars, character)
	}

	for _, character := range r.pendingChars {
		row := character.InputCoord.Row
		r.groupByRow[row] = append(r.groupByRow[row], character)
	}
	r.pendingChars = nil
}

func (r *Rain) Next() (string, bool) {
	if len(r.groupByRow) > 0 || len(r.activeCharacters) > 0 || len(r.pendingChars) > 0 {
		if len(r.pendingChars) == 0 && len(r.groupByRow) > 0 {
			minRow := minRowKey(r.groupByRow)
			r.pendingChars = append(r.pendingChars, r.groupByRow[minRow]...)
			delete(r.groupByRow, minRow)
		}
		if len(r.pendingChars) > 0 {
			spawnCount := utils.RandIntn(2) + 1
			for i := 0; i < spawnCount; i++ {
				if len(r.pendingChars) == 0 {
					break
				}
				index := utils.RandIntn(len(r.pendingChars))
				next := r.pendingChars[index]
				r.pendingChars = append(r.pendingChars[:index], r.pendingChars[index+1:]...)
				r.base.Terminal.SetCharacterVisibility(next, true)
				r.activeCharacters[next] = struct{}{}
			}
		}
		for character := range r.activeCharacters {
			character.Tick()
			if !character.IsActive() {
				delete(r.activeCharacters, character)
			}
		}
		return r.base.Terminal.GetFormattedOutputString(), true
	}
	return "", false
}

func randomRange(minValue, maxValue float64) float64 {
	if maxValue <= minValue {
		return minValue
	}
	return minValue + utils.RandFloat64()*(maxValue-minValue)
}

func minRowKey(groups map[int][]*engine.EffectCharacter) int {
	min := 0
	for key := range groups {
		min = key
		break
	}
	for key := range groups {
		if key < min {
			min = key
		}
	}
	return min
}

func (r *Rain) CanvasHeight() int {
	return r.base.CanvasHeight()
}
