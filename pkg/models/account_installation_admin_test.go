package models

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func TestInstallationAdmin(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("new accounts are not installation admins by default", func(t *testing.T) {
		account, err := CreateAccount("Regular User", "regular@example.com")
		require.NoError(t, err)
		assert.False(t, account.IsInstallationAdmin())
	})

	t.Run("PromoteToInstallationAdmin sets the flag", func(t *testing.T) {
		account, err := CreateAccount("Admin Candidate", "candidate@example.com")
		require.NoError(t, err)
		assert.False(t, account.IsInstallationAdmin())

		err = PromoteToInstallationAdmin(account.ID.String())
		require.NoError(t, err)

		// Re-fetch from database
		refreshed, err := FindAccountByID(account.ID.String())
		require.NoError(t, err)
		assert.True(t, refreshed.IsInstallationAdmin())
	})

	t.Run("DemoteFromInstallationAdmin clears the flag", func(t *testing.T) {
		account, err := CreateAccount("Temp Admin", "temp-admin@example.com")
		require.NoError(t, err)

		err = PromoteToInstallationAdmin(account.ID.String())
		require.NoError(t, err)

		err = DemoteFromInstallationAdmin(account.ID.String())
		require.NoError(t, err)

		refreshed, err := FindAccountByID(account.ID.String())
		require.NoError(t, err)
		assert.False(t, refreshed.IsInstallationAdmin())
	})

	t.Run("IsInstallationAdmin returns correct value", func(t *testing.T) {
		nonAdmin := &Account{InstallationAdmin: false}
		assert.False(t, nonAdmin.IsInstallationAdmin())

		admin := &Account{InstallationAdmin: true}
		assert.True(t, admin.IsInstallationAdmin())
	})
}

