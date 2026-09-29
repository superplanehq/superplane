import { useEffect, useRef, useState } from "react";

import {
  provisionDiscussionHandler,
  type CreateDiscussionHandler,
  type ListFactoryPRFeedbackHandlers,
} from "./onboarding/onboardingProvision";

const discussionProvisionInFlight = new Map<string, Promise<void>>();

export function resetDiscussionHandlerProvisionAttempts() {
  discussionProvisionInFlight.clear();
}

export function useAutoProvisionDiscussionHandler(args: {
  factoryId: string;
  enabled: boolean;
  listHandlers: ListFactoryPRFeedbackHandlers;
  createHandler: CreateDiscussionHandler;
}): { failed: boolean } {
  const [failedFactoryId, setFailedFactoryId] = useState<string | null>(null);
  const failed = Boolean(args.factoryId) && failedFactoryId === args.factoryId;
  const listHandlersRef = useRef(args.listHandlers);
  const createHandlerRef = useRef(args.createHandler);
  listHandlersRef.current = args.listHandlers;
  createHandlerRef.current = args.createHandler;

  useEffect(() => {
    if (!args.enabled || !args.factoryId || failed) {
      return;
    }

    const factoryId = args.factoryId;
    const listHandlers = listHandlersRef.current;
    const createHandler = createHandlerRef.current;
    let active = true;
    const existing = discussionProvisionInFlight.get(factoryId);
    const attempt =
      existing ??
      provisionDiscussionHandler({
        listHandlers,
        createHandler: (input) => Promise.resolve(createHandler(input)),
      });
    if (!existing) {
      discussionProvisionInFlight.set(factoryId, attempt);
    }

    void attempt.then(
      () => {
        if (discussionProvisionInFlight.get(factoryId) === attempt) {
          discussionProvisionInFlight.delete(factoryId);
        }
      },
      () => {
        if (discussionProvisionInFlight.get(factoryId) === attempt) {
          discussionProvisionInFlight.delete(factoryId);
        }
        if (active) {
          setFailedFactoryId(factoryId);
        }
      },
    );

    return () => {
      active = false;
    };
  }, [args.enabled, args.factoryId, failed]);

  return { failed };
}
