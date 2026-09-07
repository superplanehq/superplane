package models

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strings"
)

const (
	FactoryRepositoryAnalysisStatusRunning = "running"
	FactoryRepositoryAnalysisStatusReady   = "ready"
	FactoryRepositoryAnalysisStatusFailed  = "failed"
)

var repositorySetupCommandPrefixes = []string{
	"go ",
	"npm ",
	"pnpm ",
	"yarn ",
	"bun ",
	"pip ",
	"pip3 ",
	"poetry ",
	"uv ",
	"make ",
}

const (
	maxRepositoryContextProjects     = 20
	maxRepositoryContextCommands     = 12
	maxRepositoryContextRequirements = 20
)

type sussDocument struct {
	SchemaVersion string        `json:"schemaVersion"`
	Projects      []sussProject `json:"projects"`
}

type sussProject struct {
	Path            string            `json:"path"`
	Languages       []sussNamedValue  `json:"languages"`
	Frameworks      []sussNamedValue  `json:"frameworks"`
	PackageManagers []sussNamedValue  `json:"packageManagers"`
	Facts           []sussFact        `json:"facts"`
	Requirements    []sussRequirement `json:"requirements"`
	Preparation     []sussCommand     `json:"preparation"`
	Commands        []sussCommand     `json:"commands"`
	Ambiguities     []json.RawMessage `json:"ambiguities"`
	Conflicts       []json.RawMessage `json:"conflicts"`
}

type sussNamedValue struct {
	Name string `json:"name"`
}

type sussRequirement struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type sussFact struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type sussInterpretation struct {
	Capability string `json:"capability"`
}

type sussCommand struct {
	Name            string               `json:"name"`
	Run             *string              `json:"run"`
	Directory       string               `json:"directory"`
	Interpretations []sussInterpretation `json:"interpretations"`
}

func BuildFactoryRepositoryAnalysis(document, commitSHA string) (FactoryRepositoryAnalysis, error) {
	raw := json.RawMessage(strings.TrimSpace(document))
	if !json.Valid(raw) || len(raw) > MaxFactoryRepositoryAnalysisBytes {
		return FactoryRepositoryAnalysis{}, ErrFactoryRepositoryAnalysisInvalid
	}

	var parsed sussDocument
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return FactoryRepositoryAnalysis{}, fmt.Errorf("%w: %v", ErrFactoryRepositoryAnalysisInvalid, err)
	}
	if parsed.SchemaVersion != "1" {
		return FactoryRepositoryAnalysis{}, fmt.Errorf("%w: unsupported Suss schema version", ErrFactoryRepositoryAnalysisInvalid)
	}

	languages := repositoryLanguages(parsed.Projects)
	setupSteps := repositorySetupSteps(parsed.Projects)
	context, err := repositoryContext(parsed.Projects)
	if err != nil {
		return FactoryRepositoryAnalysis{}, err
	}

	return FactoryRepositoryAnalysis{
		Status:     FactoryRepositoryAnalysisStatusReady,
		CommitSHA:  strings.TrimSpace(commitSHA),
		Languages:  languages,
		SetupSteps: setupSteps,
		Context:    context,
		Document:   raw,
	}, nil
}

func repositoryLanguages(projects []sussProject) []string {
	languages := []string{}
	for _, project := range projects {
		if isFixtureProject(project) {
			continue
		}
		for _, language := range project.Languages {
			name := strings.TrimSpace(language.Name)
			if name != "" && !slices.Contains(languages, name) {
				languages = append(languages, name)
			}
		}
	}
	slices.Sort(languages)
	return languages
}

func repositorySetupSteps(projects []sussProject) []FactoryRepositorySetupStep {
	steps := []FactoryRepositorySetupStep{}
	for _, project := range projects {
		if isFixtureProject(project) {
			continue
		}
		for _, command := range project.Preparation {
			if command.Run == nil {
				continue
			}
			run := strings.TrimSpace(*command.Run)
			directory, ok := safeRepositoryDirectory(project.Path, command.Directory)
			if !allowedRepositorySetupCommand(run) || !ok {
				continue
			}
			if slices.ContainsFunc(steps, func(step FactoryRepositorySetupStep) bool {
				return step.Command == run && step.Directory == directory
			}) {
				continue
			}
			steps = append(steps, FactoryRepositorySetupStep{
				Name:      strings.TrimSpace(command.Name),
				Command:   run,
				Directory: directory,
			})
		}
	}
	return steps
}

func isFixtureProject(project sussProject) bool {
	return slices.ContainsFunc(project.Facts, func(fact sussFact) bool {
		return fact.Name == "project.role" && fact.Value == "fixture"
	})
}

func allowedRepositorySetupCommand(command string) bool {
	if strings.ContainsAny(command, "\r\n\x00") {
		return false
	}
	return slices.ContainsFunc(repositorySetupCommandPrefixes, func(prefix string) bool {
		return strings.HasPrefix(command, prefix)
	})
}

func safeRepositoryDirectory(projectPath, commandDirectory string) (string, bool) {
	directory := strings.TrimSpace(commandDirectory)
	if directory == "" || directory == "." {
		directory = strings.TrimSpace(projectPath)
	}
	if directory == "" {
		directory = "."
	}
	cleaned := path.Clean(directory)
	if path.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func repositoryContext(projects []sussProject) (string, error) {
	type contextProject struct {
		Path            string            `json:"path"`
		Languages       []sussNamedValue  `json:"languages,omitempty"`
		Frameworks      []sussNamedValue  `json:"frameworks,omitempty"`
		PackageManagers []sussNamedValue  `json:"package_managers,omitempty"`
		Requirements    []sussRequirement `json:"requirements,omitempty"`
		Commands        []sussCommand     `json:"commands,omitempty"`
	}

	contextProjects := make([]contextProject, 0, min(len(projects), maxRepositoryContextProjects))
	for _, project := range projects[:min(len(projects), maxRepositoryContextProjects)] {
		if isFixtureProject(project) {
			continue
		}
		commands := slices.DeleteFunc(slices.Clone(project.Commands), func(command sussCommand) bool {
			return command.Run == nil || !slices.ContainsFunc(command.Interpretations, func(value sussInterpretation) bool {
				switch value.Capability {
				case "artifact.build", "test.run", "code.lint", "code.format", "code.typecheck":
					return true
				default:
					return false
				}
			})
		})
		commands = commands[:min(len(commands), maxRepositoryContextCommands)]
		requirements := project.Requirements[:min(len(project.Requirements), maxRepositoryContextRequirements)]
		contextProjects = append(contextProjects, contextProject{
			Path:            project.Path,
			Languages:       project.Languages,
			Frameworks:      project.Frameworks,
			PackageManagers: project.PackageManagers,
			Requirements:    requirements,
			Commands:        commands,
		})
	}

	raw, err := json.Marshal(map[string]any{"projects": contextProjects})
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrFactoryRepositoryAnalysisInvalid, err)
	}
	return string(raw), nil
}
