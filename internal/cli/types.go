package cli

const (
	exitOK           = 0
	exitGeneric      = 1
	exitUsage        = 2
	exitNetwork      = 3
	exitVerify       = 4
	exitCountMismatch = 5
	exitPartial      = 6
)

type Config struct {
	Quiet      bool
	Verbose    int
	JSON       bool
	Plain      bool
	NoColor    bool
	NoInput    bool
	ConfigPath string
}
