package githubapps

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

var adjectives = []string{
	"amber", "azure", "bold", "brisk", "calm", "clever", "crisp", "daring",
	"gentle", "golden", "green", "lucky", "mellow", "nimble", "quiet", "rapid",
	"silver", "steady", "swift", "vivid",
}

var nouns = []string{
	"brook", "cloud", "coral", "ember", "falcon", "forest", "harbor", "meadow",
	"nova", "orchid", "peak", "pine", "river", "rose", "spark", "stone", "wave",
}

// GenerateInstallationName returns a short random name such as green-rose-57383.
func GenerateInstallationName() (string, error) {
	adjective, err := randomWord(adjectives)
	if err != nil {
		return "", err
	}

	noun, err := randomWord(nouns)
	if err != nil {
		return "", err
	}

	suffix, err := rand.Int(rand.Reader, big.NewInt(90000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s-%d", adjective, noun, suffix.Int64()+10000), nil
}

func randomWord(words []string) (string, error) {
	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
	if err != nil {
		return "", err
	}

	return words[index.Int64()], nil
}
