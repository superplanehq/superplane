package models_test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func TestGenerateFactoryKeyFromName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"letters only", "SuperPlane", "SUPER"},
		{"strips numbers and punctuation", "release-2025!", "RELEA"},
		{"single word short", "Ops", "OPS"},
		{"empty when no letters", "12345", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, models.GenerateFactoryKeyFromName(c.in))
		})
	}
}

func TestValidateFactoryKey(t *testing.T) {
	cases := []struct {
		name string
		in   string
		err  error
	}{
		{"empty is required", "", models.ErrFactoryKeyRequired},
		{"too short", "A", models.ErrFactoryKeyInvalid},
		{"too long", "ABCDEF", models.ErrFactoryKeyInvalid},
		{"lowercase rejected", "ab", models.ErrFactoryKeyInvalid},
		{"numeric rejected", "AB1", models.ErrFactoryKeyInvalid},
		{"two letters ok", "SP", nil},
		{"five letters ok", "SUPER", nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := models.ValidateFactoryKey(c.in)
			if c.err == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestGenerateUniqueFactoryKey_SuffixesOnCollision(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	first, err := models.CreateFactory(db, r.Organization.ID, "Alpha", "", "AL")
	require.NoError(t, err)
	assert.Equal(t, "AL", first.Key)

	// Requesting the same seed picks a distinct candidate rather than
	// tripping the unique constraint.
	candidate, err := models.GenerateUniqueFactoryKey(db, r.Organization.ID, "Alpha")
	require.NoError(t, err)
	assert.NotEqual(t, "AL", candidate)
	require.NoError(t, models.ValidateFactoryKey(candidate))
}

func TestCreateFactory_RejectsDuplicateKey(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	_, err := models.CreateFactory(db, r.Organization.ID, "Alpha", "", "AL")
	require.NoError(t, err)

	_, err = models.CreateFactory(db, r.Organization.ID, "Alpha copy", "", "AL")
	assert.ErrorIs(t, err, models.ErrFactoryKeyAlreadyExists)
}

func TestCreateWorkOrder_AllocatesSequentialNumbers(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, "Numbers", "", "NUM")
	require.NoError(t, err)

	first, err := factory.CreateWorkOrder(db, "one", "", &r.User, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), first.Number)

	second, err := factory.CreateWorkOrder(db, "two", "", &r.User, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), second.Number)

	assert.Equal(t, "NUM-1", factory.WorkOrderKey(first.Number))
	assert.Equal(t, "NUM-2", factory.WorkOrderKey(second.Number))
}

func TestCreateWorkOrder_ConcurrentAllocationsAreUnique(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, "Concurrent", "", "CON")
	require.NoError(t, err)

	const workers = 10
	var wg sync.WaitGroup
	results := make([]int64, workers)
	errs := make([]error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			f, findErr := models.FindFactory(db, r.Organization.ID, factory.ID)
			if findErr != nil {
				errs[idx] = findErr
				return
			}
			order, createErr := f.CreateWorkOrder(db, "concurrent", "", &r.User, nil, nil)
			if createErr != nil {
				errs[idx] = createErr
				return
			}
			results[idx] = order.Number
		}(i)
	}
	wg.Wait()

	for _, err := range errs {
		require.NoError(t, err)
	}

	seen := make(map[int64]bool, workers)
	for _, n := range results {
		require.NotZero(t, n)
		assert.False(t, seen[n], "expected each work order to receive a unique number, saw %d twice", n)
		seen[n] = true
	}
}

func TestFactoryUpdate_ChangesKey(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, "Rename", "", "RN")
	require.NoError(t, err)

	newKey := "RNM"
	require.NoError(t, factory.Update(db, nil, nil, &newKey))
	assert.Equal(t, "RNM", factory.Key)

	invalid := "A"
	assert.ErrorIs(t, factory.Update(db, nil, nil, &invalid), models.ErrFactoryKeyInvalid)
}
