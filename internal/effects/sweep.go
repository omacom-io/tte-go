package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type SweepConfig struct {
	SweepSymbols         []string
	FirstSweepDirection  engine.CharacterGroup
	SecondSweepDirection engine.CharacterGroup
	FinalGradientStops   []utils.Color
	FinalGradientSteps   []int
	FinalGradientDir     utils.GradientDirection
}

type Sweep struct {
	base                 *BaseEffect
	config               SweepConfig
	groupsFirstSweep     [][]*engine.EffectCharacter
	groupsSecondSweep    [][]*engine.EffectCharacter
	easer                *utils.SequenceEaser[[]*engine.EffectCharacter]
	activeCharacters     map[*engine.EffectCharacter]struct{}
	characterFinalColors map[*engine.EffectCharacter]utils.Color
	phase                string
	complete             bool
}

func NewSweep(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	effect := &Sweep{
		base:                 base,
		config:               defaultSweepConfig(),
		activeCharacters:     map[*engine.EffectCharacter]struct{}{},
		characterFinalColors: map[*engine.EffectCharacter]utils.Color{},
		phase:                "first sweep",
		complete:             false,
	}
	effect.build()
	return effect
}

func defaultSweepConfig() SweepConfig {
	return SweepConfig{
		SweepSymbols:         []string{"█", "▓", "▒", "░"},
		FirstSweepDirection:  engine.ColumnRightToLeft,
		SecondSweepDirection: engine.ColumnLeftToRight,
		FinalGradientStops:   mustColors("8A008A", "00D1FF", "ffffff"),
		FinalGradientSteps:   []int{8},
		FinalGradientDir:     utils.GradientVertical,
	}
}

func (s *Sweep) build() {
	finalGradient, _ := utils.NewGradient(s.config.FinalGradientStops, s.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		s.base.Canvas.TextBottom,
		s.base.Canvas.TextTop,
		s.base.Canvas.TextLeft,
		s.base.Canvas.TextRight,
		s.config.FinalGradientDir,
	)

	shadesOfGray := []utils.Color{
		{R: 0xA0, G: 0xA0, B: 0xA0},
		{R: 0x80, G: 0x80, B: 0x80},
		{R: 0x40, G: 0x40, B: 0x40},
		{R: 0x20, G: 0x20, B: 0x20},
		{R: 0x10, G: 0x10, B: 0x10},
	}
	grayColor := utils.Color{R: 0x80, G: 0x80, B: 0x80}
	blackColor := utils.Color{R: 0, G: 0, B: 0}

	allChars := s.base.Terminal.GetCharacters(true, true, true, false, engine.TopToBottomLeftToRight)
	for _, character := range allChars {
		if !character.IsFillCharacter {
			s.characterFinalColors[character] = finalMapping[character.InputCoord]
		}

		// Initial sweep scene - reveals in gray
		initialSweepScene := character.Animation.NewScene("initial_sweep")
		for _, sym := range s.config.SweepSymbols {
			gray := shadesOfGray[utils.RandIntn(len(shadesOfGray))]
			_ = initialSweepScene.AddFrame(sym, 5, &utils.ColorPair{FG: &gray})
		}
		_ = initialSweepScene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &grayColor})

		// Second sweep scene - colorizes
		secondSweepScene := character.Animation.NewScene("second_sweep")
		for _, sym := range s.config.SweepSymbols {
			spectrumColor := finalGradient.Spectrum[utils.RandIntn(len(finalGradient.Spectrum))]
			_ = secondSweepScene.AddFrame(sym, 5, &utils.ColorPair{FG: &spectrumColor})
		}
		var finalColor utils.Color
		if character.IsFillCharacter {
			finalColor = blackColor
		} else {
			finalColor = finalMapping[character.InputCoord]
		}
		_ = secondSweepScene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &finalColor})

		s.base.Terminal.SetCharacterVisibility(character, false)
	}

	s.groupsFirstSweep = s.base.Terminal.GetCharactersGrouped(s.config.FirstSweepDirection, true, true, true, false)
	s.groupsSecondSweep = s.base.Terminal.GetCharactersGrouped(s.config.SecondSweepDirection, true, true, true, false)
	s.easer = utils.NewSequenceEaser(s.groupsFirstSweep, utils.InOutCirc)
}

func (s *Sweep) Next() (string, bool) {
	if s.complete && len(s.activeCharacters) == 0 {
		return "", false
	}

	s.easer.Step()
	for _, group := range s.easer.Added {
		for _, character := range group {
			if s.phase == "first sweep" {
				s.base.Terminal.SetCharacterVisibility(character, true)
			}
			if s.phase == "first sweep" {
				character.Animation.ActivateScene("initial_sweep")
			} else {
				character.Animation.ActivateScene("second_sweep")
			}
			s.activeCharacters[character] = struct{}{}
		}
	}

	if s.easer.IsComplete() && s.phase == "first sweep" {
		s.easer.Sequence = s.groupsSecondSweep
		s.easer.Reset()
		s.phase = "second sweep"
	} else if s.easer.IsComplete() && s.phase == "second sweep" {
		s.complete = true
	}

	for character := range s.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			delete(s.activeCharacters, character)
		}
	}

	return s.base.Terminal.GetFormattedOutputString(), true
}

func (s *Sweep) CanvasHeight() int {
	return s.base.CanvasHeight()
}

func (s *Sweep) CanvasWidth() int {
	return s.base.CanvasWidth()
}
