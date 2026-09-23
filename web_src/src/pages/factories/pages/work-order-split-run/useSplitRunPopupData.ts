import { useWorkOrder, useWorkOrderArtifacts } from "@/hooks/useFactoryData";

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
  const liveWorkOrderQuery = useWorkOrder(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const artifacts = resolveSplitRunPopupArtifacts({
    fixtureArtifacts,
    liveArtifacts: liveArtifactsQuery.data,
    useLive,
  });
  const pullRequests = resolveSplitRunPopupPullRequests({
    fixturePullRequests,
    livePullRequests: liveWorkOrderQuery.data?.pullRequests,
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
    artifactsError: useLive ? (liveArtifactsQuery.error ?? null) : null,
    pullRequestsLoading: useLive && liveWorkOrderQuery.isLoading,
    pullRequestsError: useLive ? (liveWorkOrderQuery.error ?? null) : null,
  };
}
