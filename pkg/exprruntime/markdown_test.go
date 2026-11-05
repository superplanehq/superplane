package exprruntime

import (
	"github.com/expr-lang/expr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMarkdownExpressions(t *testing.T) {
	for _, tc := range []struct{ expression, value, want string }{
		{`markdownText(value)`, "a|b\n<script> & **bold** [link](url) `code` \\ \r\n\u0085\u2028\u2029", "a&#124;b &#60;script&#62; &#38; &#42;&#42;bold&#42;&#42; &#91;link&#93;&#40;url&#41; &#96;code&#96; &#92;      "},
		{`markdownLink(value)`, "https://example.com/a b>\n\r<", "https://example.com/a%20b%3E%0A%0D%3C"},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			opts := append(MarkdownFunctionOptions(), expr.Env(map[string]any{"value": tc.value}))
			program, err := expr.Compile(tc.expression, opts...)
			require.NoError(t, err)
			got, err := expr.Run(program, map[string]any{"value": tc.value})
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
	for _, invalid := range []string{`markdownText()`, `markdownLink(1)`, `markdownText("a","b")`} {
		_, err := expr.Compile(invalid, MarkdownFunctionOptions()...)
		require.Error(t, err)
	}
}