func TestListAllOrganizations(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("returns empty list when no organizations exist", func(t *testing.T) {
		orgs, total, err := ListAllOrganizations(database.Conn(), "", 50, 0, "", "")
		require.NoError(t, err)
		assert.Empty(t, orgs)
		assert.Equal(t, int64(0), total)
	})

	t.Run("returns all organizations sorted by name", func(t *testing.T) {
		_, err := CreateOrganization("Zebra Org", "")
		require.NoError(t, err)
		_, err = CreateOrganization("Alpha Org", "")
		require.NoError(t, err)
		_, err = CreateOrganization("Middle Org", "")
		require.NoError(t, err)

		orgs, total, err := ListAllOrganizations(database.Conn(), "", 50, 0, "name", "asc")
		require.NoError(t, err)
		require.Len(t, orgs, 3)
		assert.Equal(t, int64(3), total)
		assert.Equal(t, "Alpha Org", orgs[0].Name)
		assert.Equal(t, "Middle Org", orgs[1].Name)
		assert.Equal(t, "Zebra Org", orgs[2].Name)
	})

	t.Run("sorts organizations by canvas count", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		lowCount, err := CreateOrganization("Low Count", "")
		require.NoError(t, err)
		highCount, err := CreateOrganization("High Count", "")
		require.NoError(t, err)
		midCount, err := CreateOrganization("Mid Count", "")
		require.NoError(t, err)

		createTestCanvas(t, lowCount.ID, "Low Canvas 1")
		createTestCanvas(t, highCount.ID, "High Canvas 1")
		createTestCanvas(t, highCount.ID, "High Canvas 2")
		createTestCanvas(t, highCount.ID, "High Canvas 3")
		createTestCanvas(t, midCount.ID, "Mid Canvas 1")
		createTestCanvas(t, midCount.ID, "Mid Canvas 2")

		orgs, total, err := ListAllOrganizations(database.Conn(), "", 50, 0, "canvas_count", "desc")
		require.NoError(t, err)
		require.Len(t, orgs, 3)
		assert.Equal(t, int64(3), total)
		assert.Equal(t, "High Count", orgs[0].Name)
		assert.Equal(t, "Mid Count", orgs[1].Name)
		assert.Equal(t, "Low Count", orgs[2].Name)
	})

	t.Run("sorts organizations by task count", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		lowCount, err := CreateOrganization("Low Tasks", "")
		require.NoError(t, err)
		highCount, err := CreateOrganization("High Tasks", "")
		require.NoError(t, err)
		midCount, err := CreateOrganization("Mid Tasks", "")
		require.NoError(t, err)

		db := database.DB(t.Context())
		createTestTasks(t, db, lowCount.ID, 1)
		createTestTasks(t, db, highCount.ID, 3)
		createTestTasks(t, db, midCount.ID, 2)

		orgs, total, err := ListAllOrganizations(db, "", 50, 0, "task_count", "desc")
		require.NoError(t, err)
		require.Len(t, orgs, 3)
		assert.Equal(t, int64(3), total)
		assert.Equal(t, "High Tasks", orgs[0].Name)
		assert.Equal(t, int64(3), orgs[0].TaskCount)
		assert.Equal(t, "Mid Tasks", orgs[1].Name)
		assert.Equal(t, int64(2), orgs[1].TaskCount)
		assert.Equal(t, "Low Tasks", orgs[2].Name)
		assert.Equal(t, int64(1), orgs[2].TaskCount)
		assert.Equal(t, int64(0), orgs[0].DoneTaskCount)
	})

	t.Run("sorts organizations by done task count", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		noneDone, err := CreateOrganization("None Done", "")
		require.NoError(t, err)
		mostDone, err := CreateOrganization("Most Done", "")
		require.NoError(t, err)
		someDone, err := CreateOrganization("Some Done", "")
		require.NoError(t, err)

		db := database.DB(t.Context())
		createTestTasks(t, db, noneDone.ID, 2)
		createTestDoneTasks(t, db, mostDone.ID, 2, 1, 1)
		createTestDoneTasks(t, db, someDone.ID, 1, 0, 1)

		orgs, total, err := ListAllOrganizations(db, "", 50, 0, "done_task_count", "desc")
		require.NoError(t, err)
		require.Len(t, orgs, 3)
		assert.Equal(t, int64(3), total)
		assert.Equal(t, "Most Done", orgs[0].Name)
		assert.Equal(t, int64(3), orgs[0].DoneTaskCount)
		assert.Equal(t, int64(4), orgs[0].TaskCount)
		assert.Equal(t, "Some Done", orgs[1].Name)
		assert.Equal(t, int64(1), orgs[1].DoneTaskCount)
		assert.Equal(t, int64(2), orgs[1].TaskCount)
		assert.Equal(t, "None Done", orgs[2].Name)
		assert.Equal(t, int64(0), orgs[2].DoneTaskCount)
	})

	t.Run("sorts organizations by member count", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		lowCount, err := CreateOrganization("Low Members", "")
		require.NoError(t, err)
		highCount, err := CreateOrganization("High Members", "")
		require.NoError(t, err)
		midCount, err := CreateOrganization("Mid Members", "")
		require.NoError(t, err)

		createTestUser(t, lowCount.ID, "low-1@example.com", "Low 1")
		createTestUser(t, highCount.ID, "high-1@example.com", "High 1")
		createTestUser(t, highCount.ID, "high-2@example.com", "High 2")
		createTestUser(t, highCount.ID, "high-3@example.com", "High 3")
		createTestUser(t, midCount.ID, "mid-1@example.com", "Mid 1")
		createTestUser(t, midCount.ID, "mid-2@example.com", "Mid 2")

		orgs, total, err := ListAllOrganizations(database.Conn(), "", 50, 0, "member_count", "desc")
		require.NoError(t, err)
		require.Len(t, orgs, 3)
		assert.Equal(t, int64(3), total)
		assert.Equal(t, "High Members", orgs[0].Name)
		assert.Equal(t, "Mid Members", orgs[1].Name)
		assert.Equal(t, "Low Members", orgs[2].Name)
	})

	t.Run("excludes soft-deleted organizations", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		org, err := CreateOrganization("Active Org", "")
		require.NoError(t, err)
		toDelete, err := CreateOrganization("Deleted Org", "")
		require.NoError(t, err)

		err = SoftDeleteOrganization(toDelete.ID.String())
		require.NoError(t, err)

		orgs, total, err := ListAllOrganizations(database.Conn(), "", 50, 0, "", "")
		require.NoError(t, err)
		require.Len(t, orgs, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, org.ID, orgs[0].ID)
	})

	t.Run("filters by search term", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		_, err := CreateOrganization("Alpha Corp", "")
		require.NoError(t, err)
		_, err = CreateOrganization("Beta Inc", "")
		require.NoError(t, err)

		orgs, total, err := ListAllOrganizations(database.Conn(), "alpha", 50, 0, "", "")
		require.NoError(t, err)
		require.Len(t, orgs, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Alpha Corp", orgs[0].Name)
	})

	t.Run("filters by organization id", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		target, err := CreateOrganization("Target Org", "")
		require.NoError(t, err)
		other, err := CreateOrganization("Other Org", "")
		require.NoError(t, err)
		deleted, err := CreateOrganization("Deleted Org", "")
		require.NoError(t, err)
		require.NoError(t, SoftDeleteOrganization(deleted.ID.String()))

		id := target.ID.String()
		cases := []struct {
			name   string
			search string
			want   []uuid.UUID
		}{
			{name: "full id in lower case", search: id, want: []uuid.UUID{target.ID}},
			{name: "full id in upper case", search: strings.ToUpper(id), want: []uuid.UUID{target.ID}},
			{name: "first 8 characters", search: id[:8], want: []uuid.UUID{target.ID}},
			{name: "surrounding spaces", search: "  " + id + "  ", want: []uuid.UUID{target.ID}},
			{name: "name", search: "target", want: []uuid.UUID{target.ID}},
			{name: "different organization id", search: other.ID.String(), want: []uuid.UUID{other.ID}},
			{name: "deleted organization id", search: deleted.ID.String()},
			{name: "id without hyphens", search: strings.ReplaceAll(id, "-", "")},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				orgs, total, err := ListAllOrganizations(database.Conn(), tc.search, 50, 0, "", "")
				require.NoError(t, err)
				assert.Equal(t, int64(len(tc.want)), total)
				require.Len(t, orgs, len(tc.want))
				for i, org := range orgs {
					assert.Equal(t, tc.want[i], org.ID)
				}
			})
		}
	})

	t.Run("paginates results", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		_, err := CreateOrganization("Aaa", "")
		require.NoError(t, err)
		_, err = CreateOrganization("Bbb", "")
		require.NoError(t, err)
		_, err = CreateOrganization("Ccc", "")
		require.NoError(t, err)

		orgs, total, err := ListAllOrganizations(database.Conn(), "", 2, 0, "", "")
		require.NoError(t, err)
		assert.Len(t, orgs, 2)
		assert.Equal(t, int64(3), total)

		orgs2, _, err := ListAllOrganizations(database.Conn(), "", 2, 2, "", "")
		require.NoError(t, err)
		assert.Len(t, orgs2, 1)
	})
}

