import type { OrganizationsCreateIntegrationResponse, OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useEffect, useMemo, useRef, useState, type MutableRefObject } from "react";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationCreateDialog, type IntegrationCreatePayload } from "@/ui/IntegrationCreateDialog";
import { ConfigureIntegrationDialog } from "@/ui/ConfigureIntegrationDialog";
import { getIntegrationWebhookUrl } from "@/lib/integrationUtils";
import { getNextIntegrationName } from "@/pages/organization/settings/components/IntegrationSetup/lib";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";

import { HomeIntegrationConnectRow, StatusDot } from "./HomeIntegrationConnectRow";
import {
  resolveHomeIntegrationStatus,
  syncSelectionsWithInstances,
  type IntegrationInstanceSummary,
  type IntegrationSelections,
} from "./homeIntegrationStatus";
import { useHomeIntegrationConnectActions } from "./useHomeIntegrationConnectActions";

export type { IntegrationSelections };

function resolveIntegrationHomeHref(args: {
  organizationId: string;
  dialogIntegrationName: string | null;
  dialogMode: "create" | "resume";
  pendingId?: string;
  selectedId?: string;
}) {
  if (!args.organizationId) return undefined;
  const integrationId = (args.dialogMode === "resume" ? args.pendingId : undefined) ?? args.selectedId;
  if (integrationId) return `/${args.organizationId}/settings/integrations/${integrationId}`;
  return `/${args.organizationId}/settings/integrations`;
}

export function IntegrationsSection({
  integrations,
  organizationId,
  selections,
  onSelectionsChange,
  variant = "select",
}: {
  integrations: string[];
  organizationId: string;
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  /** `select` shows an instance picker; `status` shows Connected / Not connected rows. */
  variant?: "select" | "status";
}) {
  const { data: connected = [], refetch } = useConnectedIntegrations(organizationId, {
    enabled: !!organizationId,
  });
  const { data: availableIntegrations = [] } = useAvailableIntegrations({ enabled: !!organizationId });
  const createIntegrationMutation = useCreateIntegration(organizationId, "install_wizard");
  const [dialogIntegrationName, setDialogIntegrationName] = useState<string | null>(null);
  /** "create" skips resuming a pending instance so "Create new" always starts fresh. */
  const [dialogMode, setDialogMode] = useState<"create" | "resume">("resume");
  const [configureIntegrationId, setConfigureIntegrationId] = useState<string | null>(null);
  const pendingConnectKeyRef = useRef<string | null>(null);
  const existingIntegrationNames = useMemo(
    () => new Set(connected.map((i) => i.metadata?.name?.trim()).filter((n): n is string => Boolean(n))),
    [connected],
  );
  const integrationData = useMemo(
    () =>
      integrations.map((name) => {
        const allInstances = connected.filter((item) => item.metadata?.integrationName === name);
        return { name, allInstances, readyInstances: allInstances.filter((item) => item.status?.state === "ready") };
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
  const integrationHomeHref = useMemo(
    () =>
      resolveIntegrationHomeHref({
        organizationId,
        dialogIntegrationName,
        dialogMode,
        pendingId: dialogPendingInstance?.metadata?.id,
        selectedId: dialogIntegrationName ? selections[dialogIntegrationName]?.id : undefined,
      }),
    [organizationId, dialogIntegrationName, dialogMode, dialogPendingInstance?.metadata?.id, selections],
  );
  const { openCapabilitySetup, openCreateIntegrationModal, openConnectDialog, openConfigureDialog } =
    useHomeIntegrationConnectActions({
      organizationId,
      availableIntegrations,
      connected,
      pendingConnectKeyRef,
      setDialogMode,
      setDialogIntegrationName,
      setConfigureIntegrationId,
    });

  return (
    <>
      <IntegrationList
        integrationData={integrationData}
        variant={variant}
        selections={selections}
        onSelectionsChange={onSelectionsChange}
        onConnect={openConnectDialog}
        onConfigure={openConfigureDialog}
        onCreateNew={openCreateIntegrationModal}
      />
      <ConfigureIntegrationDialog
        integrationId={configureIntegrationId}
        organizationId={organizationId}
        onClose={() => {
          setConfigureIntegrationId(null);
          void refetch();
        }}
      />
      <HomeIntegrationCreateDialog
        open={!!dialogIntegrationName}
        dialogIntegrationName={dialogIntegrationName}
        dialogMode={dialogMode}
        organizationId={organizationId}
        integrationHomeHref={integrationHomeHref}
        dialogDefinition={dialogDefinition}
        defaultDialogName={defaultDialogName}
        existingIntegrationNames={existingIntegrationNames}
        resumePendingForDialog={dialogMode === "resume" ? dialogPendingInstance : undefined}
        initialWebhookSetup={initialWebhookSetup}
        createIntegrationMutation={createIntegrationMutation}
        pendingConnectKeyRef={pendingConnectKeyRef}
        selections={selections}
        onSelectionsChange={onSelectionsChange}
        onClose={() => {
          setDialogIntegrationName(null);
          setDialogMode("resume");
        }}
        onCapabilitySetup={openCapabilitySetup}
        onRefetch={() => void refetch()}
      />
    </>
  );
}

function HomeIntegrationCreateDialog({
  open,
  dialogIntegrationName,
  dialogMode,
  organizationId,
  integrationHomeHref,
  dialogDefinition,
  defaultDialogName,
  existingIntegrationNames,
  resumePendingForDialog,
  initialWebhookSetup,
  createIntegrationMutation,
  pendingConnectKeyRef,
  selections,
  onSelectionsChange,
  onClose,
  onCapabilitySetup,
  onRefetch,
}: {
  open: boolean;
  dialogIntegrationName: string | null;
  dialogMode: "create" | "resume";
  organizationId: string;
  integrationHomeHref?: string;
  dialogDefinition: unknown;
  defaultDialogName: string;
  existingIntegrationNames: Set<string>;
  resumePendingForDialog?: OrganizationsIntegration;
  initialWebhookSetup?: { id: string; webhookUrl: string; config: Record<string, unknown> };
  createIntegrationMutation: {
    mutateAsync: (payload: IntegrationCreatePayload) => Promise<{ data: OrganizationsCreateIntegrationResponse }>;
    reset: () => void;
  };
  pendingConnectKeyRef: MutableRefObject<string | null>;
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  onClose: () => void;
  onCapabilitySetup: (integrationName: string, integrationId?: string) => void;
  onRefetch: () => void;
}) {
  return (
    <IntegrationCreateDialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) {
          onClose();
          createIntegrationMutation.reset();
        }
      }}
      integrationDefinition={(dialogDefinition as never) ?? null}
      organizationId={organizationId}
      integrationHomeHref={integrationHomeHref}
      onCreateIntegration={async (payload) => {
        const res = await createIntegrationMutation.mutateAsync(payload);
        return res.data;
      }}
      onReset={() => createIntegrationMutation.reset()}
      defaultName={
        dialogMode === "create"
          ? resolveDefaultDialogName(dialogIntegrationName, undefined, existingIntegrationNames)
          : defaultDialogName
      }
      onCreated={(integrationId, instanceName) => {
        // Dialog calls onOpenChange(false) before onCreated; keep the key in a ref across that close.
        const key = pendingConnectKeyRef.current;
        pendingConnectKeyRef.current = null;
        if (key) {
          onSelectionsChange({
            ...selections,
            [key]: { id: integrationId, name: instanceName },
          });
        }
        onClose();
        onRefetch();
      }}
      onCapabilitySetupRequired={(integrationName, integrationId) => {
        pendingConnectKeyRef.current = null;
        onClose();
        onCapabilitySetup(integrationName, integrationId);
      }}
      initialBrowserAction={resumePendingForDialog?.status?.browserAction}
      initialCreatedIntegrationId={resumePendingForDialog?.metadata?.id}
      initialWebhookSetup={dialogMode === "create" ? undefined : initialWebhookSetup}
      initialConfiguration={resumePendingForDialog?.spec?.configuration as Record<string, unknown> | undefined}
    />
  );
}

