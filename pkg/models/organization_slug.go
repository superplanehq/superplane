package models

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"

	uuid "github.com/google/uuid"
	"gorm.io/gorm"
)

// maxOrganizationSlugLength caps generated slugs so they stay readable in a
// URL and comfortably under typical path-segment limits.
const maxOrganizationSlugLength = 63

// defaultOrganizationSlug is used when a name slugifies to an empty string
// (for example, a name made up entirely of punctuation or non-ASCII
// characters), so every organization always has a non-empty slug.
const defaultOrganizationSlug = "org"

// reservedOrganizationSlugs are top-level path segments the frontend router
// treats as application routes (see web_src/src/lib/reservedAppPaths.ts) plus
// infrastructure roots served outside the SPA. A slug equal to one of these
// would shadow that route, so it can never be generated or accepted.
var reservedOrganizationSlugs = []string{
	// Frontend application routes.
	"admin",
	"login",
	"signup",
	"welcome",
	"onboarding",
	"create",
	"setup",
	"invite",
	"install",
	"github",
	// Infrastructure roots served outside the SPA.
	"api",
	"health",
	"assets",
	"logout",
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// IsReservedOrganizationSlug reports whether slug is a reserved top-level
// path segment that must never be used as an organization slug.
func IsReservedOrganizationSlug(slug string) bool {
	return slices.Contains(reservedOrganizationSlugs, slug)
}

// Slugify converts name into a lowercase, URL-friendly slug: it strips
// accents to their closest ASCII letter, replaces runs of characters outside
// [a-z0-9] with a single dash, trims leading/trailing dashes, and caps the
// result at maxOrganizationSlugLength characters. It returns
// defaultOrganizationSlug when name has no sluggable characters at all.
func Slugify(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
		case r < unicode.MaxASCII:
			b.WriteByte('-')
		default:
			if ascii, ok := transliterateRune(r); ok {
				b.WriteString(ascii)
			} else {
				b.WriteByte('-')
			}
		}
	}

	slug := nonSlugChars.ReplaceAllString(b.String(), "-")
	slug = strings.Trim(slug, "-")

	if len(slug) > maxOrganizationSlugLength {
		slug = strings.Trim(slug[:maxOrganizationSlugLength], "-")
	}

	if slug == "" {
		return defaultOrganizationSlug
	}

	return slug
}

// transliterateRune maps a small set of common accented Latin letters to
// their plain ASCII equivalent, so names like "Café Org" slugify to
// "cafe-org" instead of dropping the letter entirely. Runes with no mapping
// return ok=false, and the caller falls back to a dash separator.
func transliterateRune(r rune) (string, bool) {
	switch r {
	case 'à', 'á', 'â', 'ã', 'ä', 'å':
		return "a", true
	case 'è', 'é', 'ê', 'ë':
		return "e", true
	case 'ì', 'í', 'î', 'ï':
		return "i", true
	case 'ò', 'ó', 'ô', 'õ', 'ö':
		return "o", true
	case 'ù', 'ú', 'û', 'ü':
		return "u", true
	case 'ý', 'ÿ':
		return "y", true
	case 'ñ':
		return "n", true
	case 'ç':
		return "c", true
	default:
		return "", false
	}
}

// GenerateUniqueOrganizationSlug slugifies base and returns a slug that is
// not currently used by another active (non-deleted) organization. It
// appends "-2", "-3", and so on until it finds one that is free. Pass
// uuid.Nil for excludeID when creating a new organization, or the
// organization's own ID when regenerating a slug on update so the
// organization does not collide with itself.
func GenerateUniqueOrganizationSlug(tx *gorm.DB, base string, excludeID uuid.UUID) (string, error) {
	candidate := Slugify(base)
	if IsReservedOrganizationSlug(candidate) {
		candidate = fmt.Sprintf("%s-org", candidate)
	}

	for suffix := 1; ; suffix++ {
		attempt := candidate
		if suffix > 1 {
			attempt = fmt.Sprintf("%s-%d", candidate, suffix)
		}

		available, err := organizationSlugAvailable(tx, attempt, excludeID)
		if err != nil {
			return "", err
		}
		if available {
			return attempt, nil
		}
	}
}

// IsOrganizationSlugAvailable reports whether slug is free to use, ignoring
// soft-deleted organizations and, when excludeID is not uuid.Nil, the
// organization with that ID. Callers that want to let the user control the
// exact slug (rather than auto-suffixing) use this before rejecting a
// requested slug as taken.
func IsOrganizationSlugAvailable(tx *gorm.DB, slug string, excludeID uuid.UUID) (bool, error) {
	return organizationSlugAvailable(tx, slug, excludeID)
}

func organizationSlugAvailable(tx *gorm.DB, slug string, excludeID uuid.UUID) (bool, error) {
	query := tx.Model(&Organization{}).Where("slug = ?", slug)
	if excludeID != uuid.Nil {
		query = query.Where("id <> ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count == 0, nil
}

// FindOrganizationBySlug returns the active organization with the given
// slug.
func FindOrganizationBySlug(tx *gorm.DB, slug string) (*Organization, error) {
	var organization Organization

	err := tx.
		Where("slug = ?", slug).
		First(&organization).
		Error

	if err != nil {
		return nil, err
	}

	return &organization, nil
}

// FindOrganizationByIDOrSlug resolves ref to an organization. ref is tried
// as a UUID first; if it does not parse as one, it is looked up as a slug.
// This lets API callers pass either identifier at the choke points where the
// client supplies an organization reference.
func FindOrganizationByIDOrSlug(tx *gorm.DB, ref string) (*Organization, error) {
	if _, err := uuid.Parse(ref); err == nil {
		return FindOrganizationByIDInTransaction(tx, ref)
	}

	return FindOrganizationBySlug(tx, ref)
}
