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

  useEffect(() => {
    if (!shouldCreate) {
      launchedRef.current = false;
    }
  }, [shouldCreate]);

  useEffect(() => {
    if (!shouldCreate || !scanFinished || launchedRef.current) {
      return;
    }
    launchedRef.current = true;
    setCreating(true);

    const catalog = catalogQuery.isError ? [] : catalogQuery.data ?? [];
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
        setCreating(false);
      });
  }, [shouldCreate, scanFinished, catalogQuery.isError, catalogQuery.data, repository, createHandler, refetchHandlers]);

  return {
    pending: creating || (shouldCreate && needsScan && !scanFinished),
  };
}