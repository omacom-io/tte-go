package config

import "flag"

type TerminalConfig struct {
	TabWidth                 int
	XtermColors              bool
	NoColor                  bool
	TerminalBackgroundColor  string
	ExistingColorHandling    string
	WrapText                 bool
	FrameRate                int
	CanvasWidth              int
	CanvasHeight             int
	AnchorCanvas             string
	AnchorText               string
	IgnoreTerminalDimensions bool
	ReuseCanvas              bool
	NoEOL                    bool
	NoRestoreCursor          bool
}

func DefaultTerminalConfig() TerminalConfig {
	return TerminalConfig{
		TabWidth:                 4,
		XtermColors:              false,
		NoColor:                  false,
		TerminalBackgroundColor:  "#000000",
		ExistingColorHandling:    "ignore",
		WrapText:                 false,
		FrameRate:                60,
		CanvasWidth:              -1,
		CanvasHeight:             -1,
		AnchorCanvas:             "sw",
		AnchorText:               "sw",
		IgnoreTerminalDimensions: false,
		ReuseCanvas:              false,
		NoEOL:                    false,
		NoRestoreCursor:          false,
	}
}

func (cfg *TerminalConfig) BindFlags(fs *flag.FlagSet) {
	fs.IntVar(&cfg.TabWidth, "tab-width", cfg.TabWidth, "Number of spaces to use for a tab character")
	fs.BoolVar(&cfg.XtermColors, "xterm-colors", cfg.XtermColors, "Convert 24-bit RGB hex to XTerm-256")
	fs.BoolVar(&cfg.NoColor, "no-color", cfg.NoColor, "Disable all colors in the effect")
	fs.StringVar(&cfg.TerminalBackgroundColor, "terminal-background-color", cfg.TerminalBackgroundColor, "Terminal background color (XTerm 0-255 or RGB hex)")
	fs.StringVar(&cfg.ExistingColorHandling, "existing-color-handling", cfg.ExistingColorHandling, "Handling for existing ANSI colors: always|dynamic|ignore")
	fs.BoolVar(&cfg.WrapText, "wrap-text", cfg.WrapText, "Wrap text wider than the canvas width")
	fs.IntVar(&cfg.FrameRate, "frame-rate", cfg.FrameRate, "Target frame rate (fps). Set to 0 to disable")
	fs.IntVar(&cfg.CanvasWidth, "canvas-width", cfg.CanvasWidth, "Canvas width: >0 fixed, 0 terminal width, -1 input width")
	fs.IntVar(&cfg.CanvasHeight, "canvas-height", cfg.CanvasHeight, "Canvas height: >0 fixed, 0 terminal height, -1 input height")
	fs.StringVar(&cfg.AnchorCanvas, "anchor-canvas", cfg.AnchorCanvas, "Anchor point for canvas (sw,s,se,e,ne,n,nw,w,c)")
	fs.StringVar(&cfg.AnchorText, "anchor-text", cfg.AnchorText, "Anchor point for text (n,ne,e,se,s,sw,w,nw,c)")
	fs.BoolVar(&cfg.IgnoreTerminalDimensions, "ignore-terminal-dimensions", cfg.IgnoreTerminalDimensions, "Ignore terminal dimensions")
	fs.BoolVar(&cfg.ReuseCanvas, "reuse-canvas", cfg.ReuseCanvas, "Reuse existing canvas rows")
	fs.BoolVar(&cfg.NoEOL, "no-eol", cfg.NoEOL, "Suppress trailing newline after effect completes")
	fs.BoolVar(&cfg.NoRestoreCursor, "no-restore-cursor", cfg.NoRestoreCursor, "Do not restore cursor visibility after effect")
}
