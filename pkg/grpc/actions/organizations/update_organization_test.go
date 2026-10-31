package organizations

import (
	"context"
	"sync"
	"testing"
	"time"

	uuid "github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/productanalytics"
	protos "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/test/support"
	"google.golang.org/grpc/codes"
	"gorm.io/gorm"
)

func Test__UpdateOrganization(t *testing.T) {
	r := support.Setup(t)

	t.Run("organization does not exist -> error", func(t *testing.T) {
		organization := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name: "updated-name",
			},
		}

		_, err := UpdateOrganization(context.Background(), uuid.New().String(), organization)
		require.Error(t, err)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.NotFound, code)
		assert.Equal(t, "organization not found", msg)
	})

	t.Run("update organization by ID -> success", func(t *testing.T) {
		updatedOrg := &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name:        "updated-org",
				Description: "Updated description",
			},
		}

		response, err := UpdateOrganization(context.Background(), r.Organization.ID.String(), updatedOrg)
		require.NoError(t, err)
		require.NotNil(t, response)
		require.NotNil(t, response.Organization)
		require.NotNil(t, response.Organization.Metadata)
		assert.Equal(t, r.Organization.ID.String(), response.Organization.Metadata.Id)
		assert.Equal(t, "updated-org", response.Organization.Metadata.Name)
		assert.Equal(t, "Updated description", response.Organization.Metadata.Description)
		assert.Equal(t, *r.Organization.CreatedAt, response.Organization.Metadata.CreatedAt.AsTime())
		assert.True(t, response.Organization.Metadata.UpdatedAt.AsTime().After(*r.Organization.UpdatedAt))
		require.NotNil(t, response.Organization.Spec)
	})

	t.Run("nil organization -> error", func(t *testing.T) {
		_, err := UpdateOrganization(context.Background(), uuid.New().String(), nil)
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "organization is required", msg)
	})

	t.Run("nil organization metadata -> error", func(t *testing.T) {
		_, err := UpdateOrganization(context.Background(), uuid.New().String(), &protos.Organization{})
		code, msg, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
		assert.Equal(t, "organization metadata is required", msg)
	})

	t.Run("update organization by slug -> success", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		response, err := UpdateOrganization(context.Background(), org.Slug, &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Description: "updated via slug",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, response.Organization)
		assert.Equal(t, org.ID.String(), response.Organization.Metadata.Id)
		assert.Equal(t, "updated via slug", response.Organization.Metadata.Description)
	})

	t.Run("update slug -> success", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		response, err := UpdateOrganization(context.Background(), org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Slug: "my-new-slug",
			},
		})
		require.NoError(t, err)
		require.NotNil(t, response.Organization)
		assert.Equal(t, "my-new-slug", response.Organization.Metadata.Slug)

		reloaded, err := models.FindOrganizationBySlug(database.Conn(), "my-new-slug")
		require.NoError(t, err)
		assert.Equal(t, org.ID, reloaded.ID)
	})

	t.Run("update name to one already in use by another organization -> success", func(t *testing.T) {
		// Only the slug must be unique among active organizations. Names may
		// repeat, for example when onboarding keeps an organization's name
		// equal to its GitHub owner across slug-collision retries.
		shared, err := models.CreateOrganization("Shared Organization Name", "")
		require.NoError(t, err)

		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		response, err := UpdateOrganization(context.Background(), org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name: shared.Name,
			},
		})
		require.NoError(t, err)
		require.NotNil(t, response.Organization)
		assert.Equal(t, shared.Name, response.Organization.Metadata.Name)
		assert.NotEqual(t, shared.Slug, response.Organization.Metadata.Slug)
	})

	t.Run("update slug to one already in use -> invalid argument", func(t *testing.T) {
		taken, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		_, err = UpdateOrganization(context.Background(), org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Slug: taken.Slug,
			},
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("update slug to a reserved word -> invalid argument", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		_, err = UpdateOrganization(context.Background(), org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Slug: "admin",
			},
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("update slug with invalid characters -> invalid argument", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		_, err = UpdateOrganization(context.Background(), org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Slug: "Not A Slug!",
			},
		})
		require.Error(t, err)
		code, _, ok := grpcerrors.HandlerStatus(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, code)
	})

	t.Run("update slug to empty -> keeps existing slug", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		response, err := UpdateOrganization(context.Background(), org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Description: "no slug change",
			},
		})
		require.NoError(t, err)
		assert.Equal(t, org.Slug, response.Organization.Metadata.Slug)
	})
}

type organizationGroupRecorder struct {
	mu     sync.Mutex
	groups []productanalytics.OrganizationGroupIdentity
}

