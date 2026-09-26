package factories

import (
	"errors"

	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

func factoryErrorToStatus(err error, internalMessage string) error {
	if _, _, ok := grpcerrors.HandlerStatus(err); ok {
		return err
	}

	switch {
	case errors.Is(err, models.ErrFactoryNameAlreadyExists):
		return grpcerrors.AlreadyExists(err, "factory with the same name already exists")
	case errors.Is(err, models.ErrFactoryNameRequired):
		return grpcerrors.InvalidArgument(err, "factory name is required")
	case errors.Is(err, models.ErrFactoryKeyRequired):
		return grpcerrors.InvalidArgument(err, "workspace key is required")
	case errors.Is(err, models.ErrFactoryKeyInvalid):
		return grpcerrors.InvalidArgument(err, "workspace key must be 2 to 5 uppercase letters")
	case errors.Is(err, models.ErrFactoryKeyAlreadyExists):
		return grpcerrors.AlreadyExists(err, "workspace key already exists in this organization")
	case errors.Is(err, models.ErrFactoryNotFound):
		return grpcerrors.NotFound(err, "factory not found")
	case errors.Is(err, models.ErrFactoryHostedSpendBudgetNegative):
		return grpcerrors.InvalidArgument(err, "hosted spend limit cannot be negative")
	case errors.Is(err, models.ErrModelNotInParentList):
		return grpcerrors.InvalidArgument(err, "model is not in the parent selected-model list")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidIssuesSource):
		return grpcerrors.InvalidArgument(err, "invalid issues source")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidAgentHarness):
		return grpcerrors.InvalidArgument(err, "invalid agent harness")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidIntegrationID):
		return grpcerrors.InvalidArgument(err, "invalid integration id")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidAppID):
		return grpcerrors.InvalidArgument(err, "invalid provisioned app id")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidLineID):
		return grpcerrors.InvalidArgument(err, "invalid provisioned line id")
	case errors.Is(err, models.ErrFactoryOnboardingInvalidRepository):
		return grpcerrors.InvalidArgument(err, "repository must use the owner/name format")
	case errors.Is(err, models.ErrFactoryOnboardingVCSIntegrationRequired):
		return grpcerrors.InvalidArgument(err, "version control integration is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingAgentIntegrationRequired):
		return grpcerrors.InvalidArgument(err, "agent integration or hosted credit is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingHostedAgentUnavailable):
		return grpcerrors.InvalidArgument(err, "agent integration or SuperPlane-hosted models are required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingAppRepositoryRequired):
		return grpcerrors.InvalidArgument(err, "app repository is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingBacklogRepoRequired):
		return grpcerrors.InvalidArgument(err, "backlog repository is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingIssuesSourceRequired):
		return grpcerrors.InvalidArgument(err, "issues source is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingAgentHarnessRequired):
		return grpcerrors.InvalidArgument(err, "agent harness is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingAppIDRequired):
		return grpcerrors.InvalidArgument(err, "provisioned app id is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryOnboardingLineIDRequired):
		return grpcerrors.InvalidArgument(err, "provisioned line id is required to complete onboarding")
	case errors.Is(err, models.ErrFactoryWorkOrderNotFound):
		return grpcerrors.NotFound(err, "work order not found")
	case errors.Is(err, models.ErrFactoryLineNotFound):
		return grpcerrors.NotFound(err, "factory line not found")
	case errors.Is(err, models.ErrFactoryLineNameAlreadyExists):
		return grpcerrors.AlreadyExists(err, "factory line with the same name already exists")
	case errors.Is(err, models.ErrFactoryWorkOrderNotDispatchable):
		return grpcerrors.FailedPrecondition(err, "work order cannot be dispatched in its current state")
	case errors.Is(err, models.ErrFactoryWorkOrderInvalidState):
		return grpcerrors.FailedPrecondition(err, err.Error())
	case errors.Is(err, models.ErrFactoryWorkOrderLineDispatchActive):
		return grpcerrors.FailedPrecondition(err, "work order already has an active line dispatch")
	case errors.Is(err, models.ErrFactoryLineHasNoSteps):
		return grpcerrors.FailedPrecondition(err, "factory line has no steps")
	case errors.Is(err, models.ErrFactoryLineStepOutOfRange):
		return grpcerrors.InvalidArgument(err, "factory line step index is out of range")
	case errors.Is(err, models.ErrFactoryLineStepNotOnRun):
		return grpcerrors.FailedPrecondition(err, "factory line step entrypoint must use the onRun trigger")
	case errors.Is(err, models.ErrFactoryWorkOrderArtifactInvalid):
		return grpcerrors.InvalidArgument(err, err.Error())
	case errors.Is(err, models.ErrFactoryIntakeNotFound):
		return grpcerrors.NotFound(err, "factory intake not found")
	case errors.Is(err, models.ErrFactoryIntakeSourceInvalid):
		return grpcerrors.InvalidArgument(err, "intake source is not supported")
	case errors.Is(err, models.ErrFactoryIntakeCanvasInUse):
		return grpcerrors.AlreadyExists(err, "canvas already implements an intake")
	case errors.Is(err, models.ErrFactoryIntakeCanvasRequired):
		return grpcerrors.InvalidArgument(err, "intake canvas is required")
	case errors.Is(err, models.ErrFactoryPRFeedbackHandlerNotFound):
		return grpcerrors.NotFound(err, "factory PR feedback handler not found")
	case errors.Is(err, models.ErrFactoryPRFeedbackHandlerSubjectInvalid):
		return grpcerrors.InvalidArgument(err, "PR feedback handler subject is not supported")
	case errors.Is(err, models.ErrFactoryPRFeedbackHandlerSourceInvalid):
		return grpcerrors.InvalidArgument(err, "PR feedback handler source is not supported")
	case errors.Is(err, models.ErrFactoryPRFeedbackHandlerCanvasInUse):
		return grpcerrors.AlreadyExists(err, "canvas already implements a PR feedback handler")
	case errors.Is(err, models.ErrFactoryPRFeedbackHandlerCanvasRequired):
		return grpcerrors.InvalidArgument(err, "PR feedback handler canvas is required")
	case errors.Is(err, models.ErrFactoryPullRequestNotFound):
		return grpcerrors.NotFound(err, "factory pull request not found")
	case errors.Is(err, models.ErrFactoryPullRequestInvalid):
		return grpcerrors.InvalidArgument(err, err.Error())
	case errors.Is(err, models.ErrFactoryPullRequestAlreadyExists):
		return grpcerrors.AlreadyExists(err, "factory pull request already exists")
	case errors.Is(err, models.ErrFactoryPullRequestRunAlreadyLinked):
		return grpcerrors.FailedPrecondition(err, "run is already linked to a different pull request")
	case errors.Is(err, models.ErrFactoryPullRequestLookupIncomplete):
		return grpcerrors.InvalidArgument(err, "pull request lookup is incomplete")
	case errors.Is(err, errFactoryGitHubNotConnected):
		return grpcerrors.FailedPrecondition(err, "GitHub is not connected.")
	case errors.Is(err, errCannotCloseBitbucketPullRequest):
		return grpcerrors.FailedPrecondition(err, "SuperPlane cannot close a Bitbucket pull request.")
	case errors.Is(err, errCannotClosePullRequest):
		return grpcerrors.FailedPrecondition(err, joinedErrorMessage(err, "SuperPlane could not close a previous pull request."))
	case errors.Is(err, errWorkOrderNotClosedForBacklog):
		return grpcerrors.FailedPrecondition(err, "Only a closed task can move to the Backlog.")
	case errors.Is(err, errFactoryPullRequestNotGitHub):
		return grpcerrors.FailedPrecondition(err, "Only GitHub pull requests can merge from SuperPlane.")
	case errors.Is(err, errFactoryPullRequestNotOpen):
		return grpcerrors.FailedPrecondition(err, "The pull request is not open.")
	case errors.Is(err, errFactoryPullRequestNotMergeable):
		return grpcerrors.FailedPrecondition(err, joinedErrorMessage(err, "The pull request cannot merge."))
	case errors.Is(err, errFactoryPullRequestMergeMethodNotAllowed):
		return grpcerrors.FailedPrecondition(err, "The repository does not allow this merge method.")
	case errors.Is(err, errFactoryPullRequestHeadMoved):
		return grpcerrors.FailedPrecondition(err, "The pull request head changed. Review the pull request and try again.")
	case errors.Is(err, models.ErrFactoryPlanningSessionNotFound):
		return grpcerrors.NotFound(err, "planning session not found")
	case errors.Is(err, models.ErrFactoryPlanningSessionInvalid):
		return grpcerrors.InvalidArgument(err, err.Error())
	case errors.Is(err, models.ErrFactoryPlanningSessionEnded):
		return grpcerrors.FailedPrecondition(err, "planning session has ended")
	case errors.Is(err, models.ErrFactoryPlanningSessionNoDraft):
		return grpcerrors.FailedPrecondition(err, "planning session has no draft")
	case errors.Is(err, models.ErrFactoryAgentResourceNotFound):
		return grpcerrors.NotFound(err, "agent resource not found")
	case errors.Is(err, models.ErrFactoryAgentResourceKindInvalid):
		return grpcerrors.InvalidArgument(err, "agent resource kind is not valid")
	case errors.Is(err, models.ErrFactoryAgentResourceNameInvalid):
		return grpcerrors.InvalidArgument(err, "name must be lowercase letters, digits, and dashes")
	case errors.Is(err, models.ErrFactoryAgentResourceNameReserved):
		return grpcerrors.InvalidArgument(err, "the name superplane is reserved")
	case errors.Is(err, models.ErrFactoryAgentResourceNameTaken):
		return grpcerrors.AlreadyExists(err, "an agent resource with this name already exists")
	case errors.Is(err, models.ErrFactoryAgentResourceAuthInvalid):
		return grpcerrors.InvalidArgument(err, "auth must be headers or oauth")
	case errors.Is(err, models.ErrFactoryAgentResourceURLRequired):
		return grpcerrors.InvalidArgument(err, "MCP URL is required")
	case errors.Is(err, models.ErrFactoryAgentResourceHeaderInvalid):
		return grpcerrors.InvalidArgument(err, "each header needs a name, secret, and key")
	case errors.Is(err, models.ErrFactoryAgentResourceKindNotSupported):
		return grpcerrors.FailedPrecondition(err, "GitHub skill packages are not available yet")
	case errors.Is(err, models.ErrFactoryAgentResourceMarkdownRequired):
		return grpcerrors.InvalidArgument(err, "SKILL.md content is required")
	case errors.Is(err, models.ErrFactoryAgentResourceMarkdownTooLarge):
		return grpcerrors.InvalidArgument(err, "SKILL.md must be 64 KiB or smaller")
	case errors.Is(err, models.ErrFactoryAgentResourceMCPCapReached):
		return grpcerrors.FailedPrecondition(err, "this workspace already has 20 enabled MCP connections")
	case errors.Is(err, errFactoryAgentResourceNotConnected):
		return grpcerrors.FailedPrecondition(err, "Connect this MCP server first.")
	case errors.Is(err, errListMCPTools):
		return grpcerrors.FailedPrecondition(err, "SuperPlane could not load the tools. Try again.")
	case errors.Is(err, models.ErrSelectableLLMModelIncomplete):
		return grpcerrors.InvalidArgument(err, "Select a model from the list.")
	case errors.Is(err, models.ErrSelectableLLMModelNotAllowed):
		return grpcerrors.FailedPrecondition(err, "This workspace does not allow the selected model.")
	case errors.Is(err, errIntakeNotConnected):
		return grpcerrors.FailedPrecondition(err, "Connect this intake first.")
	case errors.Is(err, errIntakeSearchUnsupported):
		return grpcerrors.FailedPrecondition(err, "This intake cannot search items yet.")
	case errors.Is(err, errIntakeRefreshUnsupported):
		return grpcerrors.FailedPrecondition(err, "Add a readable intake before you refresh the backlog.")
	case errors.Is(err, errIntakeItemNotFound):
		return grpcerrors.NotFound(err, "intake item not found")
	case errors.Is(err, models.ErrFileNotFound):
		return grpcerrors.NotFound(err, "file not found")
	case errors.Is(err, models.ErrFileNotReady), errors.Is(err, models.ErrFileForeignReference), errors.Is(err, models.ErrFileInvalid), errors.Is(err, models.ErrFileContentType):
		return grpcerrors.InvalidArgument(err, err.Error())
	case errors.Is(err, models.ErrFileQuotaExceeded):
		return grpcerrors.FailedPrecondition(err, err.Error())
	case errors.Is(err, errCustomAutomationsDisabled):
		return grpcerrors.FailedPrecondition(err, "Custom automations are not enabled for this organization.")
	case errors.Is(err, errFactoryAutomationReserved):
		return grpcerrors.FailedPrecondition(err, "This canvas belongs to a factory intake, line, backlog, or PR feedback handler.")
	case errors.Is(err, errFactoryPullRequestMergeDisabled):
		return grpcerrors.FailedPrecondition(err, "Pull request merge is not enabled for this organization.")
	case errors.Is(err, errWorkspaceMCPDisabled):
		return grpcerrors.FailedPrecondition(err, "Workspace MCP is not enabled for this organization.")
	case errors.Is(err, errWorkspaceSkillsDisabled):
		return grpcerrors.FailedPrecondition(err, "Workspace skills are not enabled for this organization.")
	case errors.Is(err, errInvalidArgument):
		return grpcerrors.InvalidArgument(err, err.Error())
	case errors.Is(err, gorm.ErrRecordNotFound):
		return grpcerrors.NotFound(err, "resource not found")
	default:
		return grpcerrors.Internal(err, internalMessage)
	}
}

var errInvalidArgument = errors.New("invalid argument")
var errCustomAutomationsDisabled = errors.New("custom automations are not enabled")
var errFactoryAutomationReserved = errors.New("factory automation is reserved")
var errFactoryAgentResourceNotConnected = errors.New("connect this MCP server first")
var errListMCPTools = errors.New("could not list MCP tools")
var errFactoryPullRequestMergeDisabled = errors.New("pull request merge is not enabled")
var errWorkspaceMCPDisabled = errors.New("workspace MCP is not enabled")
var errWorkspaceSkillsDisabled = errors.New("workspace skills are not enabled")
var errCannotCloseBitbucketPullRequest = errors.New("cannot close a bitbucket pull request")
var errCannotClosePullRequest = errors.New("could not close a previous pull request")
var errWorkOrderNotClosedForBacklog = errors.New("only a closed task can move to the backlog")

func invalidArgument(message string) error {
	return errors.Join(errInvalidArgument, errors.New(message))
}

func joinedErrorMessage(err error, fallback string) string {
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		return fallback
	}
	inner := joined.Unwrap()
	if len(inner) == 0 {
		return fallback
	}
	message := inner[len(inner)-1].Error()
	if message == "" {
		return fallback
	}
	return message
}
