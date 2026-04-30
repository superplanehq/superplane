package serviceaccounts

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func TestUserRefFromCreator(t *testing.T) {
	id := uuid.New()

	t.Run("active user with name", func(t *testing.T) {
		ref := userRefFromCreator(&models.User{ID: id, Name: "Alice"})
		require.Equal(t, id.String(), ref.Id)
		require.Equal(t, "Alice", ref.Name)
	})

	t.Run("soft-deleted user with empty name", func(t *testing.T) {
		ref := userRefFromCreator(&models.User{
			ID:        id,
			Name:      "",
			DeletedAt: gorm.DeletedAt{Time: time.Now(), Valid: true},
		})
		require.Equal(t, id.String(), ref.Id)
		require.Equal(t, "Former member", ref.Name)
	})

	t.Run("empty name without deletion", func(t *testing.T) {
		ref := userRefFromCreator(&models.User{ID: id, Name: ""})
		require.Equal(t, "Unknown", ref.Name)
	})
}
