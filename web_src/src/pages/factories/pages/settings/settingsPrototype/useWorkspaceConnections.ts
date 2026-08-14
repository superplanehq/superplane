import { useCallback, useState } from "react";

import { useOnboardingStorybook } from "../../onboarding/useOnboardingStorybook";
import { STORYBOOK_WORKSPACE_CONNECTIONS, type WorkspaceConnections } from "./workspaceConnections";

export function useWorkspaceConnections(workspaceId: string) {
  const onboarding = useOnboardingStorybook();
  const fromContext = onboarding?.connections(workspaceId);
  const [localConnections, setLocalConnections] = useState<WorkspaceConnections>(STORYBOOK_WORKSPACE_CONNECTIONS);
  const connections = fromContext ?? localConnections;

  const updateConnections = useCallback(
    (next: WorkspaceConnections) => {
      if (onboarding) {
        onboarding.updateConnections(workspaceId, next);
        return;
      }
      setLocalConnections(next);
    },
    [onboarding, workspaceId],
  );

  return { connections, updateConnections };
}
