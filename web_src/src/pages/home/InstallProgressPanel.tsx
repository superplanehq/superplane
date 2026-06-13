import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Check, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { canvasKeys } from "@/hooks/useCanvasData";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
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
import type { InstallParam } from "../install/types";

// Flow: integrations (optional) → params (optional) → installing → done
type Phase = "integrations" | "configuring" | "installing" | "done";

interface Step {
  label: string;
  done: boolean;
}

interface InstallResult {
  canvasId: string;
  organizationId: string;
}

// Integration type name → { id, name } of the selected/created instance
type IntegrationSelections = Record<string, { id: string; name: string }>;

interface InstallProgressPanelProps {
  app: AppEntry;
  onClose: () => void;
}

function getInitialPhase(app: AppEntry): Phase {
  if (app.integrations.length > 0) return "integrations";
  return "installing";
}

export function InstallProgressPanel({ app, onClose }: InstallProgressPanelProps) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [phase, setPhase] = useState<Phase>(() => getInitialPhase(app));
  const [steps, setSteps] = useState<Step[]>([
    { label: "Cloning repo", done: false },
    { label: "Initializing Canvas", done: false },
    { label: "Setting up Console", done: false },
  ]);
  const [installResult, setInstallResult] = useState<InstallResult | null>(null);
  const [installParams, setInstallParams] = useState<InstallParam[]>([]);
  const [paramValues, setParamValues] = useState<Record<string, string>>({});
  const [integrationSelections, setIntegrationSelections] = useState<IntegrationSelections>({});
  const installTriggered = useRef(false);

  // Fetch preview to get params
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
      .catch(() => {});
  }, [app.repo]);

  // Run install when phase transitions to "installing"
  useEffect(() => {
    if (phase !== "installing" || !organizationId || installTriggered.current) return;
    installTriggered.current = true;

    const doInstall = async () => {
      try {
        // Step 1: Cloning
        await sleep(400);
        setSteps((s) => s.map((step, i) => (i === 0 ? { ...step, done: true } : step)));

        // Build install body with integrations and params
        const body: Record<string, unknown> = {
          repo: app.repo,
          name: generateCanvasName(),
          organizationId,
        };

        if (Object.keys(paramValues).length > 0) {
          body.installParams = paramValues;
        }

        if (Object.keys(integrationSelections).length > 0) {
          body.integrations = integrationSelections;
        }

        // Step 2: Canvas
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

        const result = (await response.json()) as InstallResult;
        setInstallResult(result);
        setSteps((s) => s.map((step, i) => (i <= 1 ? { ...step, done: true } : step)));

        // Step 3: Console
        await sleep(300);
        setSteps((s) => s.map((step) => ({ ...step, done: true })));

        await queryClient.refetchQueries({ queryKey: canvasKeys.list(result.organizationId) });

        if (app.agentInstructions || app.agentInitialMessage) {
          setAgentBootContext(result.canvasId, {
            instructions: app.agentInstructions,
            initialMessage: app.agentInitialMessage,
          });
        }

        setPhase("done");
      } catch (error) {
        const message = getUsageLimitToastMessage(error, getApiErrorMessage(error, "Failed to install"));
        showErrorToast(message);
        onClose();
      }
    };

    void doInstall();
  }, [phase, organizationId, app, queryClient, onClose, paramValues, integrationSelections]);

  const handleGoToApp = useCallback(() => {
    if (!installResult) return;
    localStorage.setItem("canvasAgentSidebarOpen", "true");
    localStorage.setItem("canvasSidebarOpen", "false");
    navigate(appPath(installResult.organizationId, installResult.canvasId, "?edit=1"));
  }, [installResult, navigate]);

  const advanceFromIntegrations = useCallback(
    (selections: IntegrationSelections) => {
      setIntegrationSelections(selections);
      if (installParams.length > 0) {
        setPhase("configuring");
      } else {
        setPhase("installing");
      }
    },
    [installParams.length],
  );

  const skipToInstall = useCallback(() => {
    setPhase("installing");
  }, []);

  return (
    <div className="mt-4 rounded-lg bg-white p-5 outline outline-slate-950/10 animate-in slide-in-from-top-2 dark:bg-gray-900">
      {phase === "integrations" && (
        <IntegrationsStep
          integrations={app.integrations}
          organizationId={organizationId ?? ""}
          onNext={advanceFromIntegrations}
          onSkip={skipToInstall}
        />
      )}

      {phase === "configuring" && (
        <div className="space-y-4">
          <p className="text-sm font-medium text-slate-700">Configure {app.title}</p>
          {installParams.map((param) => (
            <div key={param.name} className="space-y-1.5">
              <Label htmlFor={`param-${param.name}`}>
                {param.label}
                {param.required && <span className="text-red-500 ml-0.5">*</span>}
              </Label>
              <Input
                id={`param-${param.name}`}
                value={paramValues[param.name] ?? ""}
                placeholder={param.placeholder}
                onChange={(e) => setParamValues((prev) => ({ ...prev, [param.name]: e.target.value }))}
              />
              {param.description && <p className="text-xs text-slate-500">{param.description}</p>}
            </div>
          ))}
          <div className="flex items-center gap-3 pt-2">
            <Button variant="default" size="sm" onClick={() => setPhase("installing")}>
              Install
            </Button>
            <Button variant="outline" size="sm" onClick={skipToInstall}>
              Skip
            </Button>
          </div>
        </div>
      )}

      {phase === "installing" && (
        <div className="space-y-3">
          <p className="text-sm font-medium text-slate-700">Installing {app.title}...</p>
          {steps.map((step) => (
            <div key={step.label} className="flex items-center gap-2.5 text-sm">
              {step.done ? (
                <Check className="h-4 w-4 text-green-500" />
              ) : (
                <Loader2 className="h-4 w-4 animate-spin text-slate-400" />
              )}
              <span className={step.done ? "text-slate-700" : "text-slate-500"}>{step.label}</span>
            </div>
          ))}
        </div>
      )}

      {phase === "done" && (
        <div className="space-y-4">
          <div className="space-y-3">
            {steps.map((step) => (
              <div key={step.label} className="flex items-center gap-2.5 text-sm">
                <Check className="h-4 w-4 text-green-500" />
                <span className="text-slate-700">{step.label}</span>
              </div>
            ))}
          </div>
          <div className="flex items-center gap-3 pt-2">
            <Button variant="default" size="sm" onClick={handleGoToApp}>
              Open App
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

function IntegrationsStep({
  integrations,
  organizationId,
  onNext,
  onSkip,
}: {
  integrations: string[];
  organizationId: string;
  onNext: (selections: IntegrationSelections) => void;
  onSkip: () => void;
}) {
  const { data: connected = [], refetch } = useConnectedIntegrations(organizationId, {
    enabled: !!organizationId,
  });
  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const createIntegrationMutation = useCreateIntegration(organizationId, "install_wizard");
  const [dialogIntegrationName, setDialogIntegrationName] = useState<string | null>(null);

  // Track which instance is selected for each integration type
  const [selections, setSelections] = useState<IntegrationSelections>({});

  const existingIntegrationNames = useMemo(
    () => new Set(connected.map((i) => i.metadata?.name?.trim()).filter((n): n is string => Boolean(n))),
    [connected],
  );

  // For each required integration type, find all ready instances
  const integrationData = useMemo(
    () =>
      integrations.map((name) => {
        const readyInstances = connected.filter(
          (item) => item.metadata?.integrationName === name && item.status?.state === "ready",
        );
        const pendingInstance = connected.find(
          (item) => item.metadata?.integrationName === name && item.status?.state !== "ready",
        );

        return {
          name,
          readyInstances,
          pendingInstance,
          pendingState: pendingInstance
            ? ((pendingInstance.status?.state === "error" ? "error" : "pending") as "error" | "pending")
            : undefined,
          stateDescription: pendingInstance?.status?.stateDescription,
        };
      }),
    [integrations, connected],
  );

  // Auto-select when there's exactly one ready instance
  useEffect(() => {
    setSelections((prev) => {
      const next = { ...prev };
      let changed = false;
      for (const data of integrationData) {
        if (next[data.name]) continue;
        if (data.readyInstances.length === 1) {
          const instance = data.readyInstances[0];
          if (instance.metadata?.id && instance.metadata?.name) {
            next[data.name] = { id: instance.metadata.id, name: instance.metadata.name };
            changed = true;
          }
        }
      }
      return changed ? next : prev;
    });
  }, [integrationData]);

  const allResolved = integrationData.every((data) => selections[data.name] || data.readyInstances.length > 0);

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
        setSelections((prev) => ({
          ...prev,
          [dialogIntegrationName]: { id: integrationId, name: instanceName },
        }));
      }
      setDialogIntegrationName(null);
      void refetch();
    },
    [dialogIntegrationName, refetch],
  );

  const handleNext = useCallback(() => {
    // Build final selections: use explicit selections, or auto-select the single ready instance
    const finalSelections: IntegrationSelections = {};
    for (const data of integrationData) {
      if (selections[data.name]) {
        finalSelections[data.name] = selections[data.name];
      } else if (data.readyInstances.length === 1) {
        const instance = data.readyInstances[0];
        if (instance.metadata?.id && instance.metadata?.name) {
          finalSelections[data.name] = { id: instance.metadata.id, name: instance.metadata.name };
        }
      }
    }
    onNext(finalSelections);
  }, [integrationData, selections, onNext]);

  return (
    <div className="space-y-4">
      <p className="text-sm font-medium text-slate-700">Connect Integrations</p>
      <p className="text-xs text-slate-500">This app requires the following integrations to be connected.</p>

      <div className="space-y-3">
        {integrationData.map((data) => {
          const displayName =
            getIntegrationTypeDisplayName(undefined, data.name) ||
            data.name.charAt(0).toUpperCase() + data.name.slice(1);

          // No ready instances — show badge to create
          if (data.readyInstances.length === 0) {
            const state = data.pendingState || "missing";
            return (
              <div key={data.name} className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setDialogIntegrationName(data.name)}
                  title={
                    state === "pending"
                      ? `Pending: ${displayName} — ${data.stateDescription || "setup in progress"}`
                      : state === "error"
                        ? `Error: ${displayName} — ${data.stateDescription || "setup failed"}`
                        : `Connect ${displayName}`
                  }
                  className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border text-xs font-medium transition-all ${badgeClassName(state)}`}
                >
                  <IntegrationIcon integrationName={data.name} className="h-4 w-4" size={16} />
                  <span>{displayName}</span>
                  {badgeDot(state)}
                </button>
              </div>
            );
          }

          // Exactly one ready instance — show green badge (auto-selected)
          if (data.readyInstances.length === 1) {
            const instance = data.readyInstances[0];
            return (
              <div key={data.name} className="flex items-center gap-2">
                <div className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-emerald-200 text-emerald-700 bg-white text-xs font-medium">
                  <IntegrationIcon integrationName={data.name} className="h-4 w-4" size={16} />
                  <span>{instance.metadata?.name || displayName}</span>
                  <span className="inline-block w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0" />
                </div>
              </div>
            );
          }

          // Multiple ready instances — show select dropdown
          return (
            <div key={data.name} className="flex items-center gap-2">
              <IntegrationIcon integrationName={data.name} className="h-4 w-4" size={16} />
              <Select
                value={selections[data.name]?.id || ""}
                onValueChange={(instanceId) => {
                  const instance = data.readyInstances.find((i) => i.metadata?.id === instanceId);
                  if (instance?.metadata?.id && instance?.metadata?.name) {
                    setSelections((prev) => ({
                      ...prev,
                      [data.name]: { id: instance.metadata!.id!, name: instance.metadata!.name! },
                    }));
                  }
                }}
              >
                <SelectTrigger className="w-64 h-8 text-xs">
                  <SelectValue placeholder={`Select ${displayName} instance`} />
                </SelectTrigger>
                <SelectContent>
                  {data.readyInstances.map((instance) => (
                    <SelectItem key={instance.metadata?.id} value={instance.metadata?.id ?? ""}>
                      {instance.metadata?.name || instance.metadata?.id}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <button
                type="button"
                onClick={() => setDialogIntegrationName(data.name)}
                className="text-xs text-blue-600 hover:text-blue-700 hover:underline"
              >
                or create new
              </button>
            </div>
          );
        })}
      </div>

      <div className="flex items-center gap-3 pt-2">
        <Button variant="default" size="sm" onClick={handleNext} disabled={!allResolved}>
          {allResolved ? "Next" : "Connect all integrations to continue"}
        </Button>
        <Button variant="outline" size="sm" onClick={onSkip}>
          Skip
        </Button>
      </div>

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
    </div>
  );
}

function badgeClassName(state: string) {
  switch (state) {
    case "ready":
      return "border-emerald-200 text-emerald-700 bg-white";
    case "pending":
      return "border-amber-200 text-amber-700 bg-white hover:bg-amber-50 hover:border-amber-300 cursor-pointer";
    case "error":
      return "border-red-200 text-red-700 bg-white hover:bg-red-50 hover:border-red-300 cursor-pointer";
    default:
      return "border-slate-200 text-slate-700 hover:bg-slate-50 hover:border-slate-300 cursor-pointer";
  }
}

function badgeDot(state: string) {
  switch (state) {
    case "ready":
      return <span className="inline-block w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0" />;
    case "pending":
      return <span className="inline-block w-1.5 h-1.5 rounded-full bg-amber-500 shrink-0" />;
    case "error":
      return <span className="inline-block w-1.5 h-1.5 rounded-full bg-red-500 shrink-0" />;
    default:
      return <span className="text-[10px] leading-none text-slate-400 font-bold shrink-0">+</span>;
  }
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
