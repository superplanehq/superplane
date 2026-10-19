package encryption

import (
	"bytes"
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
	primary string
	mu      sync.RWMutex
	cache   map[uuid.UUID]crypto.DataKey
}

var ErrMigrationUnfinished = errors.New("finish the encryption migration before creating a data key")

func NewStore(masters crypto.MasterKeys, primary string) *Store {
	return &Store{masters: masters, primary: primary, cache: make(map[uuid.UUID]crypto.DataKey)}
}

func (s *Store) PrimaryMasterKeyRef() string {
	return s.primary
}

func (s *Store) Initialize(ctx context.Context) error {
	if !s.masters.Has(s.primary) {
		return fmt.Errorf("primary master key %q is not configured", s.primary)
	}
	if _, err := models.LoadActiveApplicationEncryptionKey(database.DB(ctx)); err == nil {
		return s.verifyKeys(ctx)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("load active data key: %w", err)
	}
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		keys, err := models.ListApplicationEncryptionKeys(tx)
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			return errors.New("data keys exist without an active key")
		}
		key, err := s.createKey(ctx, s.primary)
		if err != nil {
			return err
		}
		return key.CreateInitial(tx)
	})
	if err != nil {
		if _, loadErr := models.LoadActiveApplicationEncryptionKey(database.DB(ctx)); loadErr != nil {
			return fmt.Errorf("initialize data keys: %w", err)
		}
	}
	return s.verifyKeys(ctx)
}

func (s *Store) verifyKeys(ctx context.Context) error {
	if _, err := models.LoadActiveApplicationEncryptionKey(database.DB(ctx)); err != nil {
		return fmt.Errorf("load active data key: %w", err)
	}
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
	return s.active(ctx, database.DB(ctx))
}

func (s *Store) active(ctx context.Context, db *gorm.DB) (crypto.DataKey, error) {
	active, err := models.LoadActiveApplicationEncryptionKey(db)
	if err != nil {
		return crypto.DataKey{}, err
	}
	return s.find(ctx, db, active.ID)
}

func (s *Store) Find(ctx context.Context, id uuid.UUID) (crypto.DataKey, error) {
	return s.find(ctx, database.DB(ctx), id)
}

func (s *Store) find(ctx context.Context, db *gorm.DB, id uuid.UUID) (crypto.DataKey, error) {
	s.mu.RLock()
	key, found := s.cache[id]
	s.mu.RUnlock()
	if found {
		return key, nil
	}
	row, err := models.LoadApplicationEncryptionKey(db, id)
	if err != nil {
		return crypto.DataKey{}, err
	}
	return s.unwrap(ctx, row)
}

type transactionDataKeys struct {
	store *Store
	tx    *gorm.DB
}

func (k transactionDataKeys) Active(ctx context.Context) (crypto.DataKey, error) {
	return k.store.active(ctx, k.tx)
}

func (k transactionDataKeys) Find(ctx context.Context, id uuid.UUID) (crypto.DataKey, error) {
	return k.store.find(ctx, k.tx, id)
}

func (s *Store) List(ctx context.Context) ([]models.ApplicationEncryptionKey, uuid.UUID, error) {
	keys, err := models.ListApplicationEncryptionKeys(database.DB(ctx))
	if err != nil {
		return nil, uuid.Nil, err
	}
	for _, key := range keys {
		if key.State == models.ApplicationEncryptionKeyActive {
			return keys, key.ID, nil
		}
	}
	return nil, uuid.Nil, errors.New("no active data key")
}

func (s *Store) CreateAndActivateDataKey(ctx context.Context) (uuid.UUID, error) {
	var newID uuid.UUID
	err := database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		active, err := models.LockActiveApplicationEncryptionKey(tx)
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
		key, err := s.createKey(ctx, s.primary)
		if err != nil {
			return err
		}
		if err := key.CreateAndActivate(tx, active); err != nil {
			return err
		}
		newID = key.ID
		return nil
	})
	return newID, err
}

func (s *Store) Rewrap(ctx context.Context) error {
	return database.DB(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := models.LoadActiveApplicationEncryptionKey(tx); err != nil {
			return fmt.Errorf("load active data key: %w", err)
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
			if len(plain) != 32 {
				return fmt.Errorf("unwrap data key %s: stored data key must contain 32 bytes", keys[i].ID)
			}
			wrapped, err := s.masters.Wrap(ctx, s.primary, plain, keyAssociatedData(keys[i].ID))
			if err != nil {
				return fmt.Errorf("wrap data key %s: %w", keys[i].ID, err)
			}
			verified, err := s.masters.Unwrap(ctx, s.primary, wrapped, keyAssociatedData(keys[i].ID))
			if err != nil {
				return fmt.Errorf("verify wrapped data key %s: %w", keys[i].ID, err)
			}
			if !bytes.Equal(plain, verified) {
				return fmt.Errorf("verify wrapped data key %s: decrypted key changed", keys[i].ID)
			}
			if err := keys[i].Rewrap(tx, wrapped, s.primary); err != nil {
				return err
			}
		}
		return nil
	})
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
	verified, err := s.masters.Unwrap(ctx, reference, wrapped, keyAssociatedData(id))
	if err != nil {
		return nil, fmt.Errorf("verify wrapped data key: %w", err)
	}
	if !bytes.Equal(material, verified) {
		return nil, errors.New("verify wrapped data key: decrypted key changed")
	}
	return &models.ApplicationEncryptionKey{ID: id, WrappedKey: wrapped, MasterKeyRef: reference, State: models.ApplicationEncryptionKeyAvailable}, nil
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
