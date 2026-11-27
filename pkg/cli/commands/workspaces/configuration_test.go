package workspaces

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	clitest "github.com/superplanehq/superplane/test/support/cli"
)

const configurationManifest = `apiVersion: v2
kind: Factory
name: Example workspace
stages:
  - id: implementation
    name: Implement
    run: implement
    automations:
      - id: implement
        uses: "@superplane/implement"
        with:
          agent:
            additional_instructions: Follow AGENTS.md.
`

func configurationCLI(t *testing.T, handler http.HandlerFunc) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	ctx, stdout := clitest.NewCommandContext(t, server, "text")
	root := NewCommand(core.BindOptions{
		NewAPIClient: func() *openapi_client.APIClient { return ctx.API },
		NewConfigContext: func() core.ConfigContext {
			return &clitest.FakeConfig{ActiveWorkspace: "active-workspace"}
		},
	})
	root.SetOut(stdout)
	root.SetErr(stdout)
	root.SetIn(strings.NewReader(""))
	root.SilenceErrors = true
	root.SilenceUsage = true
	return root, stdout
}

func writeConfigurationInput(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "main.yml")
	require.NoError(t, os.WriteFile(path, []byte(configurationManifest), 0600))
	return path
}

func exportPayload(content string) map[string]any {
	return map[string]any{
		"coverage": "Workspace name, repositories, stages, and automations",
		"bundle": map[string]any{
			"manifestPath": ".superplane/main.yml",
			"files":        []map[string]any{{"path": ".superplane/main.yml", "content": content}},
		},
	}
}

func TestConfigurationCommandsRequireExplicitWorkspace(t *testing.T) {
	for _, operation := range []string{"export", "plan", "apply"} {
		t.Run(operation, func(t *testing.T) {
			root, _ := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) { t.Errorf("unexpected API call: %s", r.URL) })
			args := []string{operation}
			if operation == "export" {
				args = append(args, "--output-dir", t.TempDir())
			} else {
				args = append(args, "-f", writeConfigurationInput(t))
			}
			root.SetArgs(args)
			require.ErrorContains(t, root.Execute(), "workspace")
		})
	}
}

func TestExportConfigurationDefaultsToCurrentDirectory(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)
	root, _ := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(exportPayload(configurationManifest)))
	})
	root.SetArgs([]string{"export", "--workspace", "example"})
	require.NoError(t, root.Execute())
	actual, err := os.ReadFile(filepath.Join(directory, ".superplane", "main.yml"))
	require.NoError(t, err)
	assert.Equal(t, configurationManifest, string(actual))
}

func TestExportConfigurationProtectsLocalFiles(t *testing.T) {
	outputDir := t.TempDir()
	unrelated := filepath.Join(outputDir, "README.md")
	require.NoError(t, os.WriteFile(unrelated, []byte("Keep this file."), 0600))
	content := configurationManifest
	root, stdout := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/factories/example/configuration", r.URL.Path)
		assert.Empty(t, r.URL.Query().Get("lineId"))
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(exportPayload(content)))
	})
	args := []string{"export", "--workspace", "example", "--output-dir", outputDir}
	root.SetArgs(args)
	require.NoError(t, root.Execute())
	require.Contains(t, stdout.String(), "Workspace name, repositories, stages, and automations")
	manifest := filepath.Join(outputDir, ".superplane", "main.yml")
	actual, err := os.ReadFile(manifest)
	require.NoError(t, err)
	assert.Equal(t, configurationManifest, string(actual))

	content = strings.ReplaceAll(configurationManifest, "Follow AGENTS.md.", "Use the UI changes.")
	root.SetArgs(args)
	require.ErrorContains(t, root.Execute(), "--overwrite")
	actual, err = os.ReadFile(manifest)
	require.NoError(t, err)
	assert.Equal(t, configurationManifest, string(actual))

	root.SetArgs(append(args, "--overwrite"))
	require.NoError(t, root.Execute())
	actual, err = os.ReadFile(manifest)
	require.NoError(t, err)
	assert.Equal(t, content, string(actual))
	actual, err = os.ReadFile(unrelated)
	require.NoError(t, err)
	assert.Equal(t, "Keep this file.", string(actual))
}

