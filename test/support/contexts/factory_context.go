package contexts

import "github.com/superplanehq/superplane/pkg/core"

// FactoryContext is a test double for core.FactoryContext.
// Only DefaultAgentModel returns a configured value. Other methods return zero values.
type FactoryContext struct {
	Default core.FactoryDefaultAgentModel
	Err     error
}

func (f *FactoryContext) DefaultAgentModel() (core.FactoryDefaultAgentModel, error) {
	if f == nil {
		return core.FactoryDefaultAgentModel{}, nil
	}
	if f.Err != nil {
		return core.FactoryDefaultAgentModel{}, f.Err
	}
	return f.Default, nil
}

func (f *FactoryContext) VCSProvider() (string, error) { return "", nil }

func (f *FactoryContext) CreateWorkOrder(core.WorkOrderParams) (*core.WorkOrder, bool, error) {
	return nil, false, nil
}

func (f *FactoryContext) FindWorkOrder(core.FindWorkOrderParams) (*core.WorkOrder, error) {
	return nil, nil
}

func (f *FactoryContext) UpdateWorkOrderStatus(core.UpdateWorkOrderStatusParams) (*core.WorkOrder, bool, error) {
	return nil, false, nil
}

func (f *FactoryContext) AddWorkOrderComment(core.AddWorkOrderCommentParams) error { return nil }

func (f *FactoryContext) AddWorkOrderArtifact(core.AddWorkOrderArtifactParams) (*core.WorkOrderArtifact, error) {
	return nil, nil
}

func (f *FactoryContext) ReportWorkOrderCheck(core.ReportWorkOrderCheckParams) (*core.WorkOrderCheck, error) {
	return nil, nil
}

func (f *FactoryContext) SetWorkOrderStatusNote(core.SetWorkOrderStatusNoteParams) (*core.WorkOrderStatusNote, error) {
	return nil, nil
}

func (f *FactoryContext) AddPullRequest(core.AddPullRequestParams) (*core.PullRequest, error) {
	return nil, nil
}

func (f *FactoryContext) UpdatePullRequest(core.UpdatePullRequestParams) (*core.PullRequest, error) {
	return nil, nil
}

func (f *FactoryContext) FindPullRequest(core.FindPullRequestParams) (*core.PullRequestMatch, error) {
	return nil, nil
}

func (f *FactoryContext) AddPullRequestActivity(core.AddPullRequestActivityParams) (*core.PullRequestActivityResult, error) {
	return nil, nil
}

func (f *FactoryContext) UpdatePullRequestActivity(core.UpdatePullRequestActivityParams) (*core.PullRequestActivityResult, error) {
	return nil, nil
}
