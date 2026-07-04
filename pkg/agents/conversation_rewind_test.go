package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/superplanehq/superplane/pkg/models"
)

func TestFormatRewindEntry_NotesImageOnlyUserTurn(t *testing.T) {
	entry := formatRewindEntry(models.AgentSessionMessage{
		Role:   models.AgentMessageRoleUser,
		Images: imageSlice(1),
	})

	assert.Equal(t, "User: [1 image attachment shared earlier]", entry)
}

func TestFormatRewindEntry_NotesImagesAlongsideText(t *testing.T) {
	entry := formatRewindEntry(models.AgentSessionMessage{
		Role:    models.AgentMessageRoleUser,
		Content: "look at this",
		Images:  imageSlice(2),
	})

	assert.Equal(t, "User: look at this [2 image attachments shared earlier]", entry)
}

func TestFormatRewindEntry_OmitsNoteWithoutImages(t *testing.T) {
	entry := formatRewindEntry(models.AgentSessionMessage{
		Role:    models.AgentMessageRoleUser,
		Content: "no attachments here",
	})

	assert.Equal(t, "User: no attachments here", entry)
}

func imageSlice(count int) []models.AgentSessionImage {
	images := make([]models.AgentSessionImage, 0, count)
	for i := 0; i < count; i++ {
		images = append(images, models.AgentSessionImage{MediaType: "image/png", Data: "aGVsbG8="})
	}
	return images
}
