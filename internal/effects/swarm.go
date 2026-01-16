package effects

import (
	"fmt"
	"strings"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/terminal"
	"tte-go/internal/utils"
)

type SwarmConfig struct {
	BaseColors          []utils.Color
	FlashColor          utils.Color
	SwarmSize           float64
	SwarmCoordination   float64
	SwarmAreaCountRange [2]int
	FinalGradientStops  []utils.Color
	FinalGradientSteps  []int
	FinalGradientDir    utils.GradientDirection
}

type Swarm struct {
	base                   *BaseEffect
	config                 SwarmConfig
	swarms                 [][]*engine.EffectCharacter
	currentSwarm           []*engine.EffectCharacter
	activeCharacters       map[*engine.EffectCharacter]struct{}
	characterFinalColorMap map[*engine.EffectCharacter]utils.Color
	callNext               bool
	activeSwarmArea        string
}

func NewSwarm(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	effect := &Swarm{
		base:                   base,
		config:                 defaultSwarmConfig(),
		activeCharacters:       map[*engine.EffectCharacter]struct{}{},
		characterFinalColorMap: map[*engine.EffectCharacter]utils.Color{},
		callNext:               true,
	}
	effect.build()
	return effect
}

func defaultSwarmConfig() SwarmConfig {
	return SwarmConfig{
		BaseColors:          mustColors("31a0d4"),
		FlashColor:          mustColors("f2ea79")[0],
		SwarmSize:           0.1,
		SwarmCoordination:   0.8,
		SwarmAreaCountRange: [2]int{2, 4},
		FinalGradientStops:  mustColors("31b900", "f0ff65"),
		FinalGradientSteps:  []int{12},
		FinalGradientDir:    utils.GradientHorizontal,
	}
}

