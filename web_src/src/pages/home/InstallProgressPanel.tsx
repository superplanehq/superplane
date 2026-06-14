import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ExternalLink, Loader2 } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { canvasKeys } from "@/hooks/useCanvasData";
import { IntegrationResourceFieldRenderer } from "@/ui/configurationFieldRenderer/IntegrationResourceFieldRenderer";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { setAgentBootContext } from "@/lib/agentBootContext";
import { showErrorToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { appPath } from "@/lib/appPaths";
import type { AppEntry } from "./AppDetailModal";
import { IntegrationIcons } from "./AppDetailModal";
import { IntegrationsSection, type IntegrationSelections } from "./InstallIntegrationsSection";
import { useInstallPreviewData } from "./useInstallPreviewData";
import type { InstallParam } from "../install/types";

function checkCanProceed(
  organizationId: string | undefined,
  isInstalling: boolean,
  preview: { previewLoading: boolean; previewError: string | null },
  integrations: string[],
  selections: IntegrationSelections,
): boolean {
  if (!organizationId || isInstalling || preview.previewLoading || preview.previewError) return false;
  return integrations.length === 0 || integrations.every((name) => selections[name]);
}

function checkRequiredParams(params: InstallParam[], values: Record<string, string>): boolean {
  return params.filter((p) => p.required && !p.default).every((p) => (values[p.name] ?? "").trim() !== "");
}

async function executeInstall(opts: {
  repo: string;
  organizationId: string;
  installParams?: Record<string, string>;
  integrations: IntegrationSelections;
}): Promise<{ canvasId: string; organizationId: string }> {
  const body: Record<string, unknown> = {
    repo: opts.repo,
    name: generateCanvasName(),
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

interface InstallProgressPanelProps {
  app: AppEntry;
  organizationId?: string;
  preloadedIntegrations?: string[];
  preloadedParams?: InstallParam[];
  onClose: () => void;
}

export function InstallProgressPanel({
  app,
  organizationId: propOrgId,
  preloadedIntegrations,
  preloadedParams,
  onClose,
}: InstallProgressPanelProps) {
  const { organizationId: routeOrgId } = useParams<{ organizationId: string }>();
  const organizationId = propOrgId || routeOrgId;
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const preview = useInstallPreviewData({
    repo: app.repo,
    preloadedIntegrations,
    preloadedParams,
  });

  const [integrationSelections, setIntegrationSelections] = useState<IntegrationSelections>({});
  const [isInstalling, setIsInstalling] = useState(false);

  // Clear selections when org changes
  const resetParamValuesRef = useRef(preview.resetParamValues);
  resetParamValuesRef.current = preview.resetParamValues;
  useEffect(() => {
    setIntegrationSelections({});
    resetParamValuesRef.current();
  }, [organizationId]);

  const doInstall = useCallback(
    async (skipParams: boolean) => {
      if (!organizationId || isInstalling) return;
      setIsInstalling(true);
      try {
        const result = await executeInstall({
          repo: app.repo,
          organizationId,
          installParams: !skipParams && preview.installParams.length > 0 ? preview.paramValues : undefined,
          integrations: integrationSelections,
        });
        await queryClient.refetchQueries({ queryKey: canvasKeys.list(result.organizationId) });
        if (app.agentInstructions || app.agentInitialMessage) {
          setAgentBootContext(result.canvasId, {
            instructions: app.agentInstructions,
            initialMessage: app.agentInitialMessage,
          });
        }
        localStorage.setItem("canvasAgentSidebarOpen", "true");
        localStorage.setItem("canvasSidebarOpen", "false");
        navigate(appPath(result.organizationId, result.canvasId, "?edit=1"));
      } catch (error) {
        setIsInstalling(false);
        showErrorToast(getUsageLimitToastMessage(error, getApiErrorMessage(error, "Failed to install")));
      }
    },
    [
      organizationId,
      app,
      preview.paramValues,
      preview.installParams,
      integrationSelections,
      isInstalling,
      queryClient,
      navigate,
    ],
  );

  const integrations = preview.detectedIntegrations.length > 0 ? preview.detectedIntegrations : app.integrations;
  const hasIntegrations = integrations.length > 0;
  const hasParams = preview.installParams.length > 0;
  const canProceed = checkCanProceed(organizationId, isInstalling, preview, integrations, integrationSelections);
  const canInstall = canProceed && checkRequiredParams(preview.installParams, preview.paramValues);
  // Skip only needs org + not installing + preview loaded — skips both integrations and params
  const canSkip = !!organizationId && !isInstalling && !preview.previewLoading && !preview.previewError;

  return (
    <div className="mt-4 rounded-lg bg-white p-5 outline outline-slate-950/10 animate-in slide-in-from-top-2 dark:bg-gray-900">
      <AppInfoHeader app={app} integrations={integrations} />

      {hasIntegrations && (
        <div className="mb-5 border-t border-slate-100 pt-4">
          <IntegrationsSection
            integrations={integrations}
            organizationId={organizationId ?? ""}
            selections={integrationSelections}
            onSelectionsChange={setIntegrationSelections}
          />
        </div>
      )}

      {preview.previewLoading && (
        <div className="mb-5 border-t border-slate-100 pt-4">
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Loading configuration...
          </div>
        </div>
      )}

      {preview.previewError && (
        <div className="mb-5 border-t border-slate-100 pt-4">
          <p className="text-xs text-red-600">{preview.previewError}</p>
        </div>
      )}

      {!preview.previewLoading && hasParams && (
        <div className="mb-5 border-t border-slate-100 pt-4">
          <ParamsSection
            params={preview.installParams}
            values={preview.paramValues}
            onChange={preview.setParamValues}
            organizationId={organizationId}
            integrationSelections={integrationSelections}
          />
        </div>
      )}

      {!preview.previewLoading && (
        <InstallActions
          canInstall={canInstall}
          canSkip={canSkip}
          isInstalling={isInstalling}
          onInstall={() => void doInstall(false)}
          onSkip={() => void doInstall(true)}
          onClose={onClose}
        />
      )}
    </div>
  );
}

function InstallActions({
  canInstall,
  canSkip,
  isInstalling,
  onInstall,
  onSkip,
  onClose,
}: {
  canInstall: boolean;
  canSkip: boolean;
  isInstalling: boolean;
  onInstall: () => void;
  onSkip: () => void;
  onClose: () => void;
}) {
  return (
    <div className="flex items-center gap-2 border-t border-slate-100 pt-4">
      <Button size="sm" onClick={onInstall} disabled={!canInstall}>
        {isInstalling ? (
          <>
            <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" />
            Installing...
          </>
        ) : (
          "Install"
        )}
      </Button>
      <Button variant="outline" size="sm" onClick={onSkip} disabled={!canSkip}>
        Just take me there
      </Button>
      <Button variant="ghost" size="sm" onClick={onClose} disabled={isInstalling}>
        Cancel
      </Button>
    </div>
  );
}

function AppInfoHeader({ app, integrations }: { app: AppEntry; integrations: string[] }) {
  return (
    <div className="mb-5">
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <h3 className="text-sm font-semibold text-slate-900">{app.title}</h3>
          <div className="mt-1 flex flex-wrap items-center gap-2">
            <IntegrationIcons integrations={integrations} />
            {app.tags.map((tag) => (
              <span key={tag} className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-medium text-slate-500">
                {tag}
              </span>
            ))}
          </div>
        </div>
        <a
          href={`https://${app.repo}`}
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-1 text-[10px] font-medium text-slate-400 hover:text-slate-600 shrink-0"
        >
          <ExternalLink className="h-3 w-3" />
          GitHub
        </a>
      </div>
      {app.description && <p className="mt-2 text-xs leading-relaxed text-slate-600">{app.description}</p>}
      {app.requirements.length > 0 && (
        <ul className="mt-2 space-y-0.5">
          {app.requirements.map((req) => (
            <li key={req} className="flex items-start gap-1.5 text-xs text-slate-500">
              <span className="mt-1.5 h-1 w-1 shrink-0 rounded-full bg-slate-300" />
              {req}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function ParamsSection({
  params,
  values,
  onChange,
  organizationId,
  integrationSelections,
}: {
  params: InstallParam[];
  values: Record<string, string>;
  onChange: React.Dispatch<React.SetStateAction<Record<string, string>>>;
  organizationId?: string;
  integrationSelections: IntegrationSelections;
}) {
  return (
    <>
      <p className="text-xs font-semibold text-slate-700 mb-3">Configuration</p>
      <div className="space-y-3">
        {params.map((param) => (
          <div key={param.name} className="space-y-1">
            <Label htmlFor={`param-${param.name}`} className="text-xs">
              {param.label}
              {param.required && <span className="text-red-500 ml-0.5">*</span>}
            </Label>
            {param.type === "integration-resource" && param.integration && param.resourceType ? (
              <IntegrationResourceFieldRenderer
                field={{
                  name: param.name,
                  label: param.label,
                  type: "integration-resource",
                  placeholder: param.placeholder,
                  required: param.required,
                  typeOptions: { resource: { type: param.resourceType } },
                }}
                value={values[param.name]}
                onChange={(val) =>
                  onChange((prev) => ({
                    ...prev,
                    [param.name]: typeof val === "string" ? val : Array.isArray(val) ? (val[0] ?? "") : "",
                  }))
                }
                organizationId={organizationId}
                integrationId={integrationSelections[param.integration]?.id}
              />
            ) : (
              <Input
                id={`param-${param.name}`}
                value={values[param.name] ?? ""}
                placeholder={param.placeholder}
                className="h-8 text-xs"
                onChange={(e) => onChange((prev) => ({ ...prev, [param.name]: e.target.value }))}
              />
            )}
            {param.description && <p className="text-[10px] text-slate-400">{param.description}</p>}
          </div>
        ))}
      </div>
    </>
  );
}
