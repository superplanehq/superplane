import type { FactoriesFactoryOnboarding, SuperplaneComponentsNode } from "@/api-client";
import { useFactory } from "@/hooks/useFactoryData";
import { useIntegrationResources } from "@/hooks/useIntegrations";

import { HumanReviewPolicySettings } from "./HumanReviewPolicySettings";
import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";

export function HumanReviewPolicySettingsHost({
  node,
  organizationId = "",
  factoryId = "",
  onSave,
}: {
  node: SuperplaneComponentsNode;
  organizationId?: string;
  factoryId?: string;
  onSave: (update: NodeConfigurationUpdate) => Promise<void> | void;
}) {
  const factory = useFactory(organizationId, factoryId);
  const { repository, integrationId, hasGitHubRepository } = reviewerRepositoryBinding(factory.data?.onboarding);
  const reviewers = useIntegrationResources(
    organizationId,
    integrationId,
    "reviewer",
    { repository },
    {
      enabled: hasGitHubRepository,
    },
  );
  const reviewersLoading = factory.isLoading || reviewers.isLoading;
  const reviewersError = reviewerLoadError({
    workspaceFailed: factory.isError,
    repositoryMissing: !reviewersLoading && !hasGitHubRepository,
    reviewersFailed: reviewers.isError,
  });
  const retryReviewers = () => {
    if (factory.isError) {
      void factory.refetch();
      return;
    }
    void reviewers.refetch();
  };

  return (
    <HumanReviewPolicySettings
      node={node}
      reviewers={(reviewers.data ?? []).map((reviewer) => reviewer.id?.trim() ?? "").filter(Boolean)}
      reviewersLoading={reviewersLoading}
      reviewersError={reviewersError}
      onRetryReviewers={factory.isError || hasGitHubRepository ? retryReviewers : undefined}
      onSave={onSave}
    />
  );
}

function reviewerRepositoryBinding(onboarding: FactoriesFactoryOnboarding | undefined) {
  const repository = onboarding?.appRepository?.trim() ?? "";
  const integrationId = onboarding?.vcsIntegrationId ?? "";
  return {
    repository,
    integrationId,
    hasGitHubRepository: Boolean(repository && integrationId && onboarding?.vcsProvider !== "bitbucket"),
  };
}

function reviewerLoadError({
  workspaceFailed,
  repositoryMissing,
  reviewersFailed,
}: {
  workspaceFailed: boolean;
  repositoryMissing: boolean;
  reviewersFailed: boolean;
}) {
  if (workspaceFailed) {
    return "SuperPlane could not load the workspace repository. Try again.";
  }
  if (repositoryMissing) {
    return "Connect a GitHub repository in workspace settings to select a default reviewer.";
  }
  if (reviewersFailed) {
    return "SuperPlane could not load reviewers. Check GitHub App access and try again.";
  }
  return undefined;
}
