package contexts

import (
	"bytes"
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/jira"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/storedfiles"
)

type jiraFileRead func(ctx context.Context, issueKey, description string) ([]jira.IssueFile, error)

func (c *FactoryContext) WithJiraIssueFiles(read jiraFileRead) *FactoryContext {
	c.readJiraIssueFiles = read
	return c
}

func (c *FactoryContext) ingestJiraFiles(order *models.FactoryWorkOrder) {
	if order == nil {
		return
	}
	issueKey, ok := c.jiraIssueKey(order)
	if !ok {
		return
	}

	issueFiles, err := c.readJiraFiles(context.Background(), issueKey, order.Description)
	if err != nil || len(issueFiles) == 0 {
		if err != nil {
			log.WithError(err).Warn("failed to read Jira issue files")
		}
		return
	}

	next, err := storedfiles.AppendTaskFiles(
		context.Background(),
		c.tx,
		blob.Current(),
		order.OrganizationID,
		order.FactoryID,
		order.ID,
		nil,
		order.Description,
		incomingJiraFiles(issueFiles),
	)
	if err != nil || next.Markdown == order.Description {
		if len(next.ObjectKeys) > 0 {
			_ = storedfiles.SweepObjects(
				context.Background(),
				database.Conn(),
				blob.Current(),
				order.OrganizationID,
				order.FactoryID,
				next.ObjectKeys,
			)
		}
		return
	}
	if err := order.UpdateContent(c.tx, nil, &next.Markdown); err != nil {
		_ = storedfiles.SweepObjects(
			context.Background(),
			database.Conn(),
			blob.Current(),
			order.OrganizationID,
			order.FactoryID,
			next.ObjectKeys,
		)
	}
}

func (c *FactoryContext) readJiraFiles(ctx context.Context, issueKey, description string) ([]jira.IssueFile, error) {
	if c.readJiraIssueFiles != nil {
		return c.readJiraIssueFiles(ctx, issueKey, description)
	}
	client := c.jiraClientForCanvas()
	if client == nil {
		return nil, nil
	}
	return client.IssueFiles(ctx, issueKey, description)
}

func (c *FactoryContext) jiraIssueKey(order *models.FactoryWorkOrder) (string, bool) {
	runID := order.SourceRunID
	if runID == nil && c.execution != nil {
		runID = &c.execution.RunID
	}
	if runID == nil {
		return "", false
	}
	event, err := models.FindRootEventForRun(c.tx, *runID)
	if err != nil || event == nil {
		return "", false
	}
	return jira.IssueKeyFromEventData(event.Data.Data())
}

func (c *FactoryContext) jiraClientForCanvas() *jira.Client {
	if c.registry == nil || c.encryptor == nil {
		return nil
	}
	specs, err := models.FindLiveCanvasSpecsByCanvasIDs(c.tx, []uuid.UUID{c.canvas.ID})
	if err != nil {
		return nil
	}
	spec, ok := specs[c.canvas.ID]
	if !ok {
		return nil
	}
	for i := range spec.Nodes {
		node := spec.Nodes[i]
		if node.ComponentName() != "jira.onIssue" || node.IntegrationID == nil {
			continue
		}
		integrationID, err := uuid.Parse(strings.TrimSpace(*node.IntegrationID))
		if err != nil {
			continue
		}
		integration, err := models.FindIntegrationInTransaction(c.tx, c.canvas.OrganizationID, integrationID)
		if err != nil || integration.State != models.IntegrationStateReady {
			continue
		}
		client, err := jira.NewClient(
			c.registry.HTTPContextInTransaction(c.tx),
			NewIntegrationContext(c.tx, nil, integration, c.encryptor, c.registry, nil),
		)
		if err != nil {
			continue
		}
		return client
	}
	return nil
}

func incomingJiraFiles(files []jira.IssueFile) []storedfiles.IncomingFile {
	incoming := make([]storedfiles.IncomingFile, 0, len(files))
	for _, file := range files {
		incoming = append(incoming, storedfiles.IncomingFile{
			Filename:    file.Name,
			ContentType: file.ContentType,
			Body:        bytes.NewReader(file.Body),
			ReplaceURLs: file.ReplaceURLs,
		})
	}
	return incoming
}
