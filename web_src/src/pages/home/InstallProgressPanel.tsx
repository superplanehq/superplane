import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Check, Loader2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
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

type Phase = "installing" | "done" | "integrations" | "configuring";

interface Step {
  label: string;
  done: boolean;
}

interface InstallResult {
  canvasId: string;
  organizationId: string;
}

interface InstallProgressPanelProps {
  app: AppEntry;
  onClose: () => void;
}

export function InstallProgressPanel({ app, onClose }: InstallProgressPanelProps) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [phase, setPhase] = useState<Phase>("installing");
  const [steps, setSteps] = useState<Step[]>([
    { label: "Cloning repo", done: false },
    { label: "Initializing Canvas", done: false },
    { label: "Setting up Console", done: false },
  ]);
  const [installResult, setInstallResult] = useState<InstallResult | null>(null);
  const [installParams, setInstallParams] = useState<InstallParam[]>([]);
  const [paramValues, setParamValues] = useState<Record<string, string>>({});

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

  // Run install on mount
  useEffect(() => {
    if (!organizationId) return;

    const doInstall = async () => {
      try {
        // Step 1: Cloning
        await sleep(400);
        setSteps((s) => s.map((step, i) => (i === 0 ? { ...step, done: true } : step)));

        // Step 2: Canvas
        const response = await fetch("/apps/install", {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            repo: app.repo,
            name: generateCanvasName(),
            organizationId,
          }),
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
  }, [organizationId, app, queryClient, onClose]);

  const handleGoToApp = useCallback(() => {
    if (!installResult) return;
    localStorage.setItem("canvasAgentSidebarOpen", "true");
    localStorage.setItem("canvasSidebarOpen", "false");
    navigate(appPath(installResult.organizationId, installResult.canvasId, "?edit=1"));
  }, [installResult, navigate]);

  const handleConfigure = useCallback(() => {
    setPhase("configuring");
  }, []);

  const handleApplyParams = useCallback(async () => {
    if (!installResult || !organizationId) return;

    try {
      // Re-install with params by deleting and recreating
      // For now, just navigate to the app — the agent can help configure
      handleGoToApp();
    } catch {
      showErrorToast("Failed to apply configuration");
    }
  }, [installResult, organizationId, handleGoToApp]);

  return (
    <div className="mt-4 rounded-lg bg-white p-5 outline outline-slate-950/10 animate-in slide-in-from-top-2 dark:bg-gray-900">
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
            {(app.integrations.length > 0 || installParams.length > 0) && (
              <Button
                variant="default"
                size="sm"
                onClick={() => setPhase(app.integrations.length > 0 ? "integrations" : "configuring")}
              >
                Configure
              </Button>
            )}
            <Button
              variant={app.integrations.length > 0 || installParams.length > 0 ? "outline" : "default"}
              size="sm"
              onClick={handleGoToApp}
            >
              {app.integrations.length > 0 || installParams.length > 0 ? "Just take me there" : "Open App"}
            </Button>
          </div>
        </div>
      )}

      {phase === "integrations" && (
        <IntegrationsStep
          integrations={app.integrations}
          organizationId={organizationId ?? ""}
          onNext={() => (installParams.length > 0 ? setPhase("configuring") : handleGoToApp())}
          onSkip={handleGoToApp}
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
            <Button variant="default" size="sm" onClick={handleApplyParams}>
              Apply & Open App
            </Button>
            <Button variant="outline" size="sm" onClick={handleGoToApp}>
              Skip
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
  onNext: () => void;
  onSkip: () => void;
}) {
  const { data: connected = [], refetch } = useConnectedIntegrations(organizationId, {
    enabled: !!organizationId,
  });
  const { data: availableIntegrations = [] } = useAvailableIntegrations();
  const createIntegrationMutation = useCreateIntegration(organizationId, "install_wizard");
  const [dialogIntegrationName, setDialogIntegrationName] = useState<string | null>(null);

  const existingIntegrationNames = useMemo(
    () => new Set(connected.map((i) => i.metadata?.name?.trim()).filter((n): n is string => Boolean(n))),
    [connected],
  );

  const integrationStatus = useMemo(
    () =>
      integrations.map((name) => {
        const readyInstance = connected.find(
          (item) => item.metadata?.integrationName === name && item.status?.state === "ready",
        );
        if (readyInstance) return { name, state: "ready" as const };

        const pendingInstance = connected.find(
          (item) => item.metadata?.integrationName === name && item.status?.state !== "ready",
        );
        if (pendingInstance) {
          const rawState = pendingInstance.status?.state;
          return {
            name,
            state: (rawState === "error" ? "error" : "pending") as "error" | "pending",
            stateDescription: pendingInstance.status?.stateDescription,
            pendingInstance,
          };
        }

        return { name, state: "missing" as const };
      }),
    [integrations, connected],
  );

  const allReady = integrationStatus.every((i) => i.state === "ready");

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

  const handleCreated = useCallback(() => {
    setDialogIntegrationName(null);
    void refetch();
  }, [refetch]);

  const badgeClassName = (state: string) => {
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
  };

  const badgeDot = (state: string) => {
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
  };

  return (
    <div className="space-y-4">
      <p className="text-sm font-medium text-slate-700">Connect Integrations</p>
      <p className="text-xs text-slate-500">This app requires the following integrations to be connected.</p>
      <div className="flex flex-wrap gap-2">
        {integrationStatus.map((integration) => {
          const displayName =
            getIntegrationTypeDisplayName(undefined, integration.name) ||
            integration.name.charAt(0).toUpperCase() + integration.name.slice(1);
          return (
            <button
              key={integration.name}
              type="button"
              onClick={() => {
                if (integration.state !== "ready") {
                  setDialogIntegrationName(integration.name);
                }
              }}
              title={
                integration.state === "ready"
                  ? `Connected: ${displayName}`
                  : integration.state === "pending"
                    ? `Pending: ${displayName} — ${integration.stateDescription || "setup in progress"}`
                    : integration.state === "error"
                      ? `Error: ${displayName} — ${integration.stateDescription || "setup failed"}`
                      : `Connect ${displayName}`
              }
              className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border text-xs font-medium transition-all ${badgeClassName(integration.state)}`}
            >
              <IntegrationIcon integrationName={integration.name} className="h-4 w-4" size={16} />
              <span>{displayName}</span>
              {badgeDot(integration.state)}
            </button>
          );
        })}
      </div>
      <div className="flex items-center gap-3 pt-2">
        <Button variant="default" size="sm" onClick={onNext} disabled={!allReady}>
          {allReady ? "Next" : "Connect all integrations to continue"}
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
        onCreated={() => void handleCreated()}
        initialBrowserAction={dialogPendingInstance?.status?.browserAction}
        initialCreatedIntegrationId={dialogPendingInstance?.metadata?.id}
        initialWebhookSetup={initialWebhookSetup}
        initialConfiguration={dialogPendingInstance?.spec?.configuration as Record<string, unknown> | undefined}
      />
    </div>
  );
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
