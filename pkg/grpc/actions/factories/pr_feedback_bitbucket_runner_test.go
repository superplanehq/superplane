package factories

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBitbucketCheckoutCommandChecksOutTheSourceBranch(t *testing.T) {
	command := bitbucketCheckoutCommand()

	assert.Contains(t, command, `git checkout -B "${SOURCE_BRANCH}" "${SOURCE_HASH}"`)
	assert.NotContains(t, command, `git checkout -B --`)
	assert.Contains(t, command, `git check-ref-format --branch "${SOURCE_BRANCH}"`)
}
