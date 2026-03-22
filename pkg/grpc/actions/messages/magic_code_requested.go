package messages

import "encoding/json"

const MagicCodeRequestedRoutingKey = "magic-code-requested"

type MagicCodeRequestedMessage struct {
	Email          string `json:"email"`
	Code           string `json:"code"`
	MagicLinkToken string `json:"magic_link_token"`
}

func NewMagicCodeRequestedMessage(email, code, magicLinkToken string) MagicCodeRequestedMessage {
	return MagicCodeRequestedMessage{
		Email:          email,
		Code:           code,
		MagicLinkToken: magicLinkToken,
	}
}

func (m MagicCodeRequestedMessage) Publish() error {
	body, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return Publish(CanvasExchange, MagicCodeRequestedRoutingKey, body)
}
