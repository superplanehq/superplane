package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeFeedbackCategory(t *testing.T) {
	t.Run("accepts known categories", func(t *testing.T) {
		for _, category := range []string{FeedbackCategoryBug, FeedbackCategoryFeature, FeedbackCategoryOther} {
			got, err := NormalizeFeedbackCategory(" " + category + " ")
			require.NoError(t, err)
			assert.Equal(t, category, got)
		}
	})

	t.Run("rejects unknown categories", func(t *testing.T) {
		_, err := NormalizeFeedbackCategory("praise")
		assert.ErrorIs(t, err, ErrFeedbackCategoryInvalid)
	})
}

func TestNormalizeFeedbackDetails(t *testing.T) {
	t.Run("requires non-empty details", func(t *testing.T) {
		_, err := NormalizeFeedbackDetails("  \n")
		assert.ErrorIs(t, err, ErrFeedbackDetailsRequired)
	})

	t.Run("rejects details that exceed the rune limit", func(t *testing.T) {
		_, err := NormalizeFeedbackDetails(strings.Repeat("a", MaxSupportFeedbackDetailsRunes+1))
		assert.ErrorIs(t, err, ErrFeedbackDetailsTooLong)
	})

	t.Run("trims details", func(t *testing.T) {
		got, err := NormalizeFeedbackDetails("  the canvas did not load  ")
		require.NoError(t, err)
		assert.Equal(t, "the canvas did not load", got)
	})
}

func TestNormalizeFeedbackAttachment(t *testing.T) {
	t.Run("returns nil when no file is present", func(t *testing.T) {
		attachment, err := NormalizeFeedbackAttachment("", "", nil)
		require.NoError(t, err)
		assert.Nil(t, attachment)
	})

	t.Run("rejects empty content with a name", func(t *testing.T) {
		_, err := NormalizeFeedbackAttachment("shot.png", "image/png", nil)
		assert.ErrorIs(t, err, ErrFeedbackAttachmentName)
	})

	t.Run("rejects files that exceed the size limit", func(t *testing.T) {
		_, err := NormalizeFeedbackAttachment("shot.png", "image/png", make([]byte, MaxSupportFeedbackAttachmentBytes+1))
		assert.ErrorIs(t, err, ErrFeedbackAttachmentTooLarge)
	})

	t.Run("rejects disallowed types", func(t *testing.T) {
		_, err := NormalizeFeedbackAttachment("payload.exe", "application/octet-stream", []byte("x"))
		assert.ErrorIs(t, err, ErrFeedbackAttachmentType)
	})

	t.Run("sanitizes the filename and content type", func(t *testing.T) {
		attachment, err := NormalizeFeedbackAttachment(`C:\tmp\shot.png`, "image/png; charset=binary", []byte("png"))
		require.NoError(t, err)
		require.NotNil(t, attachment)
		assert.Equal(t, "shot.png", attachment.Filename)
		assert.Equal(t, "image/png", attachment.ContentType)
		assert.Equal(t, []byte("png"), attachment.Content)
	})
}

func TestSupportFeedbackEmailSubject(t *testing.T) {
	feedback := SupportFeedback{Category: FeedbackCategoryBug, UserEmail: "ada@example.com"}
	assert.Equal(t, "[SuperPlane] Bug report from ada@example.com", feedback.EmailSubject())
}

func TestSupportFeedbackToEmail(t *testing.T) {
	assert.Equal(t, DefaultSupportFeedbackToEmail, SupportFeedbackToEmail())

	t.Setenv("SUPPORT_FEEDBACK_TO_EMAIL", " ops@example.com ")
	assert.Equal(t, "ops@example.com", SupportFeedbackToEmail())
}
