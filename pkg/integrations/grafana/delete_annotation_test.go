package grafana

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
)

func Test__normalizeAnnotationIDRaw__StringAndNumber(t *testing.T) {
	require.Equal(t, "42", normalizeAnnotationIDRaw("42"))
	require.Equal(t, "42", normalizeAnnotationIDRaw(float64(42)))
	require.Equal(t, "99", normalizeAnnotationIDRaw(int64(99)))
}

func Test__parseAnnotationIDString__Valid(t *testing.T) {
	id, err := parseAnnotationIDString("  7 ")
	require.NoError(t, err)
	require.Equal(t, int64(7), id)
}

func Test__parseAnnotationIDString__RejectsInvalid(t *testing.T) {
	_, err := parseAnnotationIDString("")
	require.Error(t, err)
	_, err = parseAnnotationIDString("0")
	require.Error(t, err)
	_, err = parseAnnotationIDString("-1")
	require.Error(t, err)
}

func Test__formatAnnotationResourceName__TextAndEmpty(t *testing.T) {
	require.Contains(t, formatAnnotationResourceName(Annotation{ID: 3, Text: "hello"}), "#3")
	require.Contains(t, formatAnnotationResourceName(Annotation{ID: 3, Text: "hello"}), "hello")
	require.Equal(t, "#9", formatAnnotationResourceName(Annotation{ID: 9, Text: "   "}))
}

func Test__DeleteAnnotation__Setup__AllowsExpression(t *testing.T) {
	d := &DeleteAnnotation{}
	err := d.Setup(core.SetupContext{
		Configuration: map[string]any{
			"annotationId": "{{ $['Create Annotation'].data.id }}",
		},
	})
	require.NoError(t, err)
}

func Test__parseAnnotationIDForExecute__RejectsUnresolvedExpression(t *testing.T) {
	_, err := parseAnnotationIDForExecute("{{ $['x'].id }}")
	require.Error(t, err)
	require.Contains(t, err.Error(), "resolve")
}

func Test__isExpressionValue__Brackets(t *testing.T) {
	require.True(t, isExpressionValue("{{ foo }}"))
	require.True(t, isExpressionValue("$['a'].b"))
	require.False(t, isExpressionValue("42"))
}
