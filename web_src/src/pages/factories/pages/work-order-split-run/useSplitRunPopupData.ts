import { useFactoryPullRequests, useWorkOrderArtifacts } from "@/hooks/useFactoryData";

import {
  collectSplitRunArtifacts,
  collectSplitRunPullRequests,
  resolveSplitRunPopupArtifacts,
  resolveSplitRunPopupPullRequests,
  splitRunDescriptionMarkdown,
  splitRunSourceDescription,
} from "./splitRunPopupModel";
import type { SplitRunFixture } from "./splitRunMocks";

function hasLiveWorkOrder(organizationId?: string, factoryId?: string, orderId?: string) {
  return Boolean(organizationId && factoryId && orderId);
}

export function useSplitRunPopupData(args: {
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  fixture: SplitRunFixture;
}) {
  const { organizationId, factoryId, orderId, fixture } = args;
  const fixtureArtifacts = collectSplitRunArtifacts(fixture);
  const fixturePullRequests = collectSplitRunPullRequests(fixture);
  const useLive = hasLiveWorkOrder(organizationId, factoryId, orderId);
  const liveArtifactsQuery = useWorkOrderArtifacts(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const livePullRequestsQuery = useFactoryPullRequests(
    organizationId ?? "",
    factoryId ?? "",
    orderId ? { workOrderIds: [orderId] } : undefined,
  );
  const artifacts = resolveSplitRunPopupArtifacts({
    fixtureArtifacts,
    liveArtifacts: liveArtifactsQuery.data,
    useLive,
  });
  const pullRequests = resolveSplitRunPopupPullRequests({
    fixturePullRequests,
    livePullRequests: livePullRequestsQuery.data,
    useLive,
  });
  const artifactDescription = splitRunDescriptionMarkdown(artifacts) || splitRunDescriptionMarkdown(fixtureArtifacts);
  const sourceDescription = splitRunSourceDescription({
    workOrderDescription: fixture.descriptionText,
    artifactDescription,
    preferWorkOrder: useLive,
  });

  return {
    artifacts,
    pullRequests,
    sourceDescription,
    useLive,
    artifactsLoading: useLive && liveArtifactsQuery.isLoading,
    pullRequestsLoading: useLive && livePullRequestsQuery.isLoading,
    pullRequestsError: useLive ? (livePullRequestsQuery.error ?? null) : null,
  };
}
