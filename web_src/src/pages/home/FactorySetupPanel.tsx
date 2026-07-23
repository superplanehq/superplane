import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import {
  useAvailableIntegrations,
  useConnectedIntegrations,
  useCreateIntegration,
  useIntegrationResources,
} from "@/hooks/useIntegrations";
import { getIntegrationWebhookUrl } from "@/lib/integrationUtils";
import { getNextIntegrationName } from "@/pages/organization/settings/components/IntegrationSetup/lib";
import { ConfigureIntegrationDialog } from "@/ui/ConfigureIntegrationDialog";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { useEffect, useMemo, useRef, useState } from "react";

import { HomeIntegrationConnectRow, type IntegrationSelections } from "./InstallIntegrationsSection";
import { homeInstallPanelClassName } from "./homePageStyles";

const FACTORY_INTEGRATIONS = ["github", "claude"] as const;

interface FactorySetupPanelProps {
  organizationId?: string;
  busy?: boolean;
  onCancel: () => void;
  onInstall: (selections: IntegrationSelections, repository: string) => void;
  onPreviewWithoutConnecting: () => void;
}

export function FactorySetupPanel({
  organizationId: propOrgId,
  busy = false,
  onCancel,
  onInstall,
  onPreviewWithoutConnecting,
}: FactorySetupPanelProps) {
  const routeOrgId = useOrganizationId();
  const organizationId = propOrgId || routeOrgId || "";
  const [selections, setSelections] = useState<IntegrationSelections>({});
  const [repository, setRepository] = useState("");
  const [dialogIntegrationName, setDialogIntegrationName] = useState<string | null>(null);
  const [configureIntegrationId, setConfigureIntegrationId] = useState<string | null>(null);
  const pendingConnectKeyRef = useRef<string | null>(null);

  const { data: connected = [], refetch } = useConnectedIntegrations(organizationId, {
    enabled: !!organizationId,
  });
  const { data: availableIntegrations = [] } = useAvailableIntegrations({ enabled: !!organizationId });
  const createIntegrationMutation = useCreateIntegration(organizationId, "install_wizard");

  const existingIntegrationNames = useMemo(
    () => new Set(connected.map((i) => i.metadata?.name?.trim()).filter((n): n is string => Boolean(n))),
    [connected],
  );

  useEffect(() => {
    let changed = false;
    const next = { ...selections };

    for (const name of FACTORY_INTEGRATIONS) {
      if (next[name]) {
        const selected = connected.find((i) => i.metadata?.id === next[name].id);
        if (selected && selected.status?.state !== "ready") {
          delete next[name];
          changed = true;
        }
      }

      if (!next[name]) {
        const firstReady = connected.find((i) => i.metadata?.integrationName === name && i.status?.state === "ready");
        if (firstReady?.metadata?.id && firstReady.metadata.name) {
          next[name] = { id: firstReady.metadata.id, name: firstReady.metadata.name };
          changed = true;
        }
      }
    }

    if (changed) setSelections(next);
  }, [connected, selections]);

  const githubConnected = Boolean(selections.github);
  const allConnected = FACTORY_INTEGRATIONS.every((name) => selections[name]);
  const canInstall = !busy && allConnected && repository.trim() !== "";

  const dialogDefinition = useMemo(
    () => (dialogIntegrationName ? availableIntegrations.find((d) => d.name === dialogIntegrationName) : undefined),
    [availableIntegrations, dialogIntegrationName],
  );

  const dialogPendingInstance = useMemo(
    () =>
      dialogIntegrationName
        ? connected.find((i) => i.metadata?.integrationName === dialogIntegrationName && i.status?.state !== "ready")
        : undefined,
    [dialogIntegrationName, connected],
  );

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

  const openConnect = (integrationName: string) => {
    const pending = connected.find(
      (i) => i.metadata?.integrationName === integrationName && i.status?.state !== "ready",
    );
    if (pending?.metadata?.id) {
      setConfigureIntegrationId(pending.metadata.id);
      return;
    }
    pendingConnectKeyRef.current = integrationName;
    setDialogIntegrationName(integrationName);
  };

  return (
    <div className={homeInstallPanelClassName} role="region" aria-label="Software Factory setup">
      <div className="mb-5">
        <h3 className="text-base font-medium text-slate-900 dark:text-gray-100">Connect your GitHub and Claude</h3>
        <p className="mt-1 text-sm text-slate-600 dark:text-gray-400">
          This will create software factory that automates your delivery from trigger to pull request.
        </p>
      </div>

      <div className="mb-5">
        <div className="divide-y divide-slate-200 rounded-md border border-slate-200 dark:divide-gray-700/70 dark:border-gray-700/70">
          {FACTORY_INTEGRATIONS.map((name) => (
            <HomeIntegrationConnectRow
              key={name}
              name={name}
              connected={Boolean(selections[name])}
              onConnect={() => openConnect(name)}
            />
          ))}
        </div>
      </div>

      {githubConnected && (
        <div className="mb-5 pt-4">
          <Label
            htmlFor="factory-repository"
            className="mb-2 block text-xs font-semibold text-slate-700 dark:text-gray-300"
          >
            Choose repository
          </Label>
          <FactoryRepositorySelect
            organizationId={organizationId}
            integrationId={selections.github?.id ?? ""}
            value={repository}
            onChange={setRepository}
          />
        </div>
      )}

      <div className="flex flex-wrap items-center pt-4">
        <div className="flex items-center gap-2.5">
          <Button type="button" disabled={!canInstall} onClick={() => onInstall(selections, repository.trim())}>
            Install
          </Button>
          <Button type="button" variant="outline" onClick={onCancel} disabled={busy}>
            Cancel
          </Button>
        </div>
        <Button
          type="button"
          variant="link"
          disabled={busy}
          onClick={onPreviewWithoutConnecting}
          className="ml-4 h-auto p-0 text-xs font-normal text-gray-800 underline decoration-gray-300 underline-offset-4 dark:text-gray-200 dark:decoration-gray-600"
        >
          Let me preview the app without connecting
        </Button>
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
        onOpenChange={(open) => {
          if (!open) {
            setDialogIntegrationName(null);
            createIntegrationMutation.reset();
          }
        }}
        integrationDefinition={dialogDefinition ?? null}
        organizationId={organizationId}
        onCreateIntegration={async (payload) => {
          const res = await createIntegrationMutation.mutateAsync(payload);
          return res.data;
        }}
        onReset={() => createIntegrationMutation.reset()}
        defaultName={defaultDialogName}
        onCreated={(integrationId, instanceName) => {
          // Dialog calls onOpenChange(false) before onCreated; keep the key in a ref across that close.
          const key = pendingConnectKeyRef.current;
          pendingConnectKeyRef.current = null;
          if (key) {
            setSelections((prev) => ({
              ...prev,
              [key]: { id: integrationId, name: instanceName },
            }));
            if (key === "github") {
              setRepository("");
            }
          }
          setDialogIntegrationName(null);
          void refetch();
        }}
        initialBrowserAction={dialogPendingInstance?.status?.browserAction}
        initialCreatedIntegrationId={dialogPendingInstance?.metadata?.id}
        initialWebhookSetup={initialWebhookSetup}
        initialConfiguration={dialogPendingInstance?.spec?.configuration as Record<string, unknown> | undefined}
      />
    </div>
  );
}

