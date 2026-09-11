package services

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	FeedbackCategoryBug     = "bug"
	FeedbackCategoryFeature = "feature"
	FeedbackCategoryOther   = "other"

	DefaultSupportFeedbackToEmail       = "support@superplane.com"
	MaxSupportFeedbackDetailsRunes      = 8000
	MaxSupportFeedbackAttachmentBytes   = 5 << 20
	MaxSupportFeedbackPagePathRunes     = 500
	MaxSupportFeedbackAttachmentNameLen = 255
)

var (
	ErrFeedbackCategoryInvalid    = errors.New("feedback category is invalid")
	ErrFeedbackDetailsRequired    = errors.New("feedback details are required")
	ErrFeedbackDetailsTooLong     = errors.New("feedback details are too long")
	ErrFeedbackAttachmentTooLarge = errors.New("feedback attachment is too large")
	ErrFeedbackAttachmentType     = errors.New("feedback attachment type is not allowed")
	ErrFeedbackAttachmentName     = errors.New("feedback attachment name is invalid")
)

var allowedFeedbackAttachmentTypes = []string{
	"image/png",
	"image/jpeg",
	"image/gif",
	"image/webp",
	"application/pdf",
	"text/plain",
	"text/markdown",
}

func SupportFeedbackToEmail() string {
	if email := strings.TrimSpace(os.Getenv("SUPPORT_FEEDBACK_TO_EMAIL")); email != "" {
		return email
	}
	return DefaultSupportFeedbackToEmail
}

func NormalizeFeedbackCategory(category string) (string, error) {
	normalized := strings.TrimSpace(category)
	switch normalized {
	case FeedbackCategoryBug, FeedbackCategoryFeature, FeedbackCategoryOther:
		return normalized, nil
	default:
		return "", ErrFeedbackCategoryInvalid
	}
}

func FeedbackCategoryLabel(category string) string {
	switch category {
	case FeedbackCategoryBug:
		return "Bug report"
	case FeedbackCategoryFeature:
		return "Feature request"
	case FeedbackCategoryOther:
		return "Other feedback"
	default:
		return "Feedback"
	}
}

type SupportFeedbackAttachment struct {
	Filename    string
	ContentType string
	Content     []byte
}

type SupportFeedback struct {
	Category         string
	Details          string
	UserName         string
	UserEmail        string
	OrganizationID   string
	OrganizationName string
	PagePath         string
	Attachment       *SupportFeedbackAttachment
}

type SupportFeedbackTemplateData struct {
	CategoryLabel    string
	Details          string
	FromLine         string
	UserName         string
	UserEmail        string
	OrganizationName string
	OrganizationID   string
	PagePath         string
	AttachmentName   string
}

func (f SupportFeedback) TemplateData() SupportFeedbackTemplateData {
	attachmentName := ""
	if f.Attachment != nil {
		attachmentName = f.Attachment.Filename
	}

	return SupportFeedbackTemplateData{
		CategoryLabel:    FeedbackCategoryLabel(f.Category),
		Details:          f.Details,
		FromLine:         feedbackFromLine(f.UserName, f.UserEmail),
		UserName:         f.UserName,
		UserEmail:        f.UserEmail,
		OrganizationName: f.OrganizationName,
		OrganizationID:   f.OrganizationID,
		PagePath:         f.PagePath,
		AttachmentName:   attachmentName,
	}
}

func feedbackFromLine(name, email string) string {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	switch {
	case name != "" && email != "":
		return name + " <" + email + ">"
	case email != "":
		return email
	default:
		return name
	}
}

func (f SupportFeedback) EmailSubject() string {
	from := strings.TrimSpace(f.UserEmail)
	if from == "" {
		from = strings.TrimSpace(f.UserName)
	}
	if from == "" {
		return "[SuperPlane] " + FeedbackCategoryLabel(f.Category)
	}
	return "[SuperPlane] " + FeedbackCategoryLabel(f.Category) + " from " + from
}

func NormalizeFeedbackDetails(details string) (string, error) {
	trimmed := strings.TrimSpace(details)
	if trimmed == "" {
		return "", ErrFeedbackDetailsRequired
	}
	if utf8.RuneCountInString(trimmed) > MaxSupportFeedbackDetailsRunes {
		return "", ErrFeedbackDetailsTooLong
	}
	return trimmed, nil
}

func NormalizeFeedbackPagePath(pagePath string) string {
	trimmed := strings.TrimSpace(pagePath)
	if trimmed == "" {
		return ""
	}
	if utf8.RuneCountInString(trimmed) > MaxSupportFeedbackPagePathRunes {
		return string([]rune(trimmed)[:MaxSupportFeedbackPagePathRunes])
	}
	return trimmed
}

func NormalizeFeedbackAttachment(filename, contentType string, content []byte) (*SupportFeedbackAttachment, error) {
	if len(content) == 0 && strings.TrimSpace(filename) == "" {
		return nil, nil
	}
	if len(content) == 0 {
		return nil, ErrFeedbackAttachmentName
	}
	if len(content) > MaxSupportFeedbackAttachmentBytes {
		return nil, ErrFeedbackAttachmentTooLarge
	}

	name := sanitizeAttachmentFilename(filename)
	if name == "" {
		return nil, ErrFeedbackAttachmentName
	}

	normalizedType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	if !slices.Contains(allowedFeedbackAttachmentTypes, normalizedType) {
		return nil, ErrFeedbackAttachmentType
	}

	return &SupportFeedbackAttachment{
		Filename:    name,
		ContentType: normalizedType,
		Content:     content,
	}, nil
}

func sanitizeAttachmentFilename(filename string) string {
	normalized := strings.ReplaceAll(filename, "\\", "/")
	name := filepath.Base(normalized)
	name = strings.TrimSpace(name)
	name = strings.NewReplacer("\r", "", "\n", "").Replace(name)
	if name == "" || name == "." || name == ".." {
		return ""
	}
	if utf8.RuneCountInString(name) > MaxSupportFeedbackAttachmentNameLen {
		name = string([]rune(name)[:MaxSupportFeedbackAttachmentNameLen])
	}
	return name
}
