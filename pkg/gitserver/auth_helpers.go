package gitserver

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/superplanehq/superplane/pkg/models"
)

func hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}

func findUserByTokenHash(hash string) (string, error) {
	user, err := models.FindActiveUserByTokenHash(hash)
	if err != nil {
		return "", err
	}
	return user.Name, nil
}