func TestExportConfigurationRejectsInvalidResponsesBeforeOverwrite(t *testing.T) {
	for _, scenario := range []string{"missing bundle", "missing coverage", "unsafe manifest", "unsafe file", "extra file", "missing content", "invalid YAML", "missing stages", "unknown field", "unknown binding", "conflicting binding", "missing repository connection", "missing primary repository", "missing repository access", "invalid repository access", "invalid repository enabled", "unknown repository field", "stage kind", "zero concurrency", "unknown runner", "YAML alias"} {
		t.Run(scenario, func(t *testing.T) {
			outputDir := t.TempDir()
			require.NoError(t, os.Mkdir(filepath.Join(outputDir, ".superplane"), 0755))
			manifest := filepath.Join(outputDir, ".superplane", "main.yml")
			require.NoError(t, os.WriteFile(manifest, []byte("existing content"), 0600))
			payload := exportPayload(configurationManifest)
			bundle := payload["bundle"].(map[string]any)
			files := bundle["files"].([]map[string]any)
			switch scenario {
			case "missing bundle":
				delete(payload, "bundle")
			case "missing coverage":
				delete(payload, "coverage")
			case "unsafe manifest":
				bundle["manifestPath"] = "../main.yml"
			case "unsafe file":
				files[0]["path"] = "/tmp/main.yml"
			case "extra file":
				bundle["files"] = append(files, map[string]any{"path": "README.md", "content": "overwrite"})
			case "missing content":
				delete(files[0], "content")
			case "invalid YAML":
				files[0]["content"] = "apiVersion: ["
			case "missing stages":
				files[0]["content"] = "apiVersion: v2\nkind: Factory\nname: Example workspace\n"
			case "unknown field":
				files[0]["content"] = configurationManifest + "unsupported: true\n"
			case "unknown binding":
				files[0]["content"] = strings.Replace(configurationManifest, "        with:", "        binding:\n          credentials: secret\n        with:", 1)
			case "conflicting binding":
				files[0]["content"] = strings.Replace(configurationManifest, "        with:", "        binding:\n          integration_name: GitHub\n          integration_id: github-id\n        with:", 1)
			case "missing repository connection":
				files[0]["content"] = configurationManifest + "repositories:\n  primary:\n    repository: acme/app\n"
			case "missing primary repository":
				files[0]["content"] = configurationManifest + "repositories:\n  binding:\n    integration_name: GitHub\n"
			case "missing repository access", "invalid repository access", "invalid repository enabled", "unknown repository field":
				repositories := "repositories:\n  binding:\n    integration_name: GitHub\n  primary:\n    repository: acme/app\n  additional:\n    - repository: acme/docs\n"
				switch scenario {
				case "invalid repository access":
					repositories += "      access: admin\n"
				case "invalid repository enabled":
					repositories += "      access: read\n      enabled: sometimes\n"
				case "unknown repository field":
					repositories += "      access: read\n      credentials: secret\n"
				}
				files[0]["content"] = configurationManifest + repositories
			case "stage kind":
				files[0]["content"] = strings.Replace(configurationManifest, "name: Implement", "name: Implement\n    kind: phase", 1)
			case "zero concurrency":
				files[0]["content"] = strings.Replace(configurationManifest, "run: implement", "run: implement\n    max_parallelism: 0", 1)
			case "unknown runner":
				files[0]["content"] = strings.Replace(configurationManifest, "    run: implement", "    run: missing", 1)
			case "YAML alias":
				aliased := strings.ReplaceAll(configurationManifest, "name: Example workspace", "name: &workspace Example workspace")
				files[0]["content"] = strings.ReplaceAll(aliased, "name: Implement", "name: *workspace")
			}
			root, _ := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(payload))
			})
			root.SetArgs([]string{"export", "--workspace", "example", "--output-dir", outputDir, "--overwrite"})
			require.Error(t, root.Execute())
			actual, err := os.ReadFile(manifest)
			require.NoError(t, err)
			assert.Equal(t, "existing content", string(actual))
		})
	}
}