func TestUpdateOrganizationPreservesOverlappingSuspensionTransition(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())
	for _, action := range []string{models.OrganizationSuspensionActionSuspend, models.OrganizationSuspensionActionReactivate} {
		t.Run(action, func(t *testing.T) {
			org, err := models.CreateOrganization(support.RandomName("org"), "")
			require.NoError(t, err)
			if action == models.OrganizationSuspensionActionReactivate {
				require.NoError(t, org.Suspend(db, r.Account.ID, "review suspension"))
				require.NoError(t, db.Model(&models.OrganizationSuspensionEvent{}).
					Where("organization_id = ?", org.ID).
					Update("cancellation_completed_at", time.Now()).Error)
			}

			// Commit the transition after the settings request reads its snapshot.
			transitioned := false
			callback := "test:organization_settings_overlap"
			require.NoError(t, db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
				loaded, ok := tx.Statement.Dest.(*models.Organization)
				if !ok || loaded.ID != org.ID || transitioned {
					return
				}
				transitioned = true
				if action == models.OrganizationSuspensionActionSuspend {
					tx.AddError(org.Suspend(db, r.Account.ID, "review suspension"))
					return
				}
				tx.AddError(org.Reactivate(db, r.Account.ID))
			}))
			t.Cleanup(func() { _ = db.Callback().Query().Remove(callback) })

			response, err := UpdateOrganization(t.Context(), org.ID.String(), &protos.Organization{
				Metadata: &protos.Organization_Metadata{
					Name: "Updated organization", Description: "Updated description", Slug: support.RandomName("updated"),
				},
			})
			require.NoError(t, err)
			require.True(t, transitioned, "transition must overlap the settings request")

			stored, err := models.FindOrganizationByID(org.ID.String())
			require.NoError(t, err)
			assert.Equal(t, response.Organization.Metadata.Name, stored.Name)
			assert.Equal(t, response.Organization.Metadata.Description, stored.Description)
			assert.Equal(t, response.Organization.Metadata.Slug, stored.Slug)
			if org.SuspendedAt == nil {
				assert.Nil(t, stored.SuspendedAt)
			} else {
				require.NotNil(t, stored.SuspendedAt)
				assert.True(t, org.SuspendedAt.Equal(*stored.SuspendedAt))
			}
			assert.Equal(t, org.SuspendedByAccountID, stored.SuspendedByAccountID)
			assert.Equal(t, org.SuspensionReason, stored.SuspensionReason)

			events, err := models.ListOrganizationSuspensionEvents(db, org.ID)
			require.NoError(t, err)
			if action == models.OrganizationSuspensionActionSuspend {
				require.Len(t, events, 1)
				assert.Nil(t, events[0].CancellationCompletedAt)
			} else {
				require.Len(t, events, 2)
				assert.NotNil(t, events[1].CancellationCompletedAt)
			}
			assert.Equal(t, action, events[0].Action)
			assert.Equal(t, r.Account.ID, events[0].AccountID)
			assert.Equal(t, org.SuspensionReason, events[0].Reason)
		})
	}
}

func (r *organizationGroupRecorder) Capture(productanalytics.Event) {}

func (r *organizationGroupRecorder) IdentifyOrganizationGroup(identity productanalytics.OrganizationGroupIdentity) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.groups = append(r.groups, identity)
}

func (r *organizationGroupRecorder) Close() error { return nil }

func (r *organizationGroupRecorder) Enabled() bool { return true }

func (r *organizationGroupRecorder) snapshot() []productanalytics.OrganizationGroupIdentity {
	r.mu.Lock()
	defer r.mu.Unlock()
	copied := make([]productanalytics.OrganizationGroupIdentity, len(r.groups))
	copy(copied, r.groups)
	return copied
}

func (r *organizationGroupRecorder) reset() {
	r.mu.Lock()
	r.groups = nil
	r.mu.Unlock()
}

func TestUpdateOrganizationIdentifiesGroupWhenNameOrSlugChanges(t *testing.T) {
	r := support.Setup(t)
	recorder := &organizationGroupRecorder{}
	previous := productanalytics.SetCurrent(recorder)
	t.Cleanup(func() {
		productanalytics.SetCurrent(previous)
	})
	ctx := authentication.SetAccountIDInMetadata(context.Background(), r.Account.ID.String())

	t.Run("name change identifies the group", func(t *testing.T) {
		recorder.reset()
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		_, err = UpdateOrganization(ctx, org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Name: "Renamed Organization",
			},
		})
		require.NoError(t, err)

		groups := recorder.snapshot()
		require.Len(t, groups, 1)
		assert.Equal(t, org.ID.String(), groups[0].ID)
		assert.Equal(t, "Renamed Organization", groups[0].Name)
		assert.Equal(t, org.Slug, groups[0].Slug)
		assert.Equal(t, r.Account.ID.String(), groups[0].AccountID)
	})

	t.Run("slug change identifies the group", func(t *testing.T) {
		recorder.reset()
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)
		nextSlug := support.RandomName("slug")

		_, err = UpdateOrganization(ctx, org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Slug: nextSlug,
			},
		})
		require.NoError(t, err)

		groups := recorder.snapshot()
		require.Len(t, groups, 1)
		assert.Equal(t, org.ID.String(), groups[0].ID)
		assert.Equal(t, org.Name, groups[0].Name)
		assert.Equal(t, nextSlug, groups[0].Slug)
		assert.Equal(t, r.Account.ID.String(), groups[0].AccountID)
	})

	t.Run("description-only save does not identify the group", func(t *testing.T) {
		recorder.reset()
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)

		_, err = UpdateOrganization(ctx, org.ID.String(), &protos.Organization{
			Metadata: &protos.Organization_Metadata{
				Description: "only the description",
			},
		})
		require.NoError(t, err)

		assert.Empty(t, recorder.snapshot())
	})
}
