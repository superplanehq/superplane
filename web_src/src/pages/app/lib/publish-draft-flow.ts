import type { QueryClient } from "@tanstack/react-query";
import type { MutableRefObject } from "react";

import { ensureDraftVersionExists } from "@/hooks/useCanvasData";

import type { RefreshLatestLiveCanvasDataOptions } from "../useRefreshLatestLiveCanvasData";

type PublishMutation = {
  mutateAsync: (input: {
    versionId: string;
    commitMessage?: string;
  }) => Promise<{ data?: { version?: { metadata?: { id?: string } } } } | undefined>;
};

export type PublishDraftVersionAndExitResult =
  | { status: "not-ready" }
  | { status: "missing"; versionIdToPublish: string }
  | { status: "published"; versionIdToPublish: string; publishedVersionId?: string }
  | { status: "failed"; versionIdToPublish: string; error: unknown };

export async function executePublishDraftVersion({
  organizationId,
  canvasId,
  versionIdToPublish,
  commitMessage,
  queryClient,
  publishCanvasVersionMutation,
  registerIgnoredCanvasUpdatedEcho,
  registerIgnoredCanvasVersionUpdatedEcho,
}: {
  organizationId: string;
  canvasId: string;
  versionIdToPublish: string;
  commitMessage?: string;
  queryClient: QueryClient;
  publishCanvasVersionMutation: PublishMutation;
  registerIgnoredCanvasUpdatedEcho?: () => () => void;
  registerIgnoredCanvasVersionUpdatedEcho?: (versionId?: string) => () => void;
}): Promise<"missing" | { status: "published"; publishedVersionId?: string }> {
  const draftExists = await ensureDraftVersionExists(queryClient, organizationId, canvasId, versionIdToPublish);
  if (!draftExists) {
    return "missing";
  }

  const releaseCanvasUpdatedEcho = registerIgnoredCanvasUpdatedEcho?.();
  const releaseCanvasVersionUpdatedEcho = registerIgnoredCanvasVersionUpdatedEcho?.(versionIdToPublish);
  let publishResponse: Awaited<ReturnType<PublishMutation["mutateAsync"]>>;
  try {
    publishResponse = await publishCanvasVersionMutation.mutateAsync({ versionId: versionIdToPublish, commitMessage });
  } catch (error) {
    releaseCanvasUpdatedEcho?.();
    releaseCanvasVersionUpdatedEcho?.();
    throw error;
  }

  return { status: "published", publishedVersionId: publishResponse?.data?.version?.metadata?.id };
}

export async function publishDraftVersionAndExit({
  organizationId,
  canvasId,
  activeCanvasVersionIdRef,
  commitMessage,
  queryClient,
  ensureVersionActionDraftReady,
  publishCanvasVersionMutation,
  registerIgnoredCanvasUpdatedEcho,
  registerIgnoredCanvasVersionUpdatedEcho,
  runExitDraftToLive,
  recoverFromMissingDraft,
}: {
  organizationId: string;
  canvasId: string;
  activeCanvasVersionIdRef: MutableRefObject<string>;
  commitMessage?: string;
  queryClient: QueryClient;
  ensureVersionActionDraftReady: (errorMessage: string) => Promise<boolean>;
  publishCanvasVersionMutation: PublishMutation;
  registerIgnoredCanvasUpdatedEcho?: () => () => void;
  registerIgnoredCanvasVersionUpdatedEcho?: (versionId?: string) => () => void;
  runExitDraftToLive: (versionId: string, options?: RefreshLatestLiveCanvasDataOptions) => Promise<void>;
  recoverFromMissingDraft: (versionId: string) => Promise<void>;
}): Promise<PublishDraftVersionAndExitResult> {
  const isReady = await ensureVersionActionDraftReady("Unable to prepare the latest version changes for publishing");
  if (!isReady) {
    return { status: "not-ready" };
  }

  const versionIdToPublish = activeCanvasVersionIdRef.current;
  if (!versionIdToPublish) {
    return { status: "not-ready" };
  }

  try {
    const publishResult = await executePublishDraftVersion({
      organizationId,
      canvasId,
      versionIdToPublish,
      commitMessage,
      queryClient,
      publishCanvasVersionMutation,
      registerIgnoredCanvasUpdatedEcho,
      registerIgnoredCanvasVersionUpdatedEcho,
    });
    if (publishResult === "missing") {
      await recoverFromMissingDraft(versionIdToPublish);
      return { status: "missing", versionIdToPublish };
    }

    const liveVersionId = publishResult.publishedVersionId ?? versionIdToPublish;
    await runExitDraftToLive(versionIdToPublish, {
      liveVersionId,
      skipDraftBranchRefetch: true,
    });
    return { status: "published", versionIdToPublish, publishedVersionId: publishResult.publishedVersionId };
  } catch (error) {
    return { status: "failed", versionIdToPublish, error };
  }
}
