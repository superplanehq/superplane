package livelogstoken

import (
	"fmt"
	"strings"

	gojwt "github.com/golang-jwt/jwt/v4"
)

const (
	Purpose  = "runner_live_logs"
	Audience = "task_broker"
)

type Claims struct {
	TaskID  string `json:"task_id"`
	Purpose string `json:"purpose"`
	gojwt.RegisteredClaims
}

func Validate(tokenString, brokerTaskID, secret string) error {
	brokerTaskID = strings.TrimSpace(brokerTaskID)
	if brokerTaskID == "" {
		return fmt.Errorf("broker task id is empty")
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return fmt.Errorf("jwt secret is empty")
	}

	claims := &Claims{}
	token, err := gojwt.ParseWithClaims(tokenString, claims, func(token *gojwt.Token) (any, error) {
		if _, ok := token.Method.(*gojwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return err
	}
	if !token.Valid {
		return fmt.Errorf("invalid token")
	}
	if claims.Purpose != Purpose {
		return fmt.Errorf("invalid purpose")
	}
	if !claims.VerifyAudience(Audience, true) {
		return fmt.Errorf("invalid audience")
	}
	if strings.TrimSpace(claims.TaskID) != brokerTaskID {
		return fmt.Errorf("task id mismatch")
	}
	return nil
}
