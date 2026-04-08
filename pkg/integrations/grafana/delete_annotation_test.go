package grafana

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
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

func Test__formatAnnotationResourceName__TruncatesByRunesPreservesUTF8(t *testing.T) {
	long := strings.Repeat("あ", 100)
	out := formatAnnotationResourceName(Annotation{ID: 1, Text: long})
	require.True(t, utf8.ValidString(out), "truncation must not split UTF-8 codepoints")
	require.Contains(t, out, "…")
	// "あ" is one rune each; truncated body is 72 runes + ellipsis
	idx := strings.Index(out, " · ")
	require.Greater(t, idx, 0)
	body := out[idx+len(" · "):]
	require.LessOrEqual(t, utf8.RuneCountInString(body), 73)
}

func Test__DeleteAnnotation__Setup__AllowsExpression(t *testing.T) {
	d := &DeleteAnnotation{}
	metadata := &contexts.MetadataContext{}
	err := d.Setup(core.SetupContext{
		Configuration: map[string]any{
			"annotationId": "{{ $['Create Annotation'].data.id }}",
		},
		Metadata: metadata,
	})
	require.NoError(t, err)
	require.Equal(t, AnnotationNodeMetadata{AnnotationLabel: "{{ $['Create Annotation'].data.id }}"}, metadata.Metadata)
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

func Test__DeleteAnnotation__Setup__StoresAnnotationLabelMetadata(t *testing.T) {
	d := &DeleteAnnotation{}
	metadata := &contexts.MetadataContext{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`{"id":42,"text":"Deploy finished","time":0}`,
				)),
			},
		},
	}

	err := d.Setup(core.SetupContext{
		Configuration: map[string]any{
			"annotationId": "42",
		},
		Metadata: metadata,
		HTTP:     httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"baseURL":  "https://grafana.example.com",
				"apiToken": "token",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, AnnotationNodeMetadata{AnnotationLabel: "#42 · Deploy finished"}, metadata.Metadata)
}
