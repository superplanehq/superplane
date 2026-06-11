package public

import (
	"encoding/base64"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/public/middleware"
)

// handleAgentChatMessageImage streams a single image attached to an agent chat
// message. Image bytes are served out-of-band (rather than embedded as base64
// in the message list response) so chat history stays small enough to fit under
// the gRPC/HTTP response size limits.
func (s *Server) handleAgentChatMessageImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	sessionID, err := uuid.Parse(vars["chatId"])
	if err != nil {
		http.Error(w, "invalid chat id", http.StatusBadRequest)
		return
	}
	messageID, err := uuid.Parse(vars["messageId"])
	if err != nil {
		http.Error(w, "invalid message id", http.StatusBadRequest)
		return
	}
	index, err := strconv.Atoi(vars["index"])
	if err != nil || index < 0 {
		http.Error(w, "invalid image index", http.StatusBadRequest)
		return
	}

	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Agent sessions are private to their creator, so this both authorizes the
	// request and scopes the lookup to the caller.
	if _, err := models.FindAgentSessionForUser(user.OrganizationID, user.ID, sessionID); err != nil {
		http.Error(w, "agent chat not found", http.StatusNotFound)
		return
	}

	message, err := models.FindAgentSessionMessage(messageID)
	if err != nil || message.SessionID != sessionID {
		http.Error(w, "message not found", http.StatusNotFound)
		return
	}

	if index >= len(message.Images) {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	image := message.Images[index]
	data, err := base64.StdEncoding.DecodeString(image.Data)
	if err != nil {
		log.Errorf("failed to decode agent chat message image %s[%d]: %v", messageID, index, err)
		http.Error(w, "invalid image data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", image.MediaType)
	w.Header().Set("Cache-Control", "private, max-age=86400, immutable")
	if _, err := w.Write(data); err != nil {
		log.Errorf("failed to write agent chat message image %s[%d]: %v", messageID, index, err)
	}
}