func TestExportConfigurationRejectsSymlinks(t *testing.T) {
	for _, location := range []string{"output", "directory", "manifest"} {
		t.Run(location, func(t *testing.T) {
			outputDir := filepath.Join(t.TempDir(), "output")
			targetDir := t.TempDir()
			target := filepath.Join(targetDir, "main.yml")
			require.NoError(t, os.WriteFile(target, []byte("Keep this file."), 0600))
			switch location {
			case "output":
				require.NoError(t, os.Symlink(targetDir, outputDir))
			case "directory":
				require.NoError(t, os.MkdirAll(outputDir, 0755))
				require.NoError(t, os.Symlink(targetDir, filepath.Join(outputDir, ".superplane")))
			case "manifest":
				require.NoError(t, os.MkdirAll(filepath.Join(outputDir, ".superplane"), 0755))
				require.NoError(t, os.Symlink(target, filepath.Join(outputDir, ".superplane", "main.yml")))
			}
			root, _ := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(exportPayload(configurationManifest)))
			})
			root.SetArgs([]string{"export", "--workspace", "example", "--output-dir", outputDir, "--overwrite"})
			require.Error(t, root.Execute())
			actual, err := os.ReadFile(target)
			require.NoError(t, err)
			assert.Equal(t, "Keep this file.", string(actual))
		})
	}
}

func TestApplyConfigurationSubmitsCapturedPlanInput(t *testing.T) {
	path := writeConfigurationInput(t)
	var requests []string
	var planned map[string]any
	root, stdout := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.NotContains(t, body, "lineId")
		switch r.URL.Path {
		case "/api/v1/factories/example/configuration:plan":
			planned = body
			require.NoError(t, os.WriteFile(path, []byte("changed after preview"), 0600))
			_, _ = fmt.Fprint(w, `{"coverage":"Workspace name, repositories, stages, and automations","baseStateToken":"preview-token","changes":[{"resource":"implementation/implement","fieldPath":"with.agent.additional_instructions","previousValue":"old","requestedValue":"Follow AGENTS.md."}]}`)
		case "/api/v1/factories/example/configuration:apply":
			assert.Equal(t, "preview-token", body["baseStateToken"])
			assert.Equal(t, planned["bundle"], body["bundle"])
			bundle := body["bundle"].(map[string]any)
			files := bundle["files"].([]any)
			assert.Equal(t, configurationManifest, files[0].(map[string]any)["content"])
			_, _ = fmt.Fprint(w, `{"coverage":"Workspace name, repositories, stages, and automations","changed":true}`)
		default:
			t.Errorf("unexpected API call: %s", r.URL)
			http.NotFound(w, r)
		}
	})
	root.SetArgs([]string{"apply", "--workspace", "example", "-f", path, "--yes"})
	require.NoError(t, root.Execute())
	assert.Len(t, requests, 2)
	assert.Contains(t, stdout.String(), "old")
	assert.Contains(t, stdout.String(), "Follow AGENTS.md.")
}

