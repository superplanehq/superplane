package notion

import (
	_ "embed"
	"sync"

	"github.com/superplanehq/superplane/pkg/utils"
)

//go:embed example_output_search.json
var searchExample []byte

//go:embed example_output_create_page.json
var createPageExample []byte

var searchExampleOnce sync.Once
var searchExampleOutput map[string]any
var createPageExampleOnce sync.Once
var createPageExampleOutput map[string]any

func (s *Search) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&searchExampleOnce, searchExample, &searchExampleOutput)
}

func (c *CreatePage) ExampleOutput() map[string]any {
	return utils.UnmarshalEmbeddedJSON(&createPageExampleOnce, createPageExample, &createPageExampleOutput)
}
