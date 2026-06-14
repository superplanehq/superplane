import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ExternalLink, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { canvasKeys } from "@/hooks/useCanvasData";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { ConfigureIntegrationDialog } from "@/ui/ConfigureIntegrationDialog";
import { getIntegrationWebhookUrl } from "@/lib/integrationUtils";
import { getNextIntegrationName } from "@/pages/organization/settings/components/IntegrationSetup/lib";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { setAgentBootContext } from "@/lib/agentBootContext";
import { showErrorToast } from "@/lib/toast";
import { getApiErrorMessage } from "@/lib/errors";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { appPath } from "@/lib/appPaths";
import type { AppEntry } from "./AppDetailModal";
import { IntegrationIcons } from "./AppDetailModal";
import type { InstallParam } from "../install/types";

// Integration type name → { id, name } of the selected/created instance
type IntegrationSelections = Record<string, { id: string; name: string }>;

interface InstallProgressPanelProps {
  app: AppEntry;
  onClose: () => void;
}

export function InstallProgressPanel({ app, onClose }: InstallProgressPanelProps) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [installParams, setInstallParams] = useState<InstallParam[]>([]);
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  const [integrationSelections, setIntegrationSelections] = useState<IntegrationSelections>({});
  const [isInstalling, setIsInstalling] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(true);

  // Fetch preview to discover params
  useEffect(() => {
    fetch(`/apps/install/preview?repo=${encodeURIComponent(app.repo)}`, { credentials: "include" })
      .then((r) => r.json())
      .then((data) => {
        if (data.installParams && data.installParams.length > 0) {
          setInstallParams(data.installParams);
          const defaults: Record<string, string> = {};
          for (const p of data.installParams) {
            if (p.default) defaults[p.name] = p.default;
          }
          setParamValues(defaults);
        }
      })
      .catch(() => {})
      .finally(() => setPreviewLoading(false));
  }, [app.repo]);

  const doInstall = useCallback(
    async (skipParams: boolean) => {
      if (!organizationId || isInstalling) return;
      setIsInstalling(true);

      try {
        const body: Record<string, unknown> = {
          repo: app.repo,
          name: generateCanvasName(),
          organizationId,
        };

        // Only send params if user chose "Install" (not "Just take me there")
        if (!skipParams && Object.keys(paramValues).length > 0) {
          body.installParams = paramValues;
        }

        if (Object.keys(integrationSelections).length > 0) {
          body.integrations = integrationSelections;
        }

        const response = await fetch("/apps/install", {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });

        if (!response.ok) {
          const message = await response.text();
          throw new Error(message || "Failed to install");
        }

        const result = (await response.json()) as { canvasId: string; organizationId: string };

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
        const message = getUsageLimitToastMessage(error, getApiErrorMessage(error, "Failed to install"));
        showErrorToast(message);
      }
    },
    [organizationId, app, paramValues, integrationSelections, isInstalling, queryClient, navigate],
  );

  const hasIntegrations = app.integrations.length > 0;
  const hasParams = installParams.length > 0;
  const repoUrl = `https://${app.repo}`;

  return (
    <div className="mt-4 rounded-lg bg-white p-5 outline outline-slate-950/10 animate-in slide-in-from-top-2 dark:bg-gray-900">
      {/* Panel 1: App Info */}
      <div className="mb-5">
        <div className="flex items-start gap-3">
          <div className="min-w-0 flex-1">
            <h3 className="text-sm font-semibold text-slate-900">{app.title}</h3>
            <div className="mt-1 flex flex-wrap items-center gap-2">
              <IntegrationIcons integrations={app.integrations} />
              {app.tags.map((tag) => (
                <span
                  key={tag}
                  className="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-medium text-slate-500"
                >
                  {tag}
                </span>
              ))}
            </div>
          </div>
          <a
            href={repoUrl}
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

      {/* Panel 2: Integrations */}
      {hasIntegrations && (
        <div className="mb-5 border-t border-slate-100 pt-4">
          <IntegrationsSection
            integrations={app.integrations}
            organizationId={organizationId ?? ""}
            selections={integrationSelections}
            onSelectionsChange={setIntegrationSelections}
          />
        </div>
      )}

      {/* Panel 3: Parameters */}
      {previewLoading && (
        <div className="mb-5 border-t border-slate-100 pt-4">
          <div className="flex items-center gap-2 text-xs text-slate-400">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
            Loading configuration...
          </div>
        </div>
      )}

      {!previewLoading && hasParams && (
        <div className="mb-5 border-t border-slate-100 pt-4">
          <p className="text-xs font-semibold text-slate-700 mb-3">Configuration</p>
          <div className="space-y-3">
            {installParams.map((param) => (
              <div key={param.name} className="space-y-1">
                <Label htmlFor={`param-${param.name}`} className="text-xs">
                  {param.label}
                  {param.required && <span className="text-red-500 ml-0.5">*</span>}
                </Label>
                <Input
                  id={`param-${param.name}`}
                  value={paramValues[param.name] ?? ""}
                  placeholder={param.placeholder}
                  className="h-8 text-xs"
                  onChange={(e) => setParamValues((prev) => ({ ...prev, [param.name]: e.target.value }))}
                />
                {param.description && <p className="text-[10px] text-slate-400">{param.description}</p>}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Action buttons */}
      {!previewLoading && (
        <div className="flex items-center gap-2 border-t border-slate-100 pt-4">
          <Button size="sm" onClick={() => void doInstall(false)} disabled={isInstalling}>
            {isInstalling ? (
              <>
                <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" />
                Installing...
              </>
            ) : (
              "Install"
            )}
          </Button>
          {hasParams && (
            <Button variant="outline" size="sm" onClick={() => void doInstall(true)} disabled={isInstalling}>
              Just take me there
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={onClose} disabled={isInstalling}>
            Cancel
          </Button>
        </div>
      )}
    </div>
  );
}

// ─── Integrations Section ────────────────────────────────────────────────────

function IntegrationsSection({
  integrations,
  organizationId,
  selections,
  onSelectionsChange,
}: {
  integrations: string[];
  organizationId: string;
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
}) {
  const { data: connected = [], refetch } = useConnectedIntegrations(organizationId, {
    enabled: !!organizationId,
  });
  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const createIntegrationMutation = useCreateIntegration(organizationId, "install_wizard");
  const [dialogIntegrationName, setDialogIntegrationName] = useState<string | null>(null);
  const [configureIntegrationId, setConfigureIntegrationId] = useState<string | null>(null);

  const existingIntegrationNames = useMemo(
    () => new Set(connected.map((i) => i.metadata?.name?.trim()).filter((n): n is string => Boolean(n))),
    [connected],
  );

  const integrationData = useMemo(
    () =>
      integrations.map((name) => {
        const allInstances = connected.filter((item) => item.metadata?.integrationName === name);
        const readyInstances = allInstances.filter((item) => item.status?.state === "ready");
        const nonReadyInstances = allInstances.filter((item) => item.status?.state !== "ready");
        return { name, allInstances, readyInstances, nonReadyInstances };
      }),
    [integrations, connected],
  );

  // Auto-select: first ready instance (covers both single and multiple cases)
  useEffect(() => {
    let changed = false;
    const next = { ...selections };
    for (const data of integrationData) {
      if (next[data.name]) continue;
      if (data.readyInstances.length > 0) {
        const instance = data.readyInstances[0];
        if (instance.metadata?.id && instance.metadata?.name) {
          next[data.name] = { id: instance.metadata.id, name: instance.metadata.name };
          changed = true;
        }
      }
    }
    if (changed) onSelectionsChange(next);
  }, [integrationData, selections, onSelectionsChange]);

  const dialogDefinition = useMemo(
    () => (dialogIntegrationName ? availableIntegrations.find((d) => d.name === dialogIntegrationName) : undefined),
    [availableIntegrations, dialogIntegrationName],
  );

  const dialogPendingInstance = useMemo(() => {
    if (!dialogIntegrationName) return undefined;
    return connected.find((i) => i.metadata?.integrationName === dialogIntegrationName && i.status?.state !== "ready");
  }, [dialogIntegrationName, connected]);

  const initialWebhookSetup = useMemo(() => {
    const webhookUrl = getIntegrationWebhookUrl(dialogPendingInstance?.status?.metadata);
    if (!webhookUrl || !dialogPendingInstance?.metadata?.id) return undefined;
    return {
      id: dialogPendingInstance.metadata.id,
      webhookUrl,
      config: { ...(dialogPendingInstance.spec?.configuration ?? {}) },
    };
  }, [dialogPendingInstance]);

  const defaultDialogName = useMemo(() => {
    if (dialogPendingInstance?.metadata?.name) return dialogPendingInstance.metadata.name;
    if (!dialogIntegrationName) return "";
    return getNextIntegrationName(dialogIntegrationName, existingIntegrationNames);
  }, [dialogIntegrationName, dialogPendingInstance, existingIntegrationNames]);

  const handleCreated = useCallback(
    (integrationId: string, instanceName: string) => {
      if (dialogIntegrationName) {
        onSelectionsChange({
          ...selections,
          [dialogIntegrationName]: { id: integrationId, name: instanceName },
        });
      }
      setDialogIntegrationName(null);
      void refetch();
    },
    [dialogIntegrationName, selections, onSelectionsChange, refetch],
  );

  return (
    <>
      <p className="text-xs font-semibold text-slate-700 mb-3">Integrations</p>
      <div className="space-y-2.5">
        {integrationData.map((data) => {
          const displayName =
            getIntegrationTypeDisplayName(undefined, data.name) ||
            data.name.charAt(0).toUpperCase() + data.name.slice(1);

          // Always show dropdown + "or create new"
          return (
            <div key={data.name} className="flex items-center gap-2">
              <IntegrationIcon integrationName={data.name} className="h-4 w-4" size={16} />
              {data.allInstances.length > 0 ? (
                <Select
                  value={selections[data.name]?.id || ""}
                  onValueChange={(instanceId) => {
                    const instance = data.allInstances.find((i) => i.metadata?.id === instanceId);
                    if (!instance?.metadata?.id) return;

                    // If the selected instance is not ready, open configure dialog
                    if (instance.status?.state !== "ready") {
                      setConfigureIntegrationId(instance.metadata.id);
                      return;
                    }

                    if (instance.metadata.name) {
                      onSelectionsChange({
                        ...selections,
                        [data.name]: { id: instance.metadata.id, name: instance.metadata.name },
                      });
                    }
                  }}
                >
                  <SelectTrigger className="w-56 h-7 text-xs">
                    <SelectValue placeholder={`Select ${displayName}`} />
                  </SelectTrigger>
                  <SelectContent>
                    {data.allInstances.map((instance) => {
                      const state = instance.status?.state;
                      const isReady = state === "ready";
                      return (
                        <SelectItem key={instance.metadata?.id} value={instance.metadata?.id ?? ""}>
                          <span className="flex items-center gap-1.5">
                            <span>{instance.metadata?.name || instance.metadata?.id}</span>
                            {isReady && (
                              <span className="inline-block w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0" />
                            )}
                            {state === "pending" && (
                              <span className="inline-block w-1.5 h-1.5 rounded-full bg-amber-500 shrink-0" />
                            )}
                            {state === "error" && (
                              <span className="inline-block w-1.5 h-1.5 rounded-full bg-red-500 shrink-0" />
                            )}
                          </span>
                        </SelectItem>
                      );
                    })}
                  </SelectContent>
                </Select>
              ) : (
                <span className="text-xs text-slate-400">Not connected</span>
              )}
              <button
                type="button"
                onClick={() => setDialogIntegrationName(data.name)}
                className="text-xs text-blue-600 hover:text-blue-700 hover:underline shrink-0"
              >
                {data.allInstances.length > 0 ? "or create new" : "create new"}
              </button>
            </div>
          );
        })}
      </div>

      <ConfigureIntegrationDialog
        integrationId={configureIntegrationId}
        organizationId={organizationId}
        onClose={() => {
          setConfigureIntegrationId(null);
          void refetch();
        }}
      />

      <IntegrationCreateDialog
        open={!!dialogIntegrationName}
        onOpenChange={(open) => !open && setDialogIntegrationName(null)}
        integrationDefinition={dialogDefinition ?? null}
        organizationId={organizationId}
        onCreateIntegration={async (payload) => {
          const res = await createIntegrationMutation.mutateAsync(payload);
          return res.data;
        }}
        onReset={() => createIntegrationMutation.reset()}
        defaultName={defaultDialogName}
        onCreated={(integrationId, instanceName) => handleCreated(integrationId, instanceName)}
        initialBrowserAction={dialogPendingInstance?.status?.browserAction}
        initialCreatedIntegrationId={dialogPendingInstance?.metadata?.id}
        initialWebhookSetup={initialWebhookSetup}
        initialConfiguration={dialogPendingInstance?.spec?.configuration as Record<string, unknown> | undefined}
      />
    </>
  );
}
