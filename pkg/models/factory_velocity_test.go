package models

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUUIDArray(t *testing.T) {
	first := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	second := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	cases := []struct {
		name string
		raw  string
		want []uuid.UUID
	}{
		{name: "empty", raw: "", want: nil},
		{name: "empty braces", raw: "{}", want: nil},
		{name: "one id", raw: "{" + first.String() + "}", want: []uuid.UUID{first}},
		{
			name: "two ids",
			raw:  "{" + first.String() + "," + second.String() + "}",
			want: []uuid.UUID{first, second},
		},
		{
			name: "quoted ids",
			raw:  `{"` + first.String() + `","` + second.String() + `"}`,
			want: []uuid.UUID{first, second},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseUUIDArray(tc.raw)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseUUIDArrayRejectsGarbage(t *testing.T) {
	_, err := parseUUIDArray("{not-a-uuid}")
	assert.Error(t, err)
}
