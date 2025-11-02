package queries

import (
	"fmt"

	pw "github.com/playwright-community/playwright-go"
)

type Runner interface {
	Page() pw.Page
}

type Query struct {
	run      func(r Runner) pw.Locator
	describe func() string
}

func (q Query) Run(r Runner) pw.Locator { return q.run(r) }
func (q Query) Describe() string        { return q.describe() }

// Lookup by test ID
func TestID(testID string) Query {
	return Query{
		run: func(r Runner) pw.Locator {
			return r.Page().GetByTestId(testID).First()
		},
		describe: func() string {
			return fmt.Sprintf("testID=\"%s\"", testID)
		},
	}
}

// Lookup by visible text
func Text(text string) Query {
	return Query{
		run: func(r Runner) pw.Locator {
			return r.Page().Locator("text=" + text).First()
		},
		describe: func() string {
			return fmt.Sprintf("text=\"%s\"", text)
		},
	}
}
