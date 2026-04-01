package widgets

import (
	"fmt"
	"io"
)

func clearRenderedLines(writer io.Writer, lines int) {
	if lines <= 0 {
		return
	}

	for i := 0; i < lines; i++ {
		_, _ = fmt.Fprint(writer, "\r\033[2K")
		if i < lines-1 {
			_, _ = fmt.Fprint(writer, "\033[1A")
		}
	}

	_, _ = fmt.Fprint(writer, "\r")
}
