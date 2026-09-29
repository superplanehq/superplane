import { useCreateFactoryPRFeedbackHandler } from "@/hooks/useFactoryPRFeedbackData";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { useEffect, useRef, useState } from "react";

import { prFeedbackSourceById } from "./prFeedbackSettingsModel";
import { autoConfigureDiscussionSettings } from "./useDiscussionPRFeedbackSetup";

const DISCUSSION_HANDLER_CONFLICT_MESSAGE = "factory already has a pull request discussion handler";

export function isDiscussionHandlerConflictError(error: unknown): boolean {
  return getApiErrorMessage(error, "").includes(DISCUSSION_HANDLER_CONFLICT_MESSAGE);
}

/**
 * Creates the pull request discussion handler on its own once onboarding is
 * complete and no discussion handler exists. It scans recent pull requests for
 * review bots and allows them by default. If the scan fails or finds none, the
 * handler is still created with an empty allowed list. The next-step comments
 * banner stays hidden while this is pending and reappears on a real failure.
 *
 * The hook keys its launch guard to `organizationId:factoryId:repository`, so
 * it starts a new create when the workspace changes while the component stays
 * mounted. Switching away from a workspace immediately clears the pending
 * state for that workspace: a generation counter stops the completion of a
 * stale request from clearing the pending state of a newer request, and it
 * also stops the pending state from getting stuck when the new workspace
 * needs no create of its own.
 */
export function useAutoConfigurePRComments({
  organizationId,
  factoryId,
  githubIntegrationId,
  repository,
  onboardingComplete,
  canConfigure,
  handlersLoaded,
  hasDiscussionHandler,
  refetchHandlers,
}: {
  organizationId: string;
  factoryId: string;
  githubIntegrationId: string;
  repository: string;
  onboardingComplete: boolean;
  canConfigure: boolean;
  handlersLoaded: boolean;
  hasDiscussionHandler: boolean;
  refetchHandlers: () => void;
}) {
  const createHandler = useCreateFactoryPRFeedbackHandler(organizationId, factoryId);
  const shouldCreate = onboardingComplete && canConfigure && handlersLoaded && !hasDiscussionHandler;
  const catalogParameters = repository.trim() ? { repository: repository.trim() } : undefined;
  const catalogQuery = useIntegrationResources(organizationId, githubIntegrationId, "review_bot", catalogParameters, {
    enabled: shouldCreate && Boolean(githubIntegrationId) && Boolean(catalogParameters),
  });

  const needsScan = shouldCreate && Boolean(githubIntegrationId) && Boolean(catalogParameters);
  const scanFinished = !needsScan || catalogQuery.data !== undefined || catalogQuery.isError;
  const [creating, setCreating] = useState(false);
  const launchedRef = useRef(false);
  const launchGenerationRef = useRef(0);
  const identity = `${organizationId}:${factoryId}:${repository.trim()}`;
  const previousIdentityRef = useRef(identity);

  useEffect(() => {
    if (!shouldCreate) {
      launchedRef.current = false;
      launchGenerationRef.current += 1;
      setCreating(false);
    }
  }, [shouldCreate]);

  useEffect(() => {
    if (previousIdentityRef.current !== identity) {
      previousIdentityRef.current = identity;
      launchedRef.current = false;
      launchGenerationRef.current += 1;
      setCreating(false);
    }
  }, [identity]);

  useEffect(() => {
    if (!shouldCreate || !scanFinished || launchedRef.current) {
      return;
    }
    launchedRef.current = true;
    const generation = ++launchGenerationRef.current;
    setCreating(true);

    const catalog = catalogQuery.isError ? [] : (catalogQuery.data ?? []);
    const discussion = autoConfigureDiscussionSettings(catalog);
    createHandler
      .mutateAsync({
        source: "SOURCE_PULL_REQUEST_DISCUSSION",
        name: prFeedbackSourceById("discussion")?.defaultName,
        settings: {
          subject: repository.trim() ? { repository: repository.trim() } : undefined,
          discussion,
        },
      })
      .catch((error) => {
        if (isDiscussionHandlerConflictError(error)) {
          refetchHandlers();
        }
      })
      .finally(() => {
        if (generation === launchGenerationRef.current) {
          setCreating(false);
        }
      });
  }, [
    shouldCreate,
    scanFinished,
    identity,
    repository,
    catalogQuery.isError,
    catalogQuery.data,
    createHandler,
    refetchHandlers,
  ]);

  return {
    pending: creating || (shouldCreate && needsScan && !scanFinished),
  };
}
