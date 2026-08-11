import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { cn } from "@/lib/utils";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client";
import { settingsPanelClassName } from "./settingsPageStyles";

export interface IntegrationCatalogEntry {
  providerName: string;
  providerLabel: string;
  integrationDef: IntegrationsIntegrationDefinition | null;
  instances: OrganizationsIntegration[];
}

interface IntegrationCatalogRowProps {
  entry: IntegrationCatalogEntry;
  isActive: boolean;
  canCreateIntegrations: boolean;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  onConnect: (integration: IntegrationsIntegrationDefinition) => void;
  onConfigure: (integration: OrganizationsIntegration) => void;
  rowRef: (node: HTMLLIElement | null) => void;
}

interface StatusTone {
  badge: string;
  dot: string;
}

const STATUS_TONES: Record<"ready" | "error" | "pending", StatusTone> = {
  ready: {
    badge:
      "border-green-200 bg-green-50 text-green-700 dark:border-green-500/20 dark:bg-green-500/10 dark:text-green-300",
    dot: "bg-green-500",
  },
  error: {
    badge: "border-red-200 bg-red-50 text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-300",
    dot: "bg-red-500",
  },
  pending: {
    badge:
      "border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-300",
    dot: "bg-amber-500",
  },
};

function statusTone(state: string | undefined): StatusTone {
  if (state === "ready") return STATUS_TONES.ready;
  if (state === "error") return STATUS_TONES.error;
  return STATUS_TONES.pending;
}

function ConnectedInstanceRow({
  integration,
  isFirst,
  canUpdateIntegrations,
  permissionsLoading,
  onConfigure,
}: {
  integration: OrganizationsIntegration;
  isFirst: boolean;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  onConfigure: (integration: OrganizationsIntegration) => void;
}) {
  const state = integration.status?.state;
  const statusLabel = state ? state.charAt(0).toUpperCase() + state.slice(1) : "Unknown";
  const tone = statusTone(state);

  return (
    <div
      className={cn("flex items-center gap-3 py-2 border-t border-gray-200 dark:border-gray-700/70", isFirst && "mt-1")}
    >
      <p className="min-w-0 truncate text-sm font-medium text-gray-800 dark:text-gray-100">
        {integration.metadata?.name}
      </p>
      <Badge variant="secondary" className={cn("gap-1.5", tone.badge)}>
        <span className={cn("size-1.5 rounded-full", tone.dot)} aria-hidden="true" />
        {statusLabel}
      </Badge>
      <div className="ml-auto flex items-center gap-4">
        <PermissionTooltip
          allowed={canUpdateIntegrations || permissionsLoading}
          message="You don't have permission to update integrations."
        >
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              if (!canUpdateIntegrations) return;
              onConfigure(integration);
            }}
            disabled={!canUpdateIntegrations}
          >
            Configure
          </Button>
        </PermissionTooltip>
      </div>
    </div>
  );
}

/**
 * A single provider in the integrations catalog. `isActive` reflects the
 * keyboard highlight driven by the filter input, not focus.
 */
export function IntegrationCatalogRow({
  entry,
  isActive,
  canCreateIntegrations,
  canUpdateIntegrations,
  permissionsLoading,
  onConnect,
  onConfigure,
  rowRef,
}: IntegrationCatalogRowProps) {
  const connectedCount = entry.instances.length;

  return (
    <li
      ref={rowRef}
      data-integration-row={entry.providerName}
      aria-current={isActive ? "true" : undefined}
      className={cn(settingsPanelClassName, "scroll-mt-4", isActive && "ring-2 ring-blue-500/70 dark:ring-blue-400/70")}
    >
      <div className="p-4 flex items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <div className="mt-0.5 flex h-8 w-8 items-center justify-center">
            <IntegrationIcon
              integrationName={entry.providerName}
              iconSlug={entry.integrationDef?.icon}
              className="w-8 h-8 text-gray-500 dark:text-gray-400"
            />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-gray-800 dark:text-gray-100">{entry.providerLabel}</h3>
            {entry.integrationDef?.description ? (
              <p className="mt-0.5 text-sm text-gray-800 dark:text-gray-400">{entry.integrationDef.description}</p>
            ) : null}
          </div>
        </div>
        <PermissionTooltip
          allowed={Boolean(entry.integrationDef) && (canCreateIntegrations || permissionsLoading)}
          message={
            entry.integrationDef
              ? "You don't have permission to connect integrations."
              : "This integration provider is no longer available for new connections."
          }
        >
          <Button
            variant="default"
            size="sm"
            onClick={() => {
              if (!entry.integrationDef) return;
              onConnect(entry.integrationDef);
            }}
            className="self-start"
            disabled={!entry.integrationDef || !canCreateIntegrations}
          >
            {entry.integrationDef ? "Connect" : "Unavailable"}
          </Button>
        </PermissionTooltip>
      </div>
      {connectedCount > 0 ? (
        <div className="pr-4 pb-4 pl-[60px]">
          <p className="mb-2 text-xs text-gray-500 dark:text-gray-400">
            {connectedCount} connected instance{connectedCount === 1 ? "" : "s"}
          </p>
          {entry.instances.map((integration, index) => (
            <ConnectedInstanceRow
              key={integration.metadata?.id}
              integration={integration}
              isFirst={index === 0}
              canUpdateIntegrations={canUpdateIntegrations}
              permissionsLoading={permissionsLoading}
              onConfigure={onConfigure}
            />
          ))}
        </div>
      ) : null}
    </li>
  );
}
