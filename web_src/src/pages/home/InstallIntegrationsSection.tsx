import type { OrganizationsIntegration } from "@/api-client";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { ConfigureIntegrationDialog } from "@/ui/ConfigureIntegrationDialog";
import { getIntegrationWebhookUrl } from "@/lib/integrationUtils";
import { getNextIntegrationName } from "@/pages/organization/settings/components/IntegrationSetup/lib";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";

export type IntegrationSelections = Record<string, { id: string; name: string }>;

type IntegrationInstanceSummary = {
  name: string;
  allInstances: OrganizationsIntegration[];
  readyInstances: OrganizationsIntegration[];
};

/**
 * Clears selections pointing to non-ready instances and auto-selects
 * the first ready instance for unselected types. Returns updated
 * selections if anything changed, or null if no changes needed.
 */
function syncSelectionsWithInstances(
  integrationData: IntegrationInstanceSummary[],
  selections: IntegrationSelections,
): IntegrationSelections | null {
  let changed = false;
  const next = { ...selections };

  for (const data of integrationData) {
    if (next[data.name]) {
      const selected = data.allInstances.find((i) => i.metadata?.id === next[data.name].id);
      if (selected && selected.status?.state !== "ready") {
        delete next[data.name];
        changed = true;
      }
    }

    if (!next[data.name] && data.readyInstances.length > 0) {
      const first = data.readyInstances[0];
      if (first.metadata?.id && first.metadata?.name) {
        next[data.name] = { id: first.metadata.id, name: first.metadata.name };
        changed = true;
      }
    }
  }

  return changed ? next : null;
}

export function IntegrationsSection({
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
        return { name, allInstances, readyInstances };
      }),
    [integrations, connected],
  );

  useEffect(() => {
    const synced = syncSelectionsWithInstances(integrationData, selections);
    if (synced) onSelectionsChange(synced);
  }, [integrationData, selections, onSelectionsChange]);

  const { dialogDefinition, dialogPendingInstance, initialWebhookSetup, defaultDialogName } = useCreateDialogProps(
    dialogIntegrationName,
    availableIntegrations,
    connected,
    existingIntegrationNames,
  );

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
        {integrationData.map((data) => (
          <IntegrationRow
            key={data.name}
            data={data}
            selectedId={selections[data.name]?.id}
            onSelect={(id, name) => onSelectionsChange({ ...selections, [data.name]: { id, name } })}
            onConfigure={setConfigureIntegrationId}
            onCreateNew={() => setDialogIntegrationName(data.name)}
          />
        ))}
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

function buildWebhookSetup(pending: OrganizationsIntegration | undefined) {
  const webhookUrl = getIntegrationWebhookUrl(pending?.status?.metadata);
  if (!webhookUrl || !pending?.metadata?.id) return undefined;
  return { id: pending.metadata.id, webhookUrl, config: { ...(pending.spec?.configuration ?? {}) } };
}

function resolveDefaultDialogName(
  dialogIntegrationName: string | null,
  pending: OrganizationsIntegration | undefined,
  existingNames: Set<string>,
): string {
  if (pending?.metadata?.name) return pending.metadata.name;
  if (!dialogIntegrationName) return "";
  return getNextIntegrationName(dialogIntegrationName, existingNames);
}

function useCreateDialogProps(
  dialogIntegrationName: string | null,
  availableIntegrations: Array<{ name?: string; [key: string]: unknown }>,
  connected: OrganizationsIntegration[],
  existingIntegrationNames: Set<string>,
) {
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

  const initialWebhookSetup = useMemo(() => buildWebhookSetup(dialogPendingInstance), [dialogPendingInstance]);

  const defaultDialogName = useMemo(
    () => resolveDefaultDialogName(dialogIntegrationName, dialogPendingInstance, existingIntegrationNames),
    [dialogIntegrationName, dialogPendingInstance, existingIntegrationNames],
  );

  return { dialogDefinition, dialogPendingInstance, initialWebhookSetup, defaultDialogName };
}

function IntegrationRow({
  data,
  selectedId,
  onSelect,
  onConfigure,
  onCreateNew,
}: {
  data: IntegrationInstanceSummary;
  selectedId?: string;
  onSelect: (id: string, name: string) => void;
  onConfigure: (id: string) => void;
  onCreateNew: () => void;
}) {
  const displayName =
    getIntegrationTypeDisplayName(undefined, data.name) || data.name.charAt(0).toUpperCase() + data.name.slice(1);

  const handleInstanceSelect = (instanceId: string) => {
    const instance = data.allInstances.find((i) => i.metadata?.id === instanceId);
    if (!instance?.metadata?.id) return;
    if (instance.status?.state !== "ready") {
      onConfigure(instance.metadata.id);
      return;
    }
    if (instance.metadata.name) {
      onSelect(instance.metadata.id, instance.metadata.name);
    }
  };

  return (
    <div className="flex items-center gap-2">
      <IntegrationIcon integrationName={data.name} className="h-4 w-4" size={16} />
      {data.allInstances.length > 0 ? (
        <Select value={selectedId || ""} onValueChange={handleInstanceSelect}>
          <SelectTrigger className="w-56 h-7 text-xs">
            <SelectValue placeholder={`Select ${displayName}`} />
          </SelectTrigger>
          <SelectContent>
            {data.allInstances.map((instance) => (
              <SelectItem key={instance.metadata?.id} value={instance.metadata?.id ?? ""}>
                <span className="flex items-center gap-1.5">
                  <span>{instance.metadata?.name || instance.metadata?.id}</span>
                  <StatusDot state={instance.status?.state} />
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : (
        <span className="text-xs text-slate-400">Not connected</span>
      )}
      <button
        type="button"
        onClick={onCreateNew}
        className="text-xs text-blue-600 hover:text-blue-700 hover:underline shrink-0"
      >
        {data.allInstances.length > 0 ? "or create new" : "create new"}
      </button>
    </div>
  );
}

const STATUS_DOT_COLORS: Record<string, string> = {
  ready: "bg-emerald-500",
  pending: "bg-amber-500",
  error: "bg-red-500",
};

function StatusDot({ state }: { state?: string }) {
  const color = state ? STATUS_DOT_COLORS[state] : undefined;
  if (!color) return null;
  return <span className={`inline-block w-1.5 h-1.5 rounded-full shrink-0 ${color}`} />;
}
