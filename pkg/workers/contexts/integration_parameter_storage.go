package contexts

import (
	"fmt"
	"slices"

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

func (s *IntegrationParameterStorage) GetString(name string) (string, error) {
	value, err := s.Get(name)
	if err != nil {
		return "", err
	}

	v, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s is not a string", name)
	}

	return v, nil
}

func (s *IntegrationParameterStorage) Delete(names ...string) error {
	newParameters := slices.Clone(s.integration.Parameters)

	for i, param := range s.integration.Parameters {
		if slices.Contains(names, param.Name) {
			newParameters = append(newParameters[:i], newParameters[i+1:]...)
		}
	}

	s.integration.Parameters = newParameters
	return nil
}

func (s *IntegrationParameterStorage) Create(def core.IntegrationParameterDefinition) error {
	_, err := s.Get(def.Name)
	if err == nil {
		return fmt.Errorf("parameter %s already exists", def.Name)
	}

	s.integration.Parameters = append(s.integration.Parameters, def)
	return nil
}
