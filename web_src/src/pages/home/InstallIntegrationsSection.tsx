import type { OrganizationsCreateIntegrationResponse, OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useMemo, useRef, useState, type MutableRefObject } from "react";
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
  selectionFromInstance,
  type IntegrationInstanceSummary,
  type IntegrationSelections,
} from "./homeIntegrationStatus";
import { useHomeIntegrationConnectActions } from "./useHomeIntegrationConnectActions";
import { useInstallIntegrationSelections, useRefetchOnWindowFocus } from "./useInstallIntegrationSelections";

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
  variant = "status",
}: {
  integrations: string[];
  organizationId: string;
  selections: IntegrationSelections;
  onSelectionsChange: (selections: IntegrationSelections) => void;
  /** `status` shows Connected / Pending rows with switcher; `select` keeps the legacy instance picker. */
  variant?: "select" | "status";
}) {
  const { data: connected = [], refetch } = useConnectedIntegrations(organizationId, {
    enabled: !!organizationId,
  });
  const { data: availableIntegrations = [] } = useAvailableIntegrations({ enabled: !!organizationId });
  const createIntegrationMutation = useCreateIntegration(organizationId, "install_wizard");
  const [dialogIntegrationName, setDialogIntegrationName] = useState<string | null>(null);
  /** "create" skips resuming a pending instance so "Connect new" always starts fresh. */
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
  const { rememberPreferredInstance } = useInstallIntegrationSelections({
    integrationData,
    selections,
    onSelectionsChange,
  });
  useRefetchOnWindowFocus(refetch);

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

  const openCapabilitySetupAndPrefer = (integrationName: string, integrationId?: string) => {
    if (integrationId) rememberPreferredInstance(integrationName, integrationId);
    openCapabilitySetup(integrationName, integrationId);
    void refetch();
  };

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
        onPreferInstance={rememberPreferredInstance}
        onClose={() => {
          setDialogIntegrationName(null);
          setDialogMode("resume");
        }}
        onCapabilitySetup={openCapabilitySetupAndPrefer}
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
  onPreferInstance,
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
  onPreferInstance: (integrationName: string, integrationId: string) => void;
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
          onPreferInstance(key, integrationId);
          // Newly created instances are not ready until setup finishes.
          onSelectionsChange({
            ...selections,
            [key]: { id: integrationId, name: instanceName, ready: false },
          });
        }
        onClose();
        onRefetch();
      }}
      onCapabilitySetupRequired={(integrationName, integrationId) => {
        pendingConnectKeyRef.current = null;
        onPreferInstance(integrationName, integrationId);
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
    <div className="divide-y divide-edge-subtle rounded-md border border-edge-default">
      {integrationData.map((data) =>
        variant === "status" ? (
          <HomeIntegrationConnectRow
            key={data.name}
            name={data.name}
            status={resolveHomeIntegrationStatus(data, selections[data.name]?.id)}
            instances={data.allInstances}
            selectedId={selections[data.name]?.id}
            selectedName={selections[data.name]?.name}
            onConnect={() => onConnect(data.name)}
            onConfigure={onConfigure}
            onSelect={(id, instanceName) => {
              const instance = data.allInstances.find((item) => item.metadata?.id === id);
              const selection = instance ? selectionFromInstance(instance) : null;
              onSelectionsChange({
                ...selections,
                [data.name]: selection ?? { id, name: instanceName, ready: false },
              });
            }}
            onCreateNew={() => onCreateNew(data.name)}
          />
        ) : (
          <IntegrationRow
            key={data.name}
            data={data}
            selectedId={selections[data.name]?.id}
            onSelect={(id, name) => {
              const instance = data.allInstances.find((item) => item.metadata?.id === id);
              const selection = instance ? selectionFromInstance(instance) : null;
              onSelectionsChange({
                ...selections,
                [data.name]: selection ?? { id, name, ready: false },
              });
            }}
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
      <span className="shrink-0 truncate text-sm font-medium text-content-primary">{displayName}</span>
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
          <span className="min-w-0 flex-1 text-xs font-medium text-content-muted">Not connected</span>
          <Button type="button" variant="outline" size="xs" className="shrink-0" onClick={onCreateNew}>
            Connect
          </Button>
        </>
      )}
    </div>
  );
}
