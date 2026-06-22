import { useCallback, useRef, useState } from "react";
import type { NavigateFunction } from "react-router-dom";
import type { QueryClient } from "@tanstack/react-query";
import { canvasKeys } from "@/hooks/useCanvasData";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { setAgentBootContext } from "@/lib/agentBootContext";
import { showErrorToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { appPath } from "@/lib/appPaths";
import type { AppEntry } from "./AppDetailModal";
import type { IntegrationSelections } from "./InstallIntegrationsSection";
import type { InstallParam } from "../install/types";

async function executeInstall(opts: {
  repo: string;
  organizationId: string;
  name?: string;
  installParams?: Record<string, string>;
  integrations: IntegrationSelections;
}): Promise<{ canvasId: string; organizationId: string }> {
  const body: Record<string, unknown> = {
    repo: opts.repo,
    name: opts.name || generateCanvasName(),
    organizationId: opts.organizationId,
  };
  if (opts.installParams) body.installParams = opts.installParams;
  if (Object.keys(opts.integrations).length > 0) body.integrations = opts.integrations;
  const response = await fetch("/apps/install", {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) throw new Error((await response.text()) || "Failed to install");
  return response.json() as Promise<{ canvasId: string; organizationId: string }>;
}

function prepareAgentSidebar(app: AppEntry, canvasId: string) {
  if (app.agentInstructions || app.agentInitialMessage) {
    setAgentBootContext(canvasId, {
      instructions: app.agentInstructions,
      initialMessage: app.agentInitialMessage,
    });
  }
  localStorage.setItem("canvasAgentSidebarOpen", "true");
  localStorage.setItem("canvasSidebarOpen", "false");
}

interface UseInstallActionOptions {
  organizationId: string | undefined;
  app: AppEntry;
  canvasName?: string;
  installParams: InstallParam[];
  paramValues: Record<string, string>;
  integrationSelections: IntegrationSelections;
  queryClient: QueryClient;
  navigate: NavigateFunction;
}

export function useInstallAction({
  organizationId,
  app,
  canvasName,
  installParams,
  paramValues,
  integrationSelections,
  queryClient,
  navigate,
}: UseInstallActionOptions) {
  const [isInstalling, setIsInstalling] = useState(false);
  const isInstallingRef = useRef(false);

  const doInstall = useCallback(
    async (skipParams: boolean) => {
      if (!organizationId || isInstallingRef.current) return;
      isInstallingRef.current = true;
      setIsInstalling(true);
      try {
        const result = await executeInstall({
          repo: app.repo,
          organizationId,
          name: canvasName,
          installParams: !skipParams && installParams.length > 0 ? paramValues : undefined,
          integrations: integrationSelections,
        });
        await queryClient.refetchQueries({ queryKey: canvasKeys.list(result.organizationId) });
        prepareAgentSidebar(app, result.canvasId);
        navigate(appPath(result.organizationId, result.canvasId, "?edit=1"));
      } catch (error) {
        isInstallingRef.current = false;
        setIsInstalling(false);
        showErrorToast(getUsageLimitToastMessage(error, getApiErrorMessage(error, "Failed to install")));
      }
    },
    [organizationId, app, canvasName, paramValues, installParams, integrationSelections, queryClient, navigate],
  );

  return { doInstall, isInstalling };
}
