package logs

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const activeLogReadSkew = 30 * time.Second

func SignActiveLogRead(secret string, taskID uuid.UUID, cursor string, now time.Time) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("active log read secret is required")
	}
	if taskID == uuid.Nil {
		return "", errors.New("task ID is required")
	}
	issued := now.Unix()
	mac := activeLogReadMAC(secret, taskID, cursor, issued)
	return strconv.FormatInt(issued, 10) + "." + hex.EncodeToString(mac.Sum(nil)), nil
}

func VerifyActiveLogRead(secret, authorization string, taskID uuid.UUID, cursor string, now time.Time) error {
	token, ok := strings.CutPrefix(strings.TrimSpace(authorization), "Bearer ")
	if !ok || token == "" || strings.TrimSpace(secret) == "" || taskID == uuid.Nil {
		return errors.New("unauthorized")
	}
	issuedText, macText, ok := strings.Cut(token, ".")
	if !ok {
		return errors.New("unauthorized")
	}
	issued, err := strconv.ParseInt(issuedText, 10, 64)
	if err != nil {
		return errors.New("unauthorized")
	}
	delta := now.Sub(time.Unix(issued, 0))
	if delta < 0 {
		delta = -delta
	}
	if delta > activeLogReadSkew {
		return errors.New("unauthorized")
	}
	got, err := hex.DecodeString(macText)
	if err != nil {
		return errors.New("unauthorized")
	}
	expected := activeLogReadMAC(secret, taskID, cursor, issued).Sum(nil)
	if !hmac.Equal(got, expected) {
		return errors.New("unauthorized")
	}
	return nil
}

func activeLogReadMAC(secret string, taskID uuid.UUID, cursor string, issued int64) hash.Hash {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = io.WriteString(mac, taskID.String())
	_, _ = io.WriteString(mac, "\n")
	_, _ = io.WriteString(mac, cursor)
	_, _ = io.WriteString(mac, "\n")
	_, _ = io.WriteString(mac, strconv.FormatInt(issued, 10))
	return mac
}