func (s *Swarm) build() {
	for _, character := range s.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		s.base.Terminal.SetCharacterVisibility(character, false)
	}

	allCharacters := s.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight)
	swarmSize := max(utils.PyRound(float64(len(allCharacters))*s.config.SwarmSize), 1)
	s.makeSwarms(swarmSize)

	finalGradient, _ := utils.NewGradient(s.config.FinalGradientStops, s.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		s.base.Canvas.TextBottom,
		s.base.Canvas.TextTop,
		s.base.Canvas.TextLeft,
		s.base.Canvas.TextRight,
		s.config.FinalGradientDir,
	)
	for _, character := range allCharacters {
		s.characterFinalColorMap[character] = finalMapping[character.InputCoord]
	}

	flashList := make([]utils.Color, 10)
	for i := range flashList {
		flashList[i] = s.config.FlashColor
	}
	minDimension := minInt(s.base.Canvas.Right, s.base.Canvas.Top)
	areaDiameter := max(minDimension/6, 1) * 2
	radius := max(minDimension/2, 1)
	for _, swarm := range s.swarms {
		baseColor := s.config.BaseColors[utils.RandIntn(len(s.config.BaseColors))]
		swarmGradient, _ := utils.NewGradient([]utils.Color{baseColor, s.config.FlashColor}, []int{7}, false)
		swarmGradientMirror := make([]utils.Color, 0, len(swarmGradient.Spectrum)*2+len(flashList))
		swarmGradientMirror = append(swarmGradientMirror, swarmGradient.Spectrum...)
		swarmGradientMirror = append(swarmGradientMirror, flashList...)
		for i := len(swarmGradient.Spectrum) - 1; i >= 0; i-- {
			swarmGradientMirror = append(swarmGradientMirror, swarmGradient.Spectrum[i])
		}

		swarmSpawn := randomCoord(s.base.Canvas, true)
		swarmAreaCount := randIntRange(s.config.SwarmAreaCountRange[0], s.config.SwarmAreaCountRange[1])
		lastFocusCoord := swarmSpawn
		swarmAreaCoordsList := make([][]utils.Coord, 0, swarmAreaCount)
		for len(swarmAreaCoordsList) < swarmAreaCount {
			potential := utils.FindCoordsOnCircle(lastFocusCoord, radius, 0, true)
			if len(potential) == 0 {
				potential = []utils.Coord{randomCoord(s.base.Canvas, false)}
			}
			utils.Shuffle(len(potential), func(i, j int) { potential[i], potential[j] = potential[j], potential[i] })
			nextFocusCoord := utils.Coord{}
			found := false
			for _, coord := range potential {
				if coordIsInCanvas(s.base.Canvas, coord) {
					nextFocusCoord = coord
					found = true
					break
				}
			}
			if !found {
				nextFocusCoord = randomCoord(s.base.Canvas, false)
			}
			swarmAreaCoordsList = append(swarmAreaCoordsList, utils.FindCoordsInCircle(lastFocusCoord, areaDiameter))
			lastFocusCoord = nextFocusCoord
		}

		for _, character := range swarm {
			paths := []*engine.Path{}
			character.Motion.SetCoordinate(swarmSpawn)
			flashScene := character.Animation.NewScene("flash")
			for _, step := range swarmGradientMirror {
				color := step
				_ = flashScene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &color})
			}
			swarmAreaIndex := 0
			for _, swarmAreaCoords := range swarmAreaCoordsList {
				if len(swarmAreaCoords) == 0 {
					swarmAreaIndex++
					continue
				}
				swarmAreaName := fmt.Sprintf("%d_swarm_area", swarmAreaIndex)
				swarmAreaIndex++
				originPath, _ := character.Motion.NewPath(0.4, utils.OutSine, nil, 0, false, swarmAreaName)
				originPath.AddWaypoint(swarmAreaCoords[utils.RandIntn(len(swarmAreaCoords))])
				_ = character.EventHandler.RegisterEvent(engine.EventPathActivated, originPath, engine.ActionActivateScene, flashScene)
				_ = character.EventHandler.RegisterEvent(engine.EventPathActivated, originPath, engine.ActionSetLayer, 1)
				_ = character.EventHandler.RegisterEvent(engine.EventPathComplete, originPath, engine.ActionDeactivateScene, nil)
				paths = append(paths, originPath)

				for i := 0; i < 2; i++ {
					nextCoord := swarmAreaCoords[utils.RandIntn(len(swarmAreaCoords))]
					innerPath, _ := character.Motion.NewPath(0.18, utils.InOutSine, nil, 0, false, "")
					innerPath.AddWaypoint(nextCoord)
					paths = append(paths, innerPath)
				}
			}

			inputPath, _ := character.Motion.NewPath(0.45, utils.InOutQuad, nil, 0, false, "")
			inputPath.AddWaypoint(character.InputCoord)
			inputScene := character.Animation.NewScene("input")
			inputGradient, _ := utils.NewGradient([]utils.Color{s.config.FlashColor, s.characterFinalColorMap[character]}, []int{10}, false)
			for _, step := range inputGradient.Spectrum {
				color := step
				_ = inputScene.AddFrame(character.Symbol, 3, &utils.ColorPair{FG: &color})
			}
			_ = character.EventHandler.RegisterEvent(engine.EventPathComplete, inputPath, engine.ActionActivateScene, inputScene)
			_ = character.EventHandler.RegisterEvent(engine.EventPathComplete, inputPath, engine.ActionSetLayer, 0)
			_ = character.EventHandler.RegisterEvent(engine.EventPathActivated, inputPath, engine.ActionActivateScene, flashScene)
			paths = append(paths, inputPath)

			chainPaths(character, paths)
		}
	}

	s.callNext = true
	s.activeSwarmArea = "0_swarm_area"
}

func (s *Swarm) makeSwarms(swarmSize int) {
	unswarmed := s.base.Terminal.GetCharacters(true, false, false, false, engine.BottomToTopRightToLeft)
	for len(unswarmed) > 0 {
		newSwarm := []*engine.EffectCharacter{}
		for i := 0; i < swarmSize && len(unswarmed) > 0; i++ {
			last := len(unswarmed) - 1
			newSwarm = append(newSwarm, unswarmed[last])
			unswarmed = unswarmed[:last]
		}
		s.swarms = append(s.swarms, newSwarm)
	}
	if len(s.swarms) == 0 {
		return
	}
	finalSwarm := s.swarms[len(s.swarms)-1]
	s.swarms = s.swarms[:len(s.swarms)-1]
	if len(finalSwarm) < swarmSize/2 && len(s.swarms) > 0 {
		s.swarms[len(s.swarms)-1] = append(s.swarms[len(s.swarms)-1], finalSwarm...)
		return
	}
	s.swarms = append(s.swarms, finalSwarm)
}

