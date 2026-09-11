package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOrganizationURL(t *testing.T) {
	t.Parallel()

	t.Run("accepts and normalizes typical organization URLs", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name  string
			input string
			want  string
		}{
			{name: "canonical URL", input: "https://acme.semaphoreci.com", want: "https://acme.semaphoreci.com"},
			{name: "trailing slash", input: "https://acme.semaphoreci.com/", want: "https://acme.semaphoreci.com"},
			{name: "multiple trailing slashes", input: "https://acme.semaphoreci.com///", want: "https://acme.semaphoreci.com"},
			{name: "surrounding whitespace", input: "  https://acme.semaphoreci.com/  ", want: "https://acme.semaphoreci.com"},
			{name: "path copied from the address bar", input: "https://acme.semaphoreci.com/projects", want: "https://acme.semaphoreci.com"},
			{name: "path with trailing slash", input: "https://acme.semaphoreci.com/projects/", want: "https://acme.semaphoreci.com"},
			{name: "query string", input: "https://acme.semaphoreci.com/?ref=home", want: "https://acme.semaphoreci.com"},
			{name: "fragment", input: "https://acme.semaphoreci.com/#org", want: "https://acme.semaphoreci.com"},
			{name: "missing scheme", input: "acme.semaphoreci.com", want: "https://acme.semaphoreci.com"},
			{name: "missing scheme with trailing slash", input: "acme.semaphoreci.com/", want: "https://acme.semaphoreci.com"},
			{name: "https with port", input: "https://semaphore.internal:3000/", want: "https://semaphore.internal:3000"},
			{name: "uppercase host", input: "https://Acme.SemaphoreCI.com/", want: "https://acme.semaphoreci.com"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				got, err := ParseOrganizationURL(tc.input)
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			})
		}
	})

	t.Run("rejects invalid organization URLs", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name    string
			input   string
			wantErr string
		}{
			{name: "empty", input: "", wantErr: "organization URL is required"},
			{name: "whitespace only", input: "   ", wantErr: "organization URL is required"},
			{name: "http scheme", input: "http://acme.semaphoreci.com", wantErr: "organization URL must use https"},
			{name: "unsupported scheme", input: "ftp://acme.semaphoreci.com", wantErr: "organization URL must use https"},
			{name: "missing host", input: "https://", wantErr: "organization URL must include a host"},
			{name: "credentials", input: "https://user:pass@acme.semaphoreci.com", wantErr: "organization URL must not include credentials"},
			{name: "spaces in host", input: "https://acme semaphoreci.com", wantErr: "invalid organization URL"},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				_, err := ParseOrganizationURL(tc.input)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			})
		}
	})
}
