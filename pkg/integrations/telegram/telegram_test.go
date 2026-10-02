package telegram

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__Telegram__MentionsBot(t *testing.T) {
	t.Run("bot mentioned -> true", func(t *testing.T) {
		message := &TelegramMessage{
			Text:     "@mybot hello!",
			Entities: []MessageEntity{{Type: "mention", Offset: 0, Length: 6}},
		}

		assert.True(t, mentionsBot(message, "mybot"))
	})

	t.Run("bot mentioned with different casing -> true", func(t *testing.T) {
		message := &TelegramMessage{
			Text:     "@MyBot hello!",
			Entities: []MessageEntity{{Type: "mention", Offset: 0, Length: 6}},
		}

		assert.True(t, mentionsBot(message, "mybot"))
	})

	t.Run("another bot mentioned -> false", func(t *testing.T) {
		message := &TelegramMessage{
			Text:     "@otherbot hello!",
			Entities: []MessageEntity{{Type: "mention", Offset: 0, Length: 9}},
		}

		assert.False(t, mentionsBot(message, "mybot"))
	})

	t.Run("non-mention entity -> false", func(t *testing.T) {
		message := &TelegramMessage{
			Text:     "/start@mybot",
			Entities: []MessageEntity{{Type: "bot_command", Offset: 0, Length: 12}},
		}

		assert.False(t, mentionsBot(message, "mybot"))
	})

	t.Run("bot without a username -> false", func(t *testing.T) {
		message := &TelegramMessage{
			Text:     "@mybot hello!",
			Entities: []MessageEntity{{Type: "mention", Offset: 0, Length: 6}},
		}

		assert.False(t, mentionsBot(message, ""))
	})

	//
	// Telegram reports offsets in UTF-16 code units, so every character before the
	// mention that is not a single UTF-8 byte shifts a byte-indexed slice.
	//
	t.Run("mention after a multi-byte character -> true", func(t *testing.T) {
		message := &TelegramMessage{
			Text:     "héllo @mybot",
			Entities: []MessageEntity{{Type: "mention", Offset: 6, Length: 6}},
		}

		assert.True(t, mentionsBot(message, "mybot"))
	})

	t.Run("mention after an emoji -> true", func(t *testing.T) {
		message := &TelegramMessage{
			Text:     "👋 @mybot",
			Entities: []MessageEntity{{Type: "mention", Offset: 3, Length: 6}},
		}

		assert.True(t, mentionsBot(message, "mybot"))
	})

	t.Run("mention after a CJK character -> true", func(t *testing.T) {
		message := &TelegramMessage{
			Text:     "こんにちは @mybot",
			Entities: []MessageEntity{{Type: "mention", Offset: 6, Length: 6}},
		}

		assert.True(t, mentionsBot(message, "mybot"))
	})

	t.Run("entity out of range -> false", func(t *testing.T) {
		message := &TelegramMessage{
			Text:     "hi",
			Entities: []MessageEntity{{Type: "mention", Offset: 0, Length: 100}},
		}

		assert.False(t, mentionsBot(message, "mybot"))
	})

	t.Run("negative entity offset -> false", func(t *testing.T) {
		message := &TelegramMessage{
			Text:     "@mybot",
			Entities: []MessageEntity{{Type: "mention", Offset: -1, Length: 6}},
		}

		assert.False(t, mentionsBot(message, "mybot"))
	})
}

func Test__Telegram__HandleRequest(t *testing.T) {
	integration := &Telegram{}

	handle := func(t *testing.T, body string) *httptest.ResponseRecorder {
		t.Helper()

		request := httptest.NewRequest(http.MethodPost, "/api/v1/integrations/test/events", bytes.NewBufferString(body))
		recorder := httptest.NewRecorder()

		integration.HandleRequest(core.HTTPRequestContext{
			Logger:   logrus.NewEntry(logrus.New()),
			Request:  request,
			Response: recorder,
			Integration: &contexts.IntegrationContext{
				Metadata: map[string]any{"botId": float64(42), "username": "mybot"},
			},
		})

		return recorder
	}

	t.Run("invalid body -> 400", func(t *testing.T) {
		assert.Equal(t, http.StatusBadRequest, handle(t, "not json").Code)
	})

	t.Run("update without a message -> 200", func(t *testing.T) {
		assert.Equal(t, http.StatusOK, handle(t, `{"update_id": 1}`).Code)
	})

	t.Run("message without a mention -> 200", func(t *testing.T) {
		body := `{"update_id": 1, "message": {"message_id": 1, "text": "hello", "chat": {"id": 1, "type": "group"}}}`

		assert.Equal(t, http.StatusOK, handle(t, body).Code)
	})

	t.Run("entity out of range -> 200", func(t *testing.T) {
		body := `{
			"update_id": 1,
			"message": {
				"message_id": 1,
				"text": "hi",
				"chat": {"id": 1, "type": "group"},
				"entities": [{"type": "mention", "offset": 0, "length": 100}]
			}
		}`

		assert.Equal(t, http.StatusOK, handle(t, body).Code)
	})
}
