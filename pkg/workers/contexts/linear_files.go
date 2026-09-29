package contexts

import (
	"bytes"
	"context"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/blob"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/integrations/linear"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/storedfiles"
)

type linearFileRead func(ctx context.Context, issueID, description string) ([]linear.IssueFile, []linear.IssueLink, error)

func (c *FactoryContext) WithLinearIssueFiles(read linearFileRead) *FactoryContext {
	c.readLinearIssueFiles = read
	return c
}

func (c *FactoryContext) skipDuplicateLinearWorkOrder(factoryModel *models.Factory) (bool, error) {
	if c.execution == nil {
		return false, nil
	}

	event, err := models.FindRootEventForRun(c.tx, c.execution.RunID)
	if err != nil {
		return false, nil
	}

	ref, ok := linear.IssueRefFromEventData(event.Data.Data())
	if !ok {
		return false, nil
	}

	if err := linear.LockIssueWorkOrder(c.tx, factoryModel, ref); err != nil {
		return false, err
	}

	hasOrder, err := linear.IssueHasWorkOrder(c.tx, factoryModel, ref)
	if err != nil {
		return false, err
	}
	if hasOrder {
		log.Infof("skipping Linear issue %s: work order already exists", ref.Identifier)
	}
	return hasOrder, nil
}

func (c *FactoryContext) ingestLinearFiles(order *models.FactoryWorkOrder) {
	if order == nil {
		return
	}
	issueID, ok := c.linearIssueID(order)
	if !ok {
		return
	}

	issueFiles, links, err := c.readLinearFiles(context.Background(), issueID, order.Description)
	if err != nil {
		log.WithError(err).Warn("failed to read Linear issue files")
		return
	}

	description := order.Description
	if section := linear.LinkSectionMarkdown(links); section != "" && !strings.Contains(description, section) {
		if strings.TrimSpace(description) == "" {
			description = section
		} else {
			description = description + "\n\n" + section
		}
	}

	if len(issueFiles) == 0 {
		if description == order.Description {
			return
		}
		_ = order.UpdateContent(c.tx, nil, &description)
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
		description,
		incomingLinearFiles(issueFiles),
	)
	if err != nil || (next.Markdown == order.Description && description == order.Description) {
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
	markdown := next.Markdown
	if markdown == "" {
		markdown = description
	}
	if err := order.UpdateContent(c.tx, nil, &markdown); err != nil {
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

func (c *FactoryContext) readLinearFiles(ctx context.Context, issueID, description string) ([]linear.IssueFile, []linear.IssueLink, error) {
	if c.readLinearIssueFiles != nil {
		return c.readLinearIssueFiles(ctx, issueID, description)
	}
	client := c.linearClientForCanvas()
	if client == nil {
		return nil, nil, nil
	}
	return client.IssueFiles(ctx, issueID, description)
}

func (c *FactoryContext) linearIssueID(order *models.FactoryWorkOrder) (string, bool) {
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
	return linear.IssueIDFromEventData(event.Data.Data())
}

func (c *FactoryContext) linearClientForCanvas() *linear.Client {
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
		if node.ComponentName() != "linear.onIssue" || node.IntegrationID == nil {
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
		client, err := linear.NewClient(
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

func incomingLinearFiles(files []linear.IssueFile) []storedfiles.IncomingFile {
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
