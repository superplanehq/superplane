import { useCallback, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { generateCanvasName } from "@/lib/canvasNameGenerator";

interface InstallResult {
  canvasId: string;
  organizationId: string;
}

export function useInstallTemplate() {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const [isInstalling, setIsInstalling] = useState(false);

  const installTemplate = useCallback(
    async (repo: string, appMeta?: { title: string; description: string }) => {
      if (!organizationId || isInstalling) return;

      setIsInstalling(true);
      try {
        const response = await fetch("/apps/install", {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            repo,
            name: generateCanvasName(),
            organizationId,
          }),
        });

        if (!response.ok) {
          const message = await response.text();
          throw new Error(message || "Failed to install template");
        }

        const result = (await response.json()) as InstallResult;
        localStorage.setItem("canvasAgentSidebarOpen", "true");
        if (appMeta) sessionStorage.setItem("agent-boot-context", JSON.stringify(appMeta));
        navigate(`/${result.organizationId}/canvases/${result.canvasId}?edit=1`);
      } catch (error) {
        const message = getUsageLimitToastMessage(error, getApiErrorMessage(error, "Failed to install template"));
        showErrorToast(message);
      } finally {
        setIsInstalling(false);
      }
    },
    [isInstalling, navigate, organizationId],
  );

  return { installTemplate, isInstalling };
}