func (s *Swarm) Next() (string, bool) {
	if len(s.swarms) == 0 && len(s.activeCharacters) == 0 {
		return "", false
	}
	if len(s.swarms) > 0 && s.callNext {
		s.callNext = false
		s.currentSwarm = s.swarms[len(s.swarms)-1]
		s.swarms = s.swarms[:len(s.swarms)-1]
		s.activeSwarmArea = "0_swarm_area"
		for _, character := range s.currentSwarm {
			if path, err := character.Motion.QueryPath("0_swarm_area"); err == nil {
				character.Motion.ActivatePath(path)
			}
			s.base.Terminal.SetCharacterVisibility(character, true)
			s.activeCharacters[character] = struct{}{}
		}
	}
	if len(s.activeCharacters) < len(s.currentSwarm) {
		s.callNext = true
	}
	if len(s.currentSwarm) > 0 {
		for _, character := range s.currentSwarm {
			activePath := character.Motion.ActivePath
			if activePath == nil {
				continue
			}
			if activePath.ID == s.activeSwarmArea {
				continue
			}
			if !strings.Contains(activePath.ID, "swarm_area") {
				continue
			}
			if swarmAreaIndex(activePath.ID) > swarmAreaIndex(s.activeSwarmArea) {
				s.activeSwarmArea = activePath.ID
				for _, other := range s.currentSwarm {
					if other == character {
						continue
					}
					if utils.RandFloat64() < s.config.SwarmCoordination {
						if path, ok := other.Motion.Paths[s.activeSwarmArea]; ok {
							other.Motion.ActivatePath(path)
						}
					}
				}
				break
			}
		}
	}
	for character := range s.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			delete(s.activeCharacters, character)
		}
	}
	return s.base.Terminal.GetFormattedOutputString(), true
}

func chainPaths(character *engine.EffectCharacter, paths []*engine.Path) {
	if len(paths) < 2 {
		return
	}
	for i := 1; i < len(paths); i++ {
		_ = character.EventHandler.RegisterEvent(engine.EventPathComplete, paths[i-1], engine.ActionActivatePath, paths[i])
	}
}

func randomCoord(canvas *terminal.Canvas, outside bool) utils.Coord {
	if canvas == nil {
		return utils.Coord{}
	}
	if outside {
		options := []utils.Coord{
			{Row: canvas.Top + 1, Col: randomColumn(canvas)},
			{Row: canvas.Bottom - 1, Col: randomColumn(canvas)},
			{Row: randomRow(canvas), Col: canvas.Left - 1},
			{Row: randomRow(canvas), Col: canvas.Right + 1},
		}
		return options[utils.RandIntn(len(options))]
	}
	return utils.Coord{Row: randomRow(canvas), Col: randomColumn(canvas)}
}

func randomColumn(canvas *terminal.Canvas) int {
	return utils.RandIntn(max(1, canvas.Width)) + canvas.Left
}

func randomRow(canvas *terminal.Canvas) int {
	return utils.RandIntn(max(1, canvas.Height)) + canvas.Bottom
}

func coordIsInCanvas(canvas *terminal.Canvas, coord utils.Coord) bool {
	return coord.Col >= canvas.Left && coord.Col <= canvas.Right && coord.Row >= canvas.Bottom && coord.Row <= canvas.Top
}

func randIntRange(minValue, maxValue int) int {
	if maxValue < minValue {
		return minValue
	}
	return utils.RandIntn(maxValue-minValue+1) + minValue
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func swarmAreaIndex(pathID string) int {
	value := 0
	for _, r := range pathID {
		if r < '0' || r > '9' {
			break
		}
		value = value*10 + int(r-'0')
	}
	return value
}
