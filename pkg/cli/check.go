package cli

import (
	"flag"
	"fmt"
	"os"
)

// Checks if an error is present.
//
// If it is present, it displays the error and exits with status 1.
//
// If you want to display a custom message use CheckWithMessage.
func Check(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())

		Exit(1)
	}
}

// Checks if an error is present.
//
// If it is present, it displays the provided message and exits with status 1.
func CheckWithMessage(err error, message string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", message)

		Exit(1)
	}
}

func Fail(message string) {
	fmt.Fprintf(os.Stderr, "error: %s\n", message)

	Exit(1)
}

func Exit(code int) {
	if flag.Lookup("test.v") == nil {
		os.Exit(1)
	} else {
		panic(fmt.Sprintf("exit %d", code))
	}
}
