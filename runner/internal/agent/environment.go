package agent

import (
	"errors"
	"os"
	"strings"

	"github.com/superplane/runner/shared/api"
)

func environmentPairs(environment []api.EnvironmentVariable) ([]string, error) {
	if msg := api.ValidateEnvironment(environment); msg != "" {
		return nil, errors.New(msg)
	}
	if len(environment) == 0 {
		return nil, nil
	}
	pairs := make([]string, 0, len(environment))
	for _, variable := range environment {
		pairs = append(pairs, variable.Name+"="+variable.Value)
	}
	return pairs, nil
}

func processEnvironment(environment []api.EnvironmentVariable) ([]string, error) {
	pairs, err := environmentPairs(environment)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, nil
	}
	overrides := make(map[string]struct{}, len(environment))
	for _, variable := range environment {
		overrides[variable.Name] = struct{}{}
	}
	out := make([]string, 0, len(os.Environ())+len(pairs))
	for _, pair := range os.Environ() {
		name, _, _ := strings.Cut(pair, "=")
		if _, ok := overrides[name]; ok {
			continue
		}
		out = append(out, pair)
	}
	return append(out, pairs...), nil
}

func dockerExecEnvironmentArgs(environment []api.EnvironmentVariable) ([]string, error) {
	pairs, err := environmentPairs(environment)
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return nil, nil
	}
	args := make([]string, 0, len(pairs)*2)
	for _, pair := range pairs {
		args = append(args, "--env", pair)
	}
	return args, nil
}
