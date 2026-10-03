package models

type ExecutionMode string

const (
	ExecutionHost   ExecutionMode = "host"
	ExecutionDocker ExecutionMode = "docker"
)

type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
