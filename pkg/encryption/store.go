package encryption

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type Store struct {
	masters crypto.MasterKeys
	mu      sync.RWMutex
	cache   map[uuid.UUID]crypto.DataKey
}

var ErrMigrationUnfinished = errors.New("finish the encryption migration before rotating data keys")
var ErrMasterKeyNotConfigured = errors.New("master key is not configured")

func NewStore(masters crypto.MasterKeys) *Store {
	return &Store{masters: masters, cache: make(map[uuid.UUID]crypto.DataKey)}
}

func (s *Store) Initialize(ctx context.Context, initialMasterKeyRef string) error {
	if _, err := models.LoadApplicationEncryptionState(database.DB(ctx)); err == nil {
		return s.verifyKeys(ctx)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("load encryption state: %w", err)
	}
	if !s.masters.Has(initialMasterKeyRef) {
		return fmt.Errorf("initial master key %q is not configured", initialMasterKeyRef)
	}
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		key, err := s.createKey(ctx, initialMasterKeyRef)
		if err != nil {
			return err
		}
		if err := models.CreateApplicationEncryptionKey(tx, key); err != nil {
			return err
		}
		return models.CreateApplicationEncryptionState(tx, &models.ApplicationEncryptionState{
			ID: 1, ActiveKeyID: key.ID, Generation: 1,
		})
	})
	if err != nil {
		if _, loadErr := models.LoadApplicationEncryptionState(database.DB(ctx)); loadErr != nil {
			return fmt.Errorf("initialize encryption state: %w", err)
		}
	}
	return s.verifyKeys(ctx)
}

func (s *Store) verifyKeys(ctx context.Context) error {
	keys, err := models.ListApplicationEncryptionKeys(database.DB(ctx))
	if err != nil {
		return fmt.Errorf("list data keys: %w", err)
	}
	for _, key := range keys {
		if _, err := s.unwrap(ctx, &key); err != nil {
			return fmt.Errorf("load data key %s: %w", key.ID, err)
		}
	}
	return nil
}

func (s *Store) Active(ctx context.Context) (crypto.DataKey, error) {
	state, err := models.LoadApplicationEncryptionState(database.DB(ctx))
	if err != nil {
		return crypto.DataKey{}, err
	}
	return s.Find(ctx, state.ActiveKeyID)
}

func (s *Store) Find(ctx context.Context, id uuid.UUID) (crypto.DataKey, error) {
	s.mu.RLock()
	key, found := s.cache[id]
	s.mu.RUnlock()
	if found {
		return key, nil
	}
	row, err := models.LoadApplicationEncryptionKey(database.DB(ctx), id)
	if err != nil {
		return crypto.DataKey{}, err
	}
	return s.unwrap(ctx, row)
}

func (s *Store) List(ctx context.Context) ([]models.ApplicationEncryptionKey, uuid.UUID, error) {
	state, err := models.LoadApplicationEncryptionState(database.DB(ctx))
	if err != nil {
		return nil, uuid.Nil, err
	}
	keys, err := models.ListApplicationEncryptionKeys(database.DB(ctx))
	return keys, state.ActiveKeyID, err
}

func (s *Store) Rotate(ctx context.Context) (uuid.UUID, error) {
	var newID uuid.UUID
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		state, err := models.LockApplicationEncryptionState(tx)
		if err != nil {
			return err
		}
		unfinished, err := models.HasUnfinishedApplicationEncryptionMigration(tx)
		if err != nil {
			return err
		}
		if unfinished {
			return ErrMigrationUnfinished
		}
		active, err := models.LoadApplicationEncryptionKey(tx, state.ActiveKeyID)
		if err != nil {
			return err
		}
		key, err := s.createKey(ctx, active.MasterKeyRef)
		if err != nil {
			return err
		}
		if err := models.CreateApplicationEncryptionKey(tx, key); err != nil {
			return err
		}
		if err := models.ActivateApplicationEncryptionKey(tx, state, key.ID); err != nil {
			return err
		}
		newID = key.ID
		return nil
	})
	return newID, err
}

func (s *Store) Rewrap(ctx context.Context, reference string) error {
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		state, err := models.LockApplicationEncryptionState(tx)
		if err != nil {
			return err
		}
		target := reference
		if target == "" {
			active, err := models.LoadApplicationEncryptionKey(tx, state.ActiveKeyID)
			if err != nil {
				return err
			}
			target = active.MasterKeyRef
		}
		if !s.masters.Has(target) {
			return fmt.Errorf("%w: %q", ErrMasterKeyNotConfigured, target)
		}
		keys, err := models.ListApplicationEncryptionKeys(tx)
		if err != nil {
			return err
		}
		for i := range keys {
			plain, err := s.masters.Unwrap(ctx, keys[i].MasterKeyRef, keys[i].WrappedKey, keyAssociatedData(keys[i].ID))
			if err != nil {
				return fmt.Errorf("unwrap data key %s: %w", keys[i].ID, err)
			}
			wrapped, err := s.masters.Wrap(ctx, target, plain, keyAssociatedData(keys[i].ID))
			if err != nil {
				return fmt.Errorf("wrap data key %s: %w", keys[i].ID, err)
			}
			if err := models.RewrapApplicationEncryptionKey(tx, &keys[i], wrapped, target); err != nil {
				return err
			}
		}
		return models.AdvanceApplicationEncryptionGeneration(tx, state)
	})
	if err != nil {
		return err
	}
	return s.verifyKeys(ctx)
}

func (s *Store) createKey(ctx context.Context, reference string) (*models.ApplicationEncryptionKey, error) {
	id := uuid.New()
	material := make([]byte, 32)
	if _, err := rand.Read(material); err != nil {
		return nil, fmt.Errorf("generate data key: %w", err)
	}
	wrapped, err := s.masters.Wrap(ctx, reference, material, keyAssociatedData(id))
	if err != nil {
		return nil, fmt.Errorf("wrap data key: %w", err)
	}
	return &models.ApplicationEncryptionKey{ID: id, WrappedKey: wrapped, MasterKeyRef: reference}, nil
}

func (s *Store) unwrap(ctx context.Context, row *models.ApplicationEncryptionKey) (crypto.DataKey, error) {
	material, err := s.masters.Unwrap(ctx, row.MasterKeyRef, row.WrappedKey, keyAssociatedData(row.ID))
	if err != nil {
		return crypto.DataKey{}, err
	}
	if len(material) != 32 {
		return crypto.DataKey{}, errors.New("stored data key must contain 32 bytes")
	}
	key := crypto.DataKey{ID: row.ID, Material: material}
	s.mu.Lock()
	s.cache[row.ID] = key
	s.mu.Unlock()
	return key, nil
}

func keyAssociatedData(id uuid.UUID) []byte {
	return []byte("superplane:data-key:" + id.String())
}
