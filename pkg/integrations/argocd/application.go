package argocd

import "strings"

type Application struct {
	Metadata ApplicationMetadata `json:"metadata"`
	Spec     ApplicationSpec     `json:"spec"`
	Status   ApplicationStatus   `json:"status"`
}

type ApplicationMetadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	UID       string `json:"uid"`
}

type ApplicationSpec struct {
	Project     string                 `json:"project"`
	Source      *ApplicationSource     `json:"source"`
	Sources     []ApplicationSource    `json:"sources"`
	Destination ApplicationDestination `json:"destination"`
}

type ApplicationSource struct {
	RepoURL        string `json:"repoURL"`
	Path           string `json:"path"`
	Chart          string `json:"chart"`
	TargetRevision string `json:"targetRevision"`
}

type ApplicationDestination struct {
	Name      string `json:"name"`
	Server    string `json:"server"`
	Namespace string `json:"namespace"`
}

type ApplicationStatus struct {
	Sync           ApplicationSyncStatus      `json:"sync"`
	Health         ApplicationHealthStatus    `json:"health"`
	OperationState *ApplicationOperationState `json:"operationState"`
	Conditions     []ApplicationCondition     `json:"conditions"`
}

type ApplicationSyncStatus struct {
	Status   string `json:"status"`
	Revision string `json:"revision"`
}

type ApplicationHealthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ApplicationOperationState struct {
	Phase      string               `json:"phase"`
	Message    string               `json:"message"`
	StartedAt  string               `json:"startedAt"`
	FinishedAt string               `json:"finishedAt"`
	Operation  ApplicationOperation `json:"operation"`
}

type ApplicationOperation struct {
	InitiatedBy ApplicationOperationInitiator `json:"initiatedBy"`
}

type ApplicationOperationInitiator struct {
	Username string `json:"username"`
}

type ApplicationCondition struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type ApplicationOutput struct {
	Application ApplicationReference        `json:"application"`
	Sources     []ApplicationSource         `json:"sources"`
	Destination ApplicationDestination      `json:"destination"`
	Sync        ApplicationSyncStatus       `json:"sync"`
	Health      ApplicationHealthStatus     `json:"health"`
	Operation   *ApplicationOperationOutput `json:"operation,omitempty"`
	Conditions  []ApplicationCondition      `json:"conditions"`
}

type ApplicationReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	UID       string `json:"uid"`
	Project   string `json:"project"`
}

type ApplicationOperationOutput struct {
	Phase       string `json:"phase"`
	Message     string `json:"message"`
	StartedAt   string `json:"startedAt"`
	FinishedAt  string `json:"finishedAt"`
	InitiatedBy string `json:"initiatedBy"`
}

func applicationOutputFrom(application Application, project string) ApplicationOutput {
	project = firstNonEmpty(application.Spec.Project, project)

	output := ApplicationOutput{
		Application: ApplicationReference{
			Name:      application.Metadata.Name,
			Namespace: application.Metadata.Namespace,
			UID:       application.Metadata.UID,
			Project:   project,
		},
		Sources:     applicationSources(application.Spec),
		Destination: application.Spec.Destination,
		Sync:        application.Status.Sync,
		Health:      application.Status.Health,
		Conditions:  application.Status.Conditions,
	}

	if application.Status.OperationState != nil {
		operation := application.Status.OperationState
		output.Operation = &ApplicationOperationOutput{
			Phase:       operation.Phase,
			Message:     operation.Message,
			StartedAt:   operation.StartedAt,
			FinishedAt:  operation.FinishedAt,
			InitiatedBy: operation.Operation.InitiatedBy.Username,
		}
	}

	return output
}

func applicationSources(spec ApplicationSpec) []ApplicationSource {
	if len(spec.Sources) > 0 {
		return spec.Sources
	}

	if spec.Source == nil {
		return []ApplicationSource{}
	}

	return []ApplicationSource{*spec.Source}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}
