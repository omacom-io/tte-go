package utils

const (
	AnsiReset            = "\x1b[0m"
	AnsiBold             = "\x1b[1m"
	AnsiDim              = "\x1b[2m"
	AnsiItalic           = "\x1b[3m"
	AnsiUnderline        = "\x1b[4m"
	AnsiBlink            = "\x1b[5m"
	AnsiReverse          = "\x1b[7m"
	AnsiHidden           = "\x1b[8m"
	AnsiStrike           = "\x1b[9m"
	AnsiResetStyles      = "\x1b[22;23;24;25;27;28;29m"
	AnsiResetColors      = "\x1b[39;49m"
	AnsiClearScreen      = "\x1b[2J"
	AnsiClearLine        = "\x1b[2K"
	AnsiHomeCursor       = "\x1b[H"
	AnsiSaveCursor       = "\x1b[s"
	AnsiRestoreCursor    = "\x1b[u"
	AnsiWrapDisable      = "\x1b[?7l"
	AnsiWrapEnable       = "\x1b[?7h"
	AnsiDecSaveCursor    = "\x1b7"
	AnsiDecRestoreCursor = "\x1b8"
)