function IntegrationList({
  integrationData,
  variant,
  selections,
  onSelectionsChange,
  onConnect,
  onConfigure,
  onCreateNew,
}: {
  integrationData: IntegrationInstanceSummary[];
  variant: "select" | "status";
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  onConnect: (name: string) => void;
  onConfigure: (id: string) => void;
  onCreateNew: (name: string) => void;
}) {
  return (
    <div className="divide-y divide-slate-200 rounded-md border border-slate-200 dark:divide-gray-700/70 dark:border-gray-700/70">
      {integrationData.map((data) =>
        variant === "status" ? (
          <HomeIntegrationConnectRow
            key={data.name}
            name={data.name}
            status={resolveHomeIntegrationStatus(data)}
            instances={data.allInstances}
            selectedId={selections[data.name]?.id}
            selectedName={selections[data.name]?.name}
            onConnect={() => onConnect(data.name)}
            onConfigure={onConfigure}
            onSelect={(id, instanceName) =>
              onSelectionsChange({ ...selections, [data.name]: { id, name: instanceName } })
            }
            onCreateNew={() => onCreateNew(data.name)}
          />
        ) : (
          <IntegrationRow
            key={data.name}
            data={data}
            selectedId={selections[data.name]?.id}
            onSelect={(id, name) => onSelectionsChange({ ...selections, [data.name]: { id, name } })}
            onConfigure={onConfigure}
            onCreateNew={() => onCreateNew(data.name)}
          />
        ),
      )}
    </div>
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
    <div className="flex min-h-7 items-center gap-2 px-3 py-2.5">
      <IntegrationIcon integrationName={data.name} className="h-4 w-4 shrink-0" size={16} />
      <span className="shrink-0 truncate text-sm font-medium text-slate-900 dark:text-gray-100">{displayName}</span>
      {data.allInstances.length > 0 ? (
        <>
          <Select value={selectedId || ""} onValueChange={handleInstanceSelect}>
            <SelectTrigger className="h-7 min-w-0 flex-1 text-xs">
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
          <Button
            type="button"
            variant="link"
            size="xs"
            onClick={onCreateNew}
            className="h-auto shrink-0 p-0 text-xs font-normal"
          >
            or create new
          </Button>
        </>
      ) : (
        <>
          <span className="min-w-0 flex-1 text-xs font-medium text-gray-400 dark:text-gray-500">Not connected</span>
          <Button type="button" variant="outline" size="xs" className="shrink-0" onClick={onCreateNew}>
            Connect
          </Button>
        </>
      )}
    </div>
  );
}