func TestConfigurationRoundTripsCustomFilesAcrossStages(t *testing.T) {
	manifest := strings.Replace(configurationManifest, `        with:
          agent:
            additional_instructions: Follow AGENTS.md.
`, `        binding:
          integration_id: repository-connection
          resource_id: repository-resource
        with:
          agent:
            timeout: 0s
            skip_setup_commands: false
          template:
            draft_pull_requests: false
`, 1) + `  - id: custom-review
    name: Review changes
    automations:
      - id: review
        name: Review changes
        uses: automations/review.yaml
  - id: verify
    name: Verify
    automations:
      - id: confidence
        name: Merge confidence
        uses: "@superplane/merge-confidence"
        with:
          template:
            enabled_checks: []
            publish_results: false
      - id: review-request
        name: Review request
        uses: "@superplane/review-request"
        with:
          template:
            merge_confidence:
              stage: verify
              automation: confidence
            fallback_reviewer: ""
      - id: discussion
        name: Pull request discussion
        uses: "@superplane/pr-discussion"
        with:
          pr_feedback:
            mention: ""
            allowed_bots: []
  - id: backlog
    name: Backlog
    automations:
      - id: issues
        name: GitHub issues
        uses: "@superplane/intake/github-issues"
        with:
          intake:
            confidence_pct: 0
            labels: []
            new_issues: false
      - id: linear
        name: Linear issues
        uses: "@superplane/intake/linear-issues"
        binding:
          integration_name: "Engineering Linear"
          team_id: engineering
  - id: done
    name: Done
    automations: []
repositories:
  binding:
    integration_name: "Engineering GitHub"
  primary:
    repository: acme/application
    ref: main
  backlog_repository: acme/backlog
  additional:
    - repository: acme/library
      access: write
    - repository: acme/docs
      ref: stable
      access: read
      enabled: false
      provider: github
`
	source := "apiVersion: v2\nkind: Automation\nname: Review changes\ntriggers:\n  - uses: manual\nsteps:\n  - name: Review\n    prompt: Review the changes.\n"
	output := t.TempDir()
	var planned map[string]any
	root, _ := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			payload := exportPayload(manifest)
			bundle := payload["bundle"].(map[string]any)
			bundle["files"] = append(bundle["files"].([]map[string]any), map[string]any{"path": ".superplane/automations/review.yaml", "content": source})
			require.NoError(t, json.NewEncoder(w).Encode(payload))
			return
		}
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		if strings.HasSuffix(r.URL.Path, ":plan") {
			planned = body["bundle"].(map[string]any)
			files := planned["files"].([]any)
			require.Len(t, files, 2)
			assert.Equal(t, manifest, files[0].(map[string]any)["content"])
			assert.Equal(t, "automations/review.yaml", files[1].(map[string]any)["path"])
			assert.Equal(t, source, files[1].(map[string]any)["content"])
			require.NoError(t, os.WriteFile(filepath.Join(output, ".superplane", "automations", "review.yaml"), []byte("changed after preview"), 0600))
			_, _ = fmt.Fprint(w, `{"coverage":"Workspace name, repositories, stages, and automations","baseStateToken":"preview-token","changes":[{"resource":"custom-review/review","fieldPath":"steps","previousValue":"old","requestedValue":"new"}]}`)
			return
		}
		assert.Equal(t, planned, body["bundle"])
		_, _ = fmt.Fprint(w, `{"coverage":"Workspace name, repositories, stages, and automations","changed":true}`)
	})
	root.SetArgs([]string{"export", "--workspace", "example", "--output-dir", output})
	require.NoError(t, root.Execute())
	actual, err := os.ReadFile(filepath.Join(output, ".superplane", "automations", "review.yaml"))
	require.NoError(t, err)
	require.Equal(t, source, string(actual))
	t.Chdir(output)
	root.SetArgs([]string{"apply", "--workspace", "example", "--yes"})
	require.NoError(t, root.Execute())
}