func TestFindOrganizationWithCounts(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("returns organization with canvas and member counts", func(t *testing.T) {
		org, err := CreateOrganization("Counted Org", "counted")
		require.NoError(t, err)
		createTestCanvas(t, org.ID, "Canvas 1")
		createTestCanvas(t, org.ID, "Canvas 2")
		createTestUser(t, org.ID, "member-1@example.com", "Member 1")

		found, err := FindOrganizationWithCounts(database.Conn(), org.ID)
		require.NoError(t, err)
		assert.Equal(t, org.ID, found.ID)
		assert.Equal(t, org.Name, found.Name)
		assert.Equal(t, org.Slug, found.Slug)
		assert.Equal(t, "counted", found.Description)
		assert.Equal(t, int64(2), found.CanvasCount)
		assert.Equal(t, int64(0), found.TaskCount)
		assert.Equal(t, int64(0), found.DoneTaskCount)
		assert.Equal(t, int64(1), found.MemberCount)
	})

	t.Run("excludes API keys from member count", func(t *testing.T) {
		org, err := CreateOrganization("Human Members", "")
		require.NoError(t, err)
		createTestUser(t, org.ID, "human-member@example.com", "Human Member")
		createTestAPIKeyUser(t, org.ID, "api-key@example.com", "API Key")

		found, err := FindOrganizationWithCounts(database.Conn(), org.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(1), found.MemberCount)
	})

	t.Run("excludes tasks from a deleted factory", func(t *testing.T) {
		org, err := CreateOrganization("Deleted Factory Org", "")
		require.NoError(t, err)
		db := database.DB(t.Context())
		factory, err := CreateFactory(db, org.ID, "Factory", "", "")
		require.NoError(t, err)
		_, err = factory.CreateWorkOrder(db, "Task 1", "", nil, nil, nil)
		require.NoError(t, err)
		require.NoError(t, factory.SoftDelete(db))

		found, err := FindOrganizationWithCounts(db, org.ID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), found.TaskCount)
		assert.Equal(t, int64(0), found.DoneTaskCount)
	})

	t.Run("returns not found for deleted organization", func(t *testing.T) {
		org, err := CreateOrganization("Soon Deleted", "")
		require.NoError(t, err)
		require.NoError(t, SoftDeleteOrganization(org.ID.String()))

		_, err = FindOrganizationWithCounts(database.Conn(), org.ID)
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	})
}

