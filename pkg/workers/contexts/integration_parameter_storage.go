package contexts

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

type IntegrationParameterStorage struct {
	integration *models.Integration
}

func NewIntegrationParameterStorage(integration *models.Integration) *IntegrationParameterStorage {
	return &IntegrationParameterStorage{integration: integration}
}

func (s *IntegrationParameterStorage) Get(name string) (any, error) {
	for _, param := range s.integration.Parameters {
		if param.Name == name {
			return param.Value, nil
		}
	}

	return nil, fmt.Errorf("parameter %s not found", name)
}

func (s *IntegrationParameterStorage) Delete(name string) error {
	for i, param := range s.integration.Parameters {
		if param.Name == name {
			s.integration.Parameters = append(s.integration.Parameters[:i], s.integration.Parameters[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("parameter %s not found", name)
}

func (s *IntegrationParameterStorage) Create(def core.IntegrationParameterDefinition) error {
	_, err := s.Get(def.Name)
	if err == nil {
		return fmt.Errorf("parameter %s already exists", def.Name)
	}

	s.integration.Parameters = append(s.integration.Parameters, def)
	return nil
}
