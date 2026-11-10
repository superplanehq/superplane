package models

import "gorm.io/gorm"

// AdmitHumanReviewRun applies task authority before preparation can start an agent.
// The caller owns the transaction and commits permanent rejection status notes.
func (p *FactoryPullRequest) AdmitHumanReviewRun(tx *gorm.DB, canvas *Canvas, run *CanvasRun) error {
	if canvas == nil || run == nil || canvas.FactoryID == nil ||
		canvas.OrganizationID != p.OrganizationID || *canvas.FactoryID != p.FactoryID || run.WorkflowID != canvas.ID {
		return ErrSubscriptionTaskAssociation
	}
	factory, err := FindFactory(tx, p.OrganizationID, p.FactoryID)
	if err != nil {
		return err
	}
	order, err := factory.FindWorkOrder(tx, p.WorkOrderID)
	if err != nil {
		return err
	}
	const title = "Request review"
	if err := order.RequireExecutionOwner(tx, true); err != nil {
		return recordColumnAppRejection(tx, order, title, err)
	}
	if err := order.ValidateSubscriptionCanvasModels(tx, canvas.ID, ""); err != nil {
		return recordColumnAppRejection(tx, order, title, err)
	}
	var revisionSHA string
	if p.CurrentRevisionID != nil {
		revision, err := FindPullRequestRevision(tx, *p.CurrentRevisionID)
		if err != nil {
			return err
		}
		revisionSHA = revision.SHA
	}
	_, err = p.CreateActivity(tx, FactoryPullRequestActivityParams{
		RunID: run.ID, Title: title, RevisionSHA: revisionSHA, Access: FactoryPullRequestAccessConcurrent,
	})
	return err
}
