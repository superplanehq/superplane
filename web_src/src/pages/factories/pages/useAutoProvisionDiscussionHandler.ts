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
  const [failed, setFailed] = useState(false);
  const listHandlersRef = useRef(args.listHandlers);
  const createHandlerRef = useRef(args.createHandler);
  listHandlersRef.current = args.listHandlers;
  createHandlerRef.current = args.createHandler;

  useEffect(() => {
    if (!args.enabled || !args.factoryId || failed) {
      return;
    }

    let active = true;
    const existing = discussionProvisionInFlight.get(args.factoryId);
    const attempt =
      existing ??
      provisionDiscussionHandler({
        listHandlers: () => listHandlersRef.current(),
        createHandler: (input) => Promise.resolve(createHandlerRef.current(input)),
      });
    if (!existing) {
      discussionProvisionInFlight.set(args.factoryId, attempt);
    }

    void attempt.then(
      () => {
        if (discussionProvisionInFlight.get(args.factoryId) === attempt) {
          discussionProvisionInFlight.delete(args.factoryId);
        }
      },
      () => {
        if (discussionProvisionInFlight.get(args.factoryId) === attempt) {
          discussionProvisionInFlight.delete(args.factoryId);
        }
        if (active) {
          setFailed(true);
        }
      },
    );

    return () => {
      active = false;
    };
  }, [args.enabled, args.factoryId, failed]);

  return { failed };
}