func TestConfigurationRejectsUnsafeCustomFiles(t *testing.T) {
	for _, scenario := range []string{"traversal", "absolute", "backslash", "symlink", "missing", "stdin"} {
		t.Run(scenario, func(t *testing.T) {
			directory := t.TempDir()
			reference := "automations/custom.yaml"
			switch scenario {
			case "traversal":
				reference = "../custom.yaml"
			case "absolute":
				reference = "/tmp/custom.yaml"
			case "backslash":
				reference = "..\\custom.yaml"
			case "symlink":
				require.NoError(t, os.Symlink(t.TempDir(), filepath.Join(directory, "automations")))
			}
			manifest := strings.Replace(configurationManifest, "\"@superplane/implement\"", "'"+reference+"'", 1)
			input := filepath.Join(directory, "main.yml")
			require.NoError(t, os.WriteFile(input, []byte(manifest), 0600))
			root, _ := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) { t.Errorf("unsafe input reached API: %s", r.URL) })
			if scenario == "stdin" {
				input = "-"
				root.SetIn(strings.NewReader(manifest))
			}
			root.SetArgs([]string{"plan", "--workspace", "example", "-f", input})
			require.Error(t, root.Execute())
		})
	}
}

func TestExportConfigurationChecksEveryFileBeforeReplacement(t *testing.T) {
	manifest := strings.Replace(configurationManifest, "\"@superplane/implement\"", "automations/custom.yaml", 1)
	for _, scenario := range []string{"unsafe path", "missing referenced file", "duplicate path", "symlink", "existing custom file"} {
		t.Run(scenario, func(t *testing.T) {
			output := t.TempDir()
			directory := filepath.Join(output, ".superplane")
			require.NoError(t, os.MkdirAll(filepath.Join(directory, "automations"), 0755))
			manifestPath := filepath.Join(directory, "main.yml")
			payload := exportPayload(manifest)
			bundle := payload["bundle"].(map[string]any)
			files := bundle["files"].([]map[string]any)
			custom := map[string]any{"path": ".superplane/automations/custom.yaml", "content": "apiVersion: v2\nkind: Automation\n"}
			args := []string{"export", "--workspace", "example", "--output-dir", output, "--overwrite"}
			if scenario != "existing custom file" {
				require.NoError(t, os.WriteFile(manifestPath, []byte("keep manifest"), 0600))
			}
			switch scenario {
			case "unsafe path":
				custom["path"] = "../custom.yaml"
			case "duplicate path":
				files = append(files, custom)
			case "symlink":
				require.NoError(t, os.Remove(filepath.Join(directory, "automations")))
				require.NoError(t, os.Symlink(t.TempDir(), filepath.Join(directory, "automations")))
			case "existing custom file":
				require.NoError(t, os.WriteFile(filepath.Join(directory, "automations", "custom.yaml"), []byte("keep custom"), 0600))
				args = args[:len(args)-1]
			}
			if scenario != "missing referenced file" {
				files = append(files, custom)
			}
			bundle["files"] = files
			root, _ := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(payload))
			})
			root.SetArgs(args)
			require.Error(t, root.Execute())
			if scenario == "existing custom file" {
				_, err := os.Stat(manifestPath)
				require.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			actual, err := os.ReadFile(manifestPath)
			require.NoError(t, err)
			require.Equal(t, "keep manifest", string(actual))
		})
	}
}

func TestConfigurationPlanAndApplyStopBeforePublishing(t *testing.T) {
	for _, scenario := range []string{"plan", "unchanged", "diagnostics", "confirmation unavailable", "missing token"} {
		t.Run(scenario, func(t *testing.T) {
			var calls int
			root, stdout := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				assert.Equal(t, "/api/v1/factories/example/configuration:plan", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				response := map[string]any{"coverage": "Workspace name, repositories, stages, and automations", "baseStateToken": "token"}
				if scenario != "unchanged" {
					response["changes"] = []map[string]string{{"resource": "implementation/implement", "fieldPath": "with.agent.additional_instructions", "previousValue": "old", "requestedValue": "new"}}
				}
				if scenario == "diagnostics" {
					response["diagnostics"] = []map[string]string{{"path": ".superplane/main.yml", "field": "stages", "message": "Unsupported stage."}}
				}
				if scenario == "missing token" {
					delete(response, "baseStateToken")
				}
				require.NoError(t, json.NewEncoder(w).Encode(response))
			})
			operation := "apply"
			if scenario == "plan" {
				operation = "plan"
			}
			root.SetArgs([]string{operation, "--workspace", "example", "-f", writeConfigurationInput(t)})
			err := root.Execute()
			switch scenario {
			case "plan", "unchanged":
				require.NoError(t, err)
			case "diagnostics":
				require.Error(t, err)
				assert.Contains(t, stdout.String(), ".superplane/main.yml")
				assert.Contains(t, stdout.String(), "stages")
			case "confirmation unavailable":
				require.ErrorContains(t, err, "--yes")
			case "missing token":
				require.ErrorContains(t, err, "token")
			}
			assert.Equal(t, 1, calls)
		})
	}
}

