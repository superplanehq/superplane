import { useCallback } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { usePermissions } from "@/contexts/usePermissions";
import { useCreateCanvas, useUpdateCanvasFolderMembership } from "@/hooks/useCanvasData";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { showErrorToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { PLACEHOLDER_NODE_CONTEXT_KEY, setAgentBootContext } from "@/lib/agentBootContext";
import { writeCanvasAgentSidebarOpen } from "@/components/CanvasToolSidebar/useCanvasToolSidebarState";
import { writeCanvasRunsSidebarOpen } from "@/components/CanvasRunsSidebar/useCanvasRunsSidebarState";
import { appPath } from "@/lib/appPaths";
import { appendCanvasToFolderMembership } from "./canvasFolderMembership";
import type { IntegrationSelections } from "./InstallIntegrationsSection";
import type { CanvasFolderData } from "./types";

interface UseCreateAppOptions {
  folder?: CanvasFolderData;
  onCreated?: () => void;
}

export interface CreateAppOptions {
  factorySetup?: {
    repository: string;
    integrations: IntegrationSelections;
  };
}

function buildFactoryBootMessage(repository: string, integrations: IntegrationSelections): string {
  const github = integrations.github;
  const claude = integrations.claude;
  const parts = [
    `Set up a Software Factory for the GitHub repository "${repository}".`,
    github ? `Use the existing GitHub integration "${github.name}" (id: ${github.id}).` : null,
    claude ? `Use the existing Claude integration "${claude.name}" (id: ${claude.id}).` : null,
    "Automate delivery from trigger to pull request.",
  ];
  return parts.filter((part): part is string => Boolean(part)).join(" ");
}

export function useCreateApp({ folder, onCreated }: UseCreateAppOptions = {}) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const { canAct } = usePermissions();
  const createCanvasMutation = useCreateCanvas(organizationId || "");
  const updateCanvasFolderMembershipMutation = useUpdateCanvasFolderMembership(organizationId || "");
  const { mutateAsync: createCanvas } = createCanvasMutation;
  const { mutateAsync: updateCanvasFolderMembership } = updateCanvasFolderMembershipMutation;

  const canCreateCanvases = canAct("canvases", "create");
  const canUpdateCanvases = canAct("canvases", "update");
  const isSaving = createCanvasMutation.isPending || updateCanvasFolderMembershipMutation.isPending;

  const createApp = useCallback(
    async (name: string, options?: CreateAppOptions) => {
      if (!organizationId || !canCreateCanvases || isSaving) {
        return;
      }

      if (folder && !canUpdateCanvases) {
        showErrorToast("You don't have permission to update canvases.");
        return;
      }

      try {
        const result = await createCanvas({
          name,
          method: "ui",
        });

        const canvasId = result?.data?.canvas?.metadata?.id;
        if (canvasId) {
          if (folder) {
            try {
              await updateCanvasFolderMembership(appendCanvasToFolderMembership(folder, canvasId));
            } catch (error) {
              showErrorToast(getApiErrorMessage(error, "App created, but failed to add it to folder"));
            }
          }

          onCreated?.();
          // A new app always starts with the agent panel open (stored per canvas).
          writeCanvasAgentSidebarOpen(canvasId, true);
          writeCanvasRunsSidebarOpen(canvasId, false);
          localStorage.setItem("canvasSidebarOpen", "false");
          if (options?.factorySetup) {
            setAgentBootContext(
              canvasId,
              buildFactoryBootMessage(options.factorySetup.repository, options.factorySetup.integrations),
            );
          } else {
            setAgentBootContext(canvasId, "blank");
            sessionStorage.setItem(PLACEHOLDER_NODE_CONTEXT_KEY, canvasId);
          }
          navigate(appPath(organizationId, canvasId, "?edit=1"));
        }
      } catch (error) {
        showErrorToast(getUsageLimitToastMessage(error, "Failed to create app"));
        throw error;
      }
    },
    [
      canCreateCanvases,
      canUpdateCanvases,
      createCanvas,
      folder,
      isSaving,
      navigate,
      onCreated,
      organizationId,
      updateCanvasFolderMembership,
    ],
  );

  return {
    createApp,
    isSaving,
  };
}
