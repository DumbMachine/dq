package output

import (
	"os"

	"github.com/mattn/go-isatty"
)

func IsTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

func DefaultFormat() string {
	if envFmt := os.Getenv("DQ_OUTPUT"); envFmt != "" {
		return envFmt
	}
	if IsTTY() {
		return "table"
	}
	return "json"
}
