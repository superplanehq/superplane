package contexts

import (
	"fmt"
	"slices"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

type IntegrationPropertyStorage struct {
	integration *models.Integration
}

func NewIntegrationPropertyStorage(integration *models.Integration) *IntegrationPropertyStorage {
	return &IntegrationPropertyStorage{integration: integration}
}

func (s *IntegrationPropertyStorage) Get(name string) (any, error) {
	for _, property := range s.integration.Properties {
		if property.Name == name {
			return property.Value, nil
		}
	}

	return nil, fmt.Errorf("property %s not found", name)
}

func (s *IntegrationPropertyStorage) GetString(name string) (string, error) {
	value, err := s.Get(name)
	if err != nil {
		return "", err
	}

	v, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("property %s is not a string", name)
	}

	return v, nil
}

func (s *IntegrationPropertyStorage) Delete(names ...string) error {
	s.integration.Properties = slices.DeleteFunc(s.integration.Properties, func(property core.IntegrationPropertyDefinition) bool {
		return slices.Contains(names, property.Name)
	})

	return nil
}

func (s *IntegrationPropertyStorage) Create(def core.IntegrationPropertyDefinition) error {
	_, err := s.Get(def.Name)
	if err == nil {
		return fmt.Errorf("property %s already exists", def.Name)
	}

	s.integration.Properties = append(s.integration.Properties, def)
	return nil
}

func (s *IntegrationPropertyStorage) CreateMany(defs []core.IntegrationPropertyDefinition) error {
	for _, def := range defs {
		err := s.Create(def)
		if err != nil {
			return err
		}
	}

	return nil
}