function FactoryRepositorySelect({
  organizationId,
  integrationId,
  value,
  onChange,
}: {
  organizationId: string;
  integrationId: string;
  value: string;
  onChange: (value: string) => void;
}) {
  const {
    data: resources = [],
    isLoading,
    isError,
    refetch,
  } = useIntegrationResources(organizationId, integrationId, "repository");
  const options = useMemo(
    () =>
      resources
        .map((resource) => {
          const name = resource.name?.trim();
          if (!name) return null;
          return { value: name, label: name };
        })
        .filter((option): option is { value: string; label: string } => option !== null),
    [resources],
  );

  if (isError) {
    return (
      <div className="space-y-2">
        <p className="text-xs text-red-600 dark:text-red-400">Couldn't load repositories. Try again.</p>
        <Button type="button" variant="outline" size="sm" onClick={() => void refetch()}>
          Retry
        </Button>
      </div>
    );
  }

  if (!isLoading && options.length === 0) {
    return (
      <p className="text-xs text-slate-500 dark:text-gray-400">
        No repositories found for this GitHub connection. Check access, then try again.
      </p>
    );
  }

  return (
    <Select value={value || undefined} onValueChange={onChange} disabled={isLoading || options.length === 0}>
      <SelectTrigger id="factory-repository" className="w-full">
        <SelectValue placeholder={isLoading ? "Loading repositories…" : "Select a repository"} />
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
