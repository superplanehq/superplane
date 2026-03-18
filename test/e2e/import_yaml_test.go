package e2e

import (
	"os"
	"path/filepath"
	"testing"

	pw "github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	q "github.com/superplanehq/superplane/test/e2e/queries"
	"github.com/superplanehq/superplane/test/e2e/session"
)

func TestImportYaml(t *testing.T) {
	t.Run("importing a canvas from pasted YAML", func(t *testing.T) {
		steps := &ImportYamlSteps{t: t}
		steps.start()
		steps.visitHomePage()
		steps.openImportDialog()
		steps.pasteYaml(validCanvasYaml("Imported Canvas"))
		steps.submitImport()
		steps.assertCanvasSavedInDB("Imported Canvas")
	})

	t.Run("importing a canvas from a YAML file", func(t *testing.T) {
		steps := &ImportYamlSteps{t: t}
		steps.start()
		steps.visitHomePage()
		steps.openImportDialog()
		steps.uploadYamlFile("test-canvas.yaml", validCanvasYaml("File Import Canvas"))
		steps.submitImport()
		steps.assertCanvasSavedInDB("File Import Canvas")
	})

	t.Run("showing error for invalid YAML syntax", func(t *testing.T) {
		steps := &ImportYamlSteps{t: t}
		steps.start()
		steps.visitHomePage()
		steps.openImportDialog()
		steps.pasteYaml("invalid: yaml: [unterminated")
		steps.submitImport()
		steps.assertImportErrorVisible()
	})

	t.Run("showing error when metadata name is missing", func(t *testing.T) {
		steps := &ImportYamlSteps{t: t}
		steps.start()
		steps.visitHomePage()
		steps.openImportDialog()
		steps.pasteYaml("apiVersion: v1\nkind: Canvas\nmetadata:\n  description: no name\nspec:\n  nodes: []\n  edges: []")
		steps.submitImport()
		steps.assertImportErrorVisible()
	})
}

func validCanvasYaml(name string) string {
	return "apiVersion: v1\nkind: Canvas\nmetadata:\n  name: " + name + "\nspec:\n  nodes: []\n  edges: []"
}

type ImportYamlSteps struct {
	t       *testing.T
	session *session.TestSession
}

func (s *ImportYamlSteps) start() {
	s.session = ctx.NewSession(s.t)
	s.session.Start()
	s.session.Login()
}

func (s *ImportYamlSteps) visitHomePage() {
	s.session.VisitHomePage()
	s.session.Sleep(500)
}

func (s *ImportYamlSteps) openImportDialog() {
	s.session.Click(q.TestID("import-yaml-button"))
	s.session.Sleep(500)
}

func (s *ImportYamlSteps) pasteYaml(yamlContent string) {
	s.session.FillIn(q.TestID("import-yaml-textarea"), yamlContent)
}

func (s *ImportYamlSteps) uploadYamlFile(filename string, content string) {
	tmpDir := s.t.TempDir()
	filePath := filepath.Join(tmpDir, filename)
	err := os.WriteFile(filePath, []byte(content), 0o644)
	require.NoError(s.t, err)

	fileInput := q.TestID("import-yaml-file-input").Run(s.session)
	err = fileInput.SetInputFiles(pw.InputFile{Name: filename, Buffer: []byte(content)})
	require.NoError(s.t, err)

	s.session.Sleep(500)
}

func (s *ImportYamlSteps) submitImport() {
	s.session.Click(q.TestID("import-yaml-submit"))
	s.session.Sleep(1000)
}

func (s *ImportYamlSteps) assertCanvasSavedInDB(canvasName string) {
	canvas, err := models.FindCanvasByName(canvasName, s.session.OrgID)
	assert.NoError(s.t, err)
	assert.Equal(s.t, canvasName, canvas.Name)
}

func (s *ImportYamlSteps) assertImportErrorVisible() {
	s.session.AssertVisible(q.TestID("import-yaml-error"))
}
