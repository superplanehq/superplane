package messages

import "encoding/json"

const MagicCodeRequestedRoutingKey = "magic-code-requested"

type MagicCodeRequestedMessage struct {
	Email          string `json:"email"`
	Code           string `json:"code"`
	MagicLinkToken string `json:"magic_link_token"`
	RedirectURL    string `json:"redirect_url,omitempty"`
	SignupIntent   bool   `json:"signup_intent,omitempty"`
}

func NewMagicCodeRequestedMessage(email, code, magicLinkToken, redirectURL string, signupIntent bool) MagicCodeRequestedMessage {
	return MagicCodeRequestedMessage{
		Email:          email,
		Code:           code,
		MagicLinkToken: magicLinkToken,
		RedirectURL:    redirectURL,
		SignupIntent:   signupIntent,
	}
}

func (m MagicCodeRequestedMessage) Publish() error {
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return Publish(CanvasExchange, MagicCodeRequestedRoutingKey, body)
}
