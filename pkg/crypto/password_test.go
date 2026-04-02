package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test__ValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{name: "valid password", password: "Abcdef1!", wantErr: false},
		{name: "complex valid password", password: "MyP@ssw0rd!", wantErr: false},
		{name: "too short", password: "Ab1!", wantErr: true},
		{name: "no uppercase", password: "abcdef1!", wantErr: true},
		{name: "no lowercase", password: "ABCDEF1!", wantErr: true},
		{name: "no digit", password: "Abcdefg!", wantErr: true},
		{name: "no symbol", password: "Abcdef12", wantErr: true},
		{name: "empty password", password: "", wantErr: true},
		{name: "only letters", password: "Abcdefgh", wantErr: true},
		{name: "only digits", password: "12345678", wantErr: true},
		{name: "meets all requirements", password: "Test123!@#", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