func createTestTasks(t *testing.T, tx *gorm.DB, organizationID uuid.UUID, count int) {
	t.Helper()

	factory, err := CreateFactory(tx, organizationID, "Factory", "", "")
	require.NoError(t, err)
	for i := 0; i < count; i++ {
		_, err := factory.CreateWorkOrder(tx, fmt.Sprintf("Task %d", i+1), "", nil, nil, nil)
		require.NoError(t, err)
	}
}

func createTestDoneTasks(t *testing.T, tx *gorm.DB, organizationID uuid.UUID, completed, failed, rejected int) {
	t.Helper()

	factory, err := CreateFactory(tx, organizationID, "Factory", "", "")
	require.NoError(t, err)

	createClosed := func(title, result string) {
		t.Helper()
		order, err := factory.CreateWorkOrder(tx, title, "", nil, nil, nil)
		require.NoError(t, err)
		require.NoError(t, tx.Model(order).Updates(map[string]any{
			"state":  FactoryWorkOrderStateClosed,
			"result": result,
		}).Error)
	}

	for i := 0; i < completed; i++ {
		createClosed(fmt.Sprintf("Completed %d", i+1), FactoryWorkOrderResultCompleted)
	}
	for i := 0; i < failed; i++ {
		createClosed(fmt.Sprintf("Failed %d", i+1), FactoryWorkOrderResultFailed)
	}
	for i := 0; i < rejected; i++ {
		createClosed(fmt.Sprintf("Rejected %d", i+1), FactoryWorkOrderResultRejected)
	}
}

func createTestCanvas(t *testing.T, organizationID uuid.UUID, name string) {
	t.Helper()

	now := time.Now()
	liveVersionID := uuid.New()
	canvas := &Canvas{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		LiveVersionID:  &liveVersionID,
		Name:           name,
		CreatedAt:      &now,
		UpdatedAt:      &now,
	}

	require.NoError(t, database.Conn().Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(canvas).Error; err != nil {
			return err
		}

		return tx.Create(&CanvasVersion{
			ID:         liveVersionID,
			WorkflowID: canvas.ID,
			Nodes:      datatypes.NewJSONSlice([]Node{}),
			Edges:      datatypes.NewJSONSlice([]Edge{}),
			CreatedAt:  &now,
			UpdatedAt:  &now,
		}).Error
	}))
}

func createTestUser(t *testing.T, organizationID uuid.UUID, email, name string) {
	t.Helper()

	account, err := CreateAccount(name, email)
	require.NoError(t, err)

	_, err = CreateUser(organizationID, account.ID, account.Email, account.Name)
	require.NoError(t, err)
}

func createTestAPIKeyUser(t *testing.T, organizationID uuid.UUID, email, name string) {
	t.Helper()

	apiKey := &User{
		OrganizationID: organizationID,
		Email:          &email,
		Name:           name,
		Type:           UserTypeAPIKey,
	}
	require.NoError(t, database.Conn().Create(apiKey).Error)
}

func TestListActiveUsersByOrganization(t *testing.T) {
	require.NoError(t, database.TruncateTables())

	t.Run("returns human users for organization", func(t *testing.T) {
		org, err := CreateOrganization("Test Org", "")
		require.NoError(t, err)

		account, err := CreateAccount("Test User", "user@example.com")
		require.NoError(t, err)

		_, err = CreateUser(org.ID, account.ID, account.Email, account.Name)
		require.NoError(t, err)

		users, total, err := ListActiveUsersByOrganization(org.ID.String(), "", 50, 0)
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Test User", users[0].Name)
	})

	t.Run("excludes API keys", func(t *testing.T) {
		require.NoError(t, database.TruncateTables())

		org, err := CreateOrganization("SA Test Org", "")
		require.NoError(t, err)

		account, err := CreateAccount("Human", "human@example.com")
		require.NoError(t, err)

		_, err = CreateUser(org.ID, account.ID, account.Email, account.Name)
		require.NoError(t, err)

		saEmail := "sa@example.com"
		sa := &User{
			OrganizationID: org.ID,
			Email:          &saEmail,
			Name:           "Bot",
			Type:           UserTypeAPIKey,
		}
		err = database.Conn().Create(sa).Error
		require.NoError(t, err)

		users, total, err := ListActiveUsersByOrganization(org.ID.String(), "", 50, 0)
		require.NoError(t, err)
		require.Len(t, users, 1)
		assert.Equal(t, int64(1), total)
		assert.Equal(t, "Human", users[0].Name)
	})
}
