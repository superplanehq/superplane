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

func TestNormalizeFactoryKey(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"uppercase becomes lowercase", "SP", "sp"},
		{"mixed case becomes lowercase", "SuPeR", "super"},
		{"trims surrounding whitespace", "  sp  ", "sp"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, models.NormalizeFactoryKey(c.in))
		})
	}
}

func TestValidateFactoryKey_ErrorText(t *testing.T) {
	err := models.ValidateFactoryKey("ABC")
	require.Error(t, err)
	assert.Equal(t, "factory key must be 2 to 5 lowercase letters", err.Error())
}

func TestGenerateFactoryKeyFromName(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"letters only", "SuperPlane", "super"},
		{"strips numbers and punctuation", "release-2025!", "relea"},
		{"single word short", "Ops", "ops"},
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
		{"too short", "a", models.ErrFactoryKeyInvalid},
		{"too long", "abcdef", models.ErrFactoryKeyInvalid},
		{"uppercase rejected", "AB", models.ErrFactoryKeyInvalid},
		{"numeric rejected", "ab1", models.ErrFactoryKeyInvalid},
		{"two letters ok", "sp", nil},
		{"five letters ok", "super", nil},
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

	first, err := models.CreateFactory(db, r.Organization.ID, "Alpha", "", "al")
	require.NoError(t, err)
	assert.Equal(t, "al", first.Key)

	// Requesting the same seed picks a distinct candidate rather than
	// tripping the unique constraint.
	candidate, err := models.GenerateUniqueFactoryKey(db, r.Organization.ID, "Alpha")
	require.NoError(t, err)
	assert.NotEqual(t, "al", candidate)
	require.NoError(t, models.ValidateFactoryKey(candidate))
}

func TestGenerateUniqueFactoryKey_WalksLowercaseAlphabetOnCollision(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	// "alpha" is exactly what GenerateFactoryKeyFromName derives from "Alpha"
	// (5 letters, the max length), so requesting it again collides.
	_, err := models.CreateFactory(db, r.Organization.ID, "Alpha", "", "alpha")
	require.NoError(t, err)

	// The walk keeps the first four letters and cycles the last one through
	// the lowercase alphabet: "alphb" is the first free candidate (it skips
	// "alpha" itself).
	candidate, err := models.GenerateUniqueFactoryKey(db, r.Organization.ID, "Alpha")
	require.NoError(t, err)
	assert.Equal(t, "alphb", candidate)
}

func TestCreateFactory_NormalizesUppercaseKeyToLowercase(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, support.RandomName("factory"), "", "SUPER")
	require.NoError(t, err)
	assert.Equal(t, "super", factory.Key)
}

func TestCreateFactory_RejectsDuplicateKey(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	_, err := models.CreateFactory(db, r.Organization.ID, "Alpha", "", "al")
	require.NoError(t, err)

	_, err = models.CreateFactory(db, r.Organization.ID, "Alpha copy", "", "al")
	assert.ErrorIs(t, err, models.ErrFactoryKeyAlreadyExists)
}

func TestCreateWorkOrder_AllocatesSequentialNumbers(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, "Numbers", "", "num")
	require.NoError(t, err)

	first, err := factory.CreateWorkOrder(db, "one", "", &r.User, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), first.Number)

	second, err := factory.CreateWorkOrder(db, "two", "", &r.User, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(2), second.Number)

	assert.Equal(t, "num-1", factory.WorkOrderKey(first.Number))
	assert.Equal(t, "num-2", factory.WorkOrderKey(second.Number))
}

func TestCreateWorkOrder_ConcurrentAllocationsAreUnique(t *testing.T) {
	r := support.Setup(t)
	db := database.DB(t.Context())

	factory, err := models.CreateFactory(db, r.Organization.ID, "Concurrent", "", "con")
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

	factory, err := models.CreateFactory(db, r.Organization.ID, "Rename", "", "rn")
	require.NoError(t, err)

	newKey := "rnm"
	require.NoError(t, factory.Update(db, nil, nil, &newKey))
	assert.Equal(t, "rnm", factory.Key)

	invalid := "A"
	assert.ErrorIs(t, factory.Update(db, nil, nil, &invalid), models.ErrFactoryKeyInvalid)
}
