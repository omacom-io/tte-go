package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Decrypt creates a movie style text decryption effect.
// Characters are typed out with encrypted symbols, then decrypt to reveal the final text.
type Decrypt struct {
	base *BaseEffect

	typingPendingChars     []*engine.EffectCharacter
	decryptingPendingChars map[*engine.EffectCharacter]struct{}
	activeCharacters       map[*engine.EffectCharacter]struct{}
	characterFinalColor    map[*engine.EffectCharacter]utils.Color
	encryptedSymbols       []string
	phase                  string // "typing" or "decrypting"

	// Config
	typingSpeed            int
	ciphertextColors       []utils.Color
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

func NewDecrypt(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	ciphertextColors := mustColors("008000", "00cb00", "00ff00")
	finalGradientStops := mustColors("eda000")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	d := &Decrypt{
		base:                   base,
		typingPendingChars:     make([]*engine.EffectCharacter, 0),
		decryptingPendingChars: make(map[*engine.EffectCharacter]struct{}),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		characterFinalColor:    make(map[*engine.EffectCharacter]utils.Color),
		encryptedSymbols:       make([]string, 0),
		phase:                  "typing",
		typingSpeed:            2,
		ciphertextColors:       ciphertextColors,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	d.makeEncryptedSymbols()
	d.build()
	return d
}

// makeEncryptedSymbols creates the list of encrypted symbols to use
func (d *Decrypt) makeEncryptedSymbols() {
	// Keyboard: range(33, 127)
	for n := 33; n < 127; n++ {
		d.encryptedSymbols = append(d.encryptedSymbols, string(rune(n)))
	}
	// Blocks: range(9608, 9632)
	for n := 9608; n < 9632; n++ {
		d.encryptedSymbols = append(d.encryptedSymbols, string(rune(n)))
	}
	// Box drawing: range(9472, 9599)
	for n := 9472; n < 9599; n++ {
		d.encryptedSymbols = append(d.encryptedSymbols, string(rune(n)))
	}
	// Misc: range(174, 452)
	for n := 174; n < 452; n++ {
		d.encryptedSymbols = append(d.encryptedSymbols, string(rune(n)))
	}
}

func (d *Decrypt) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(d.finalGradientStops, d.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		d.base.Canvas.TextBottom,
		d.base.Canvas.TextTop,
		d.base.Canvas.TextLeft,
		d.base.Canvas.TextRight,
		d.finalGradientDirection,
	)

	for _, character := range d.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		d.characterFinalColor[character] = finalGradientMapping[character.InputCoord]
	}

	d.prepareDataForTypeEffect()
	d.prepareDataForDecryptEffect()
}

func (d *Decrypt) prepareDataForTypeEffect() {
	blockChars := []string{"▉", "▓", "▒", "░"}

	for _, character := range d.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		typingScene := character.Animation.NewScene("typing")

		// Block characters cycling through
		for _, blockChar := range blockChars {
			color := d.ciphertextColors[utils.RandIntn(len(d.ciphertextColors))]
			_ = typingScene.AddFrame(blockChar, 2, &utils.ColorPair{FG: &color})
		}

		// Final encrypted symbol
		symbol := d.encryptedSymbols[utils.RandIntn(len(d.encryptedSymbols))]
		color := d.ciphertextColors[utils.RandIntn(len(d.ciphertextColors))]
		_ = typingScene.AddFrame(symbol, 1, &utils.ColorPair{FG: &color})

		d.typingPendingChars = append(d.typingPendingChars, character)
	}
}

func (d *Decrypt) prepareDataForDecryptEffect() {
	for _, character := range d.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		color := d.ciphertextColors[utils.RandIntn(len(d.ciphertextColors))]

		// Fast decrypt scene - 80 rapid frames
		fastDecryptScene := character.Animation.NewScene("fast_decrypt")
		for i := 0; i < 80; i++ {
			symbol := d.encryptedSymbols[utils.RandIntn(len(d.encryptedSymbols))]
			colorCopy := color
			_ = fastDecryptScene.AddFrame(symbol, 2, &utils.ColorPair{FG: &colorCopy})
		}

		// Slow decrypt scene - 1-15 longer duration units
		slowDecryptScene := character.Animation.NewScene("slow_decrypt")
		numFrames := utils.RandIntn(15) + 1
		for i := 0; i < numFrames; i++ {
			symbol := d.encryptedSymbols[utils.RandIntn(len(d.encryptedSymbols))]
			// 30% chance of extra long duration
			var duration int
			if utils.RandIntn(100) <= 30 {
				duration = utils.RandIntn(25) + 35 // range(35, 60)
			} else {
				duration = utils.RandIntn(3) + 3 // range(3, 6)
			}
			colorCopy := color
			_ = slowDecryptScene.AddFrame(symbol, duration, &utils.ColorPair{FG: &colorCopy})
		}

		// Discovered scene - flash white then fade to final color
		discoveredScene := character.Animation.NewScene("discovered")
		white := mustColors("ffffff")[0]
		finalColor := d.characterFinalColor[character]
		discoveredGradient, _ := utils.NewGradient([]utils.Color{white, finalColor}, []int{10}, false)
		symbols := make([]string, len(discoveredGradient.Spectrum))
		for i := range symbols {
			symbols[i] = character.Symbol
		}
		_ = discoveredScene.ApplyGradientToSymbols(symbols, 5, discoveredGradient, nil)

		// Register events: fast_decrypt -> slow_decrypt -> discovered
		character.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			fastDecryptScene,
			engine.ActionActivateScene,
			slowDecryptScene,
		)
		character.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			slowDecryptScene,
			engine.ActionActivateScene,
			discoveredScene,
		)

		// Don't activate yet - typing phase happens first
		d.decryptingPendingChars[character] = struct{}{}
	}
}

func (d *Decrypt) Next() (string, bool) {
	if d.phase == "typing" {
		if len(d.typingPendingChars) > 0 || len(d.activeCharacters) > 0 {
			// 75% chance to type characters this frame
			if len(d.typingPendingChars) > 0 && utils.RandIntn(100) <= 75 {
				for i := 0; i < d.typingSpeed; i++ {
					if len(d.typingPendingChars) > 0 {
						nextChar := d.typingPendingChars[0]
						d.typingPendingChars = d.typingPendingChars[1:]
						d.base.Terminal.SetCharacterVisibility(nextChar, true)
						nextChar.Animation.ActivateScene("typing")
						d.activeCharacters[nextChar] = struct{}{}
					}
				}
			}

			d.update()
			return d.base.Terminal.GetFormattedOutputString(), true
		}

		// Typing phase complete, switch to decrypting
		d.activeCharacters = d.decryptingPendingChars
		for char := range d.activeCharacters {
			char.Animation.ActivateScene("fast_decrypt")
		}
		d.phase = "decrypting"
	}

	if d.phase == "decrypting" {
		if len(d.activeCharacters) > 0 {
			d.update()
			return d.base.Terminal.GetFormattedOutputString(), true
		}
		return d.base.Terminal.GetFormattedOutputString(), false // Done
	}

	return d.base.Terminal.GetFormattedOutputString(), false
}

func (d *Decrypt) update() {
	for char := range d.activeCharacters {
		char.Tick()
		if !char.IsActive() {
			delete(d.activeCharacters, char)
		}
	}
}

func (d *Decrypt) CanvasHeight() int {
	return d.base.CanvasHeight()
}

func (d *Decrypt) CanvasWidth() int {
	return d.base.CanvasWidth()
}