func TestConfigurationPlanReadsManifestFromCurrentDirectory(t *testing.T) {
	for _, scenario := range []string{"default", "override", "missing"} {
		t.Run(scenario, func(t *testing.T) {
			directory := t.TempDir()
			t.Chdir(directory)
			require.NoError(t, os.Mkdir(".superplane", 0755))
			args := []string{"plan", "--workspace", "example"}
			switch scenario {
			case "default":
				require.NoError(t, os.WriteFile(".superplane/main.yml", []byte(configurationManifest), 0600))
			case "override":
				require.NoError(t, os.WriteFile(".superplane/main.yml", []byte("invalid manifest"), 0600))
				args = append(args, "-f", writeConfigurationInput(t))
			}
			calls := 0
			root, _ := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				assert.Equal(t, "/api/v1/factories/example/configuration:plan", r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				bundle := body["bundle"].(map[string]any)
				assert.Equal(t, "main.yml", bundle["manifestPath"])
				files := bundle["files"].([]any)
				require.Len(t, files, 1)
				assert.Equal(t, configurationManifest, files[0].(map[string]any)["content"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"coverage":"Workspace name, repositories, stages, and automations"}`)
			})
			root.SetArgs(args)
			err := root.Execute()
			if scenario == "missing" {
				require.ErrorContains(t, err, ".superplane/main.yml")
				assert.Zero(t, calls)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, 1, calls)
		})
	}
}

// A shared source can serve independent manifests while only selected files reach the API.
func TestConfigurationPackagesSelectedManifestWithinRoot(t *testing.T) {
	for _, folder := range []string{".superplane", ".superplane/workspaces"} {
		t.Run(folder, func(t *testing.T) {
			directory := t.TempDir()
			t.Chdir(directory)
			require.NoError(t, os.MkdirAll(folder, 0755))
			require.NoError(t, os.MkdirAll(".superplane/automations", 0755))
			reference := "automations/review.yml"
			if folder != ".superplane" {
				reference = "../automations/review.yml"
			}
			manifest := strings.Replace(configurationManifest, "\"@superplane/implement\"", reference, 1)
			manifest = strings.Replace(manifest, "run: implement", "run: second-review", 1)
			manifest += "  - id: backlog\n    name: Backlog\n    automations:\n      - id: second-review\n        uses: " + strings.Replace(reference, "review.yml", "./review.yml", 1) + "\n"
			manifest += "  - id: second-phase\n    name: Second phase\n    run: second-review\n    automations: []\n"
			require.NoError(t, os.WriteFile(filepath.Join(folder, "workspace-1.yml"), []byte(manifest), 0600))
			require.NoError(t, os.WriteFile(filepath.Join(folder, "workspace-2.yml"), []byte("unrelated manifest"), 0600))
			source := "apiVersion: v2\nkind: Automation\nname: Shared review\ntriggers:\n  - uses: manual\nsteps:\n  - name: Review\n    bash: echo review\n"
			require.NoError(t, os.WriteFile(".superplane/automations/review.yml", []byte(source), 0600))
			calls := 0
			root, _ := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				bundle := body["bundle"].(map[string]any)
				assert.Equal(t, folder+"/workspace-1.yml", bundle["manifestPath"])
				files := bundle["files"].([]any)
				require.Len(t, files, 2)
				assert.Equal(t, folder+"/workspace-1.yml", files[0].(map[string]any)["path"])
				assert.Equal(t, manifest, files[0].(map[string]any)["content"])
				assert.Equal(t, ".superplane/automations/review.yml", files[1].(map[string]any)["path"])
				assert.Equal(t, source, files[1].(map[string]any)["content"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"coverage":"Workspace name, repositories, stages, and automations"}`)
			})
			root.SetArgs([]string{"plan", "--workspace", "example", "--root", ".", "-f", folder + "/workspace-1.yml"})
			require.NoError(t, root.Execute())
			assert.Equal(t, 1, calls)
		})
	}
}

