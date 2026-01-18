package engine

import (
	"fmt"

	"tte-go/internal/utils"
)

type Frame struct {
	Visual       CharacterVisual
	Duration     int
	TicksElapsed int
}

type Scene struct {
	ID                string
	IsLooping         bool
	Frames            []*Frame
	PlayedFrames      []*Frame
	Preexisting       *utils.ColorPair
	UseXtermColors    bool
	NoColor           bool
	Ease              utils.EasingFunction // When set, use eased playback instead of linear frame stepping
	frameIndexMap     map[int]*Frame
	easingTotalSteps  int
	easingCurrentStep int
}

func NewScene(id string, loop bool, useXterm bool, noColor bool) *Scene {
	return &Scene{
		ID:             id,
		IsLooping:      loop,
		UseXtermColors: useXterm,
		NoColor:        noColor,
		frameIndexMap:  map[int]*Frame{},
	}
}

func (s *Scene) AddFrame(symbol string, duration int, colors *utils.ColorPair) error {
	if duration < 1 {
		return fmt.Errorf("frame duration must be >= 1")
	}
	if s.Preexisting != nil {
		colors = s.Preexisting
	}
	visual := CharacterVisual{Symbol: symbol, Colors: colors, UseXterm: s.UseXtermColors, NoColor: s.NoColor}
	frame := &Frame{Visual: visual, Duration: duration}
	s.Frames = append(s.Frames, frame)
	for i := 0; i < duration; i++ {
		s.frameIndexMap[s.easingTotalSteps] = frame
		s.easingTotalSteps++
	}
	return nil
}

func (s *Scene) ApplyGradientToSymbols(symbols []string, duration int, fgGradient *utils.Gradient, bgGradient *utils.Gradient) error {
	if fgGradient == nil && bgGradient == nil {
		return fmt.Errorf("foreground and background gradients are nil")
	}
	if len(symbols) == 0 {
		return fmt.Errorf("symbols cannot be empty")
	}
	colorPairs := []utils.ColorPair{}
	if fgGradient != nil && len(fgGradient.Spectrum) > 0 && bgGradient != nil && len(bgGradient.Spectrum) > 0 {
		if len(fgGradient.Spectrum) >= len(bgGradient.Spectrum) {
			for _, pair := range cyclicDistribution(fgGradient.Spectrum, bgGradient.Spectrum) {
				colorPairs = append(colorPairs, utils.ColorPair{FG: &pair.left, BG: &pair.right})
			}
		} else {
			for _, pair := range cyclicDistribution(bgGradient.Spectrum, fgGradient.Spectrum) {
				colorPairs = append(colorPairs, utils.ColorPair{FG: &pair.right, BG: &pair.left})
			}
		}
	} else if fgGradient != nil && len(fgGradient.Spectrum) > 0 {
		for _, color := range fgGradient.Spectrum {
			c := color
			colorPairs = append(colorPairs, utils.ColorPair{FG: &c})
		}
	} else if bgGradient != nil && len(bgGradient.Spectrum) > 0 {
		for _, color := range bgGradient.Spectrum {
			c := color
			colorPairs = append(colorPairs, utils.ColorPair{BG: &c})
		}
	}
	if len(colorPairs) == 0 {
		return fmt.Errorf("no gradient colors available")
	}
	if len(symbols) >= len(colorPairs) {
		for _, pair := range cyclicDistribution(symbols, colorPairs) {
			if err := s.AddFrame(pair.left, duration, &pair.right); err != nil {
				return err
			}
		}
		return nil
	}
	for _, pair := range cyclicDistribution(colorPairs, symbols) {
		if err := s.AddFrame(pair.right, duration, &pair.left); err != nil {
			return err
		}
	}
	return nil
}

func (s *Scene) Activate() (CharacterVisual, error) {
	if len(s.Frames) == 0 {
		return CharacterVisual{}, fmt.Errorf("scene has no frames")
	}
	return s.Frames[0].Visual, nil
}

