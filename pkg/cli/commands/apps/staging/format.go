package staging

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

func formatStagedPathLine(w io.Writer, path string) string {
	if !colorEnabled(w) {
		return "M " + path
	}

	// Match git short status: green status letter, default path.
	return "\x1b[32mM\x1b[0m " + path
}

func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	switch os.Getenv("FORCE_COLOR") {
	case "0", "false", "no":
		return false
	case "1", "true", "yes":
		return true
	}

	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	return isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
}