func TestConfigurationRejectsUnsafeManifestInputs(t *testing.T) {
	for _, scenario := range []string{"manifest symlink", "parent symlink", "directory", "outside root", "reference directory", "self reference", "old stage key", "stage concurrency", "duplicate automation ID"} {
		t.Run(scenario, func(t *testing.T) {
			directory := t.TempDir()
			manifest := configurationManifest
			input := filepath.Join(directory, "workspace.yml")
			args := []string{"plan", "--workspace", "example", "-f", input}
			switch scenario {
			case "manifest symlink":
				target := filepath.Join(t.TempDir(), "workspace.yml")
				require.NoError(t, os.WriteFile(target, []byte(manifest), 0600))
				require.NoError(t, os.Symlink(target, input))
			case "parent symlink":
				target := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(target, "workspace.yml"), []byte(manifest), 0600))
				require.NoError(t, os.Symlink(target, filepath.Join(directory, "linked")))
				args = []string{"plan", "--workspace", "example", "-f", filepath.Join(directory, "linked", "workspace.yml")}
			case "directory":
				require.NoError(t, os.Mkdir(input, 0755))
			case "outside root":
				args = append(args, "--root", t.TempDir())
			case "reference directory":
				manifest = strings.Replace(manifest, "\"@superplane/implement\"", "automations/review.yml", 1)
				require.NoError(t, os.MkdirAll(filepath.Join(directory, "automations", "review.yml"), 0755))
			case "self reference":
				manifest = strings.Replace(manifest, "\"@superplane/implement\"", "./workspace.yml", 1)
			case "old stage key":
				manifest = strings.Replace(manifest, "  - id: implementation", "  - key: implementation", 1)
			case "stage concurrency":
				manifest = strings.Replace(manifest, "    run: implement", "    max_parallelism: 20\n    run: implement", 1)
			case "duplicate automation ID":
				manifest += "  - id: backlog\n    name: Backlog\n    automations:\n      - id: implement\n        uses: '@superplane/implement'\n"
			}
			if scenario != "manifest symlink" && scenario != "parent symlink" && scenario != "directory" {
				require.NoError(t, os.WriteFile(input, []byte(manifest), 0600))
			}
			root, _ := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) { t.Errorf("unsafe input reached API: %s", r.URL) })
			root.SetArgs(args)
			require.Error(t, root.Execute())
		})
	}
}

func TestConfigurationReadsStdinAndFilesystemRoot(t *testing.T) {
	for _, scenario := range []string{"stdin", "filesystem root"} {
		t.Run(scenario, func(t *testing.T) {
			args := []string{"plan", "--workspace", "example", "-f", "-"}
			if scenario == "filesystem root" {
				args = []string{"plan", "--workspace", "example", "--root", "/", "-f", writeConfigurationInput(t)}
			}
			calls := 0
			root, _ := configurationCLI(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, `{"coverage":"Workspace name, repositories, stages, and automations"}`)
			})
			root.SetIn(strings.NewReader(configurationManifest))
			root.SetArgs(args)
			require.NoError(t, root.Execute())
			assert.Equal(t, 1, calls)
		})
	}
}
