package automationspec

import (
	_ "embed"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const schemaName = "automation-v2.schema.json"

//go:embed automation-v2.schema.json
var schemaSource string

var (
	definitionSchema = compileDefinitionSchema()
	schemaMessages   = message.NewPrinter(language.English)
)

func compileDefinitionSchema() *jsonschema.Schema {
	schema, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaSource))
	if err != nil {
		panic(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaName, schema); err != nil {
		panic(err)
	}
	return compiler.MustCompile(schemaName)
}

func (d *Document) validateSchema(value any) error {
	err := definitionSchema.Validate(value)
	if err == nil {
		return nil
	}
	var failure *jsonschema.ValidationError
	if !errors.As(err, &failure) {
		return err
	}
	for len(failure.Causes) != 0 {
		failure = failure.Causes[0]
	}

	segments := failure.InstanceLocation
	switch issue := failure.ErrorKind.(type) {
	case *kind.AdditionalProperties:
		path, location := d.sourceAt(slices.Concat(segments, []string{slices.Min(issue.Properties)}))
		return at(path, location.key, "Remove this unknown field or correct its name.")
	case *kind.Required:
		path, location := d.sourceAt(slices.Concat(segments, issue.Missing[:1]))
		return at(path, location.value, "Add this required field.")
	case *kind.UniqueItems:
		path, location := d.sourceAt(slices.Concat(segments, []string{strconv.Itoa(issue.Duplicates[1])}))
		return at(path, location.value, "Enter each value once.")
	case *kind.MaxContains:
		// Point at the first repeated event trigger's uses field.
		path, location := d.sourceAt(slices.Concat(segments, []string{strconv.Itoa(issue.Got[issue.Want]), "uses"}))
		return at(path, location.value, "Use each event trigger once.")
	}

	path, location := d.sourceAt(segments)
	switch failure.ErrorKind.(type) {
	case *kind.FalseSchema:
		return at(path, location.value, "Remove this field.")
	case *kind.Pattern:
		return at(path, location.value, "Enter a nonblank value in the required format.")
	default:
		return at(path, location.value, failure.ErrorKind.LocalizedString(schemaMessages))
	}
}
