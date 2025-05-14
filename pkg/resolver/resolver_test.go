package resolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/test/support"
)

func Test__Resolve(t *testing.T) {
	r := support.Setup(t)

	t.Run("no variables to resolve", func(t *testing.T) {
		execution := support.CreateExecutionWithData(t, r.Source, r.Stage, []byte(`{"ref":"v1","data": {"branch": "hello"}}`), []byte(`{"ref":"v1","data": {"branch": "hello"}}`))
		template := support.RunTemplate()
		resolver := NewResolver(*execution, template)
		newTemplate, err := resolver.Resolve()
		require.NoError(t, err)
		require.NotNil(t, newTemplate)
		assert.Equal(t, models.RunTemplateTypeSemaphore, newTemplate.Type)
		assert.Equal(t, "demo-project", newTemplate.Semaphore.ProjectID)
		assert.Equal(t, "demo-task", newTemplate.Semaphore.TaskID)
		assert.Equal(t, ".semaphore/run.yml", newTemplate.Semaphore.PipelineFile)
		assert.Equal(t, "main", newTemplate.Semaphore.Branch)
		assert.Equal(t, map[string]string{
			"PARAM_1": "VALUE_1",
			"PARAM_2": "VALUE_2",
		}, newTemplate.Semaphore.Parameters)
	})

	t.Run("with variables to resolve", func(t *testing.T) {
		e := `{"ref":"refs/heads/hello","branch":"hello","project":"other","param1":"value1","param2":"value2"}`
		execution := support.CreateExecutionWithData(t, r.Source, r.Stage, []byte(e), []byte(`{}`))
		template := support.RunTemplate()
		template.Semaphore.Branch = "${{self.Conn('gh').branch}}"
		template.Semaphore.ProjectID = "${{self.Conn('gh').project}}"
		template.Semaphore.Parameters = map[string]string{
			"PARAM_1": "${{self.Conn('gh').param1}}",
			"PARAM_2": "${{self.Conn('gh').param2}}",
		}

		resolver := NewResolver(*execution, template)
		newTemplate, err := resolver.Resolve()
		require.NoError(t, err)
		require.NotNil(t, newTemplate)
		assert.Equal(t, models.RunTemplateTypeSemaphore, newTemplate.Type)
		assert.Equal(t, "other", newTemplate.Semaphore.ProjectID)
		assert.Equal(t, "hello", newTemplate.Semaphore.Branch)
		assert.Equal(t, ".semaphore/run.yml", newTemplate.Semaphore.PipelineFile)
		assert.Equal(t, "demo-task", newTemplate.Semaphore.TaskID)
		assert.Equal(t, map[string]string{
			"PARAM_1": "value1",
			"PARAM_2": "value2",
		}, newTemplate.Semaphore.Parameters)
	})
}
