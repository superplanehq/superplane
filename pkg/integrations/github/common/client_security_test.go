package common

import (
	"net/http"
	"testing"

	"github.com/google/go-github/v84/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPDoRejectsUnrelatedHost(t *testing.T) {
	client := &Client{underlying: github.NewClient(http.DefaultClient), host: DotCom()}
	request, err := http.NewRequest(http.MethodGet, "https://attacker.example/image.png", nil)
	require.NoError(t, err)

	response, err := client.HTTPDo(request)

	assert.Nil(t, response)
	assert.ErrorContains(t, err, "not allowed")
}