func (s *Scene) GetNextVisual() (CharacterVisual, bool) {
	if len(s.Frames) == 0 {
		return CharacterVisual{}, false
	}

	// When easing is set, use eased playback (maps linear step progress to eased frame index)
	if s.Ease != nil {
		// Calculate easing factor: ease(currentStep / totalSteps)
		var easingFactor float64
		if s.easingTotalSteps > 0 {
			easingFactor = s.Ease(float64(s.easingCurrentStep) / float64(s.easingTotalSteps))
		}

		// Map easing factor to frame index
		maxIndex := s.easingTotalSteps - 1
		if maxIndex < 0 {
			maxIndex = 0
		}
		frameIndex := int(easingFactor*float64(maxIndex) + 0.5) // round
		if frameIndex < 0 {
			frameIndex = 0
		}
		if frameIndex > maxIndex {
			frameIndex = maxIndex
		}

		// Get frame from index map
		frame, ok := s.frameIndexMap[frameIndex]
		if !ok {
			// Fallback to first frame if map lookup fails
			frame = s.Frames[0]
		}
		visual := frame.Visual

		// Advance step
		s.easingCurrentStep++

		// Check for completion
		if s.easingCurrentStep >= s.easingTotalSteps {
			if s.IsLooping {
				s.easingCurrentStep = 0
			} else {
				// Mark scene as complete by moving all frames to played
				s.PlayedFrames = append(s.PlayedFrames, s.Frames...)
				s.Frames = nil
			}
		}

		return visual, true
	}

	// Non-eased playback: linear frame stepping
	current := s.Frames[0]
	visual := current.Visual
	current.TicksElapsed++
	if current.TicksElapsed == current.Duration {
		current.TicksElapsed = 0
		s.PlayedFrames = append(s.PlayedFrames, current)
		s.Frames = s.Frames[1:]
		if s.IsLooping && len(s.Frames) == 0 {
			s.Frames = append(s.Frames, s.PlayedFrames...)
			s.PlayedFrames = nil
		}
	}
	return visual, true
}

func (s *Scene) ResetScene() {
	for _, frame := range s.Frames {
		frame.TicksElapsed = 0
		s.PlayedFrames = append(s.PlayedFrames, frame)
	}
	s.Frames = s.PlayedFrames
	s.PlayedFrames = nil
	s.easingCurrentStep = 0
}

func (s *Scene) IsComplete() bool {
	return !s.IsLooping && len(s.Frames) == 0
}

type Animation struct {
	Scenes      map[string]*Scene
	ActiveScene *Scene
	InputColors *utils.ColorPair
	UseXterm    bool
	NoColor     bool
}

func NewAnimation(useXterm bool, noColor bool, inputColors *utils.ColorPair) *Animation {
	return &Animation{Scenes: map[string]*Scene{}, UseXterm: useXterm, NoColor: noColor, InputColors: inputColors}
}

func (a *Animation) NewScene(sceneID string) *Scene {
	scene := NewScene(sceneID, false, a.UseXterm, a.NoColor)
	a.Scenes[sceneID] = scene
	return scene
}

func (a *Animation) AddScene(scene *Scene) {
	if scene == nil {
		return
	}
	a.Scenes[scene.ID] = scene
}

func (a *Animation) ActivateScene(sceneID string) {
	if scene, ok := a.Scenes[sceneID]; ok {
		a.ActiveScene = scene
	}
}

func (a *Animation) ActivateSceneRef(scene *Scene) {
	if scene == nil {
		return
	}
	a.Scenes[scene.ID] = scene
	a.ActiveScene = scene
}

func (a *Animation) DeactivateScene() {
	a.ActiveScene = nil
}

func (a *Animation) QueryScene(sceneID string) *Scene {
	return a.Scenes[sceneID]
}

func (a *Animation) Next() (CharacterVisual, bool) {
	if a.ActiveScene == nil {
		return CharacterVisual{}, false
	}
	return a.ActiveScene.GetNextVisual()
}

func (a *Animation) AdjustColorBrightness(color utils.Color, factor float64) utils.Color {
	return utils.AdjustBrightness(color, factor)
}

type pair[T any, U any] struct {
	left  T
	right U
}

func cyclicDistribution[T any, U any](larger []T, smaller []U) []pair[T, U] {
	repeatFactor := len(larger) / len(smaller)
	overflowCount := len(larger) % len(smaller)
	overflowUsed := false
	smallerIndex := 0
	currentRepeat := 0
	pairs := make([]pair[T, U], 0, len(larger))
	for _, value := range larger {
		if currentRepeat >= repeatFactor {
			if overflowCount > 0 {
				if overflowUsed {
					smallerIndex++
					currentRepeat = 0
					overflowUsed = false
				} else {
					overflowUsed = true
					overflowCount--
				}
			} else {
				smallerIndex++
				currentRepeat = 0
			}
		}
		currentRepeat++
		pairs = append(pairs, pair[T, U]{left: value, right: smaller[smallerIndex]})
	}
	return pairs
}
