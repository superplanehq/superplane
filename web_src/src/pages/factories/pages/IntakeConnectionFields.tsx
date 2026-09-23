import type { FactoryIntakeHealth, OrganizationsIntegration } from "@/api-client";
import { AutoCompleteSelect } from "@/components/AutoCompleteSelect/AutoCompleteSelect";
import { Link } from "@/components/Link/link";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { integrationDetailPath, useIntegrationsBasePath } from "@/lib/integrationSettingsPaths";
import { cn } from "@/lib/utils";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { Check, ChevronRight, Loader2 } from "lucide-react";
import { useEffect, useMemo, type ReactNode } from "react";

import {
  INTAKE_CONNECTION_COPY,
  intakeChooseConnectionCopy,
  intakeConnectLabel,
  intakeHealthBanner,
  intakeIntegrationInstanceName,
  intakeProviderDisplayName,
  intakeReconnectLabel,
  selectedIntakeIntegration,
  showIntakeConnectAction,
  type IntakeConnectionBinding,
} from "./intakeConnectionModel";
import { intakeSettingsSectionDomId } from "./intakeSourceSettingsModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export type IntakeConnectionFieldsProps = {
  sourceId: LineIntakeSourceId;
  organizationId?: string;
  /** Factory settings integrations base path. Prefer this over the legacy org settings path. */
  integrationsBasePath?: string;
  health?: FactoryIntakeHealth;
  binding: IntakeConnectionBinding;
  integrations: OrganizationsIntegration[];
  integrationsLoading?: boolean;
  projects: Array<{ id?: string; name?: string }>;
  projectsLoading?: boolean;
  projectsError?: boolean;
  connecting?: boolean;
  connectError?: string;
  onBindingChange: (next: IntakeConnectionBinding) => void;
  onConnect: () => void;
  onReconnect?: () => void;
  onRetryProjects?: () => void;
};

export function IntakeConnectionFields({
  sourceId,
  organizationId,
  integrationsBasePath,
  health,
  binding,
  integrations,
  integrationsLoading = false,
  projects,
  projectsLoading = false,
  projectsError = false,
  connecting = false,
  connectError,
  onBindingChange,
  onConnect,
  onReconnect,
  onRetryProjects,
}: IntakeConnectionFieldsProps) {
  const providerName = intakeProviderDisplayName(sourceId);
  const selected = selectedIntakeIntegration(integrations, binding.integrationId);
  const healthBanner = intakeHealthBanner(health);
  const missingSavedConnection = Boolean(binding.integrationId) && !integrationsLoading && !selected;
  const banner = healthBanner ?? (missingSavedConnection ? INTAKE_CONNECTION_COPY.missing : undefined);
  const selectedReady = selected?.status?.state === "ready";
  const reconnectLabel = intakeReconnectLabel(selected);
  const showReconnect = Boolean(selected && !selectedReady && onReconnect);
  const showConnect = showIntakeConnectAction(integrations.length, integrationsLoading);

  return (
    <section
      id={intakeSettingsSectionDomId("connection")}
      className="scroll-mt-6 flex flex-col gap-4"
      data-testid="intake-connection"
    >
      <div className="flex items-start justify-between gap-3">
        <h3 className="workspace-section-title">{INTAKE_CONNECTION_COPY.section}</h3>
        {showConnect ? (
          <Button
            type="button"
            variant="default"
            size="sm"
            disabled={connecting}
            onClick={onConnect}
            data-testid="intake-connection-connect"
          >
            {connecting ? INTAKE_CONNECTION_COPY.connecting : intakeConnectLabel(providerName)}
          </Button>
        ) : null}
      </div>
      {banner ? (
        <p
          className="workspace-body-text text-amber-700 dark:text-amber-400"
          role="status"
          data-testid="intake-connection-banner"
        >
          {banner}
        </p>
      ) : null}
      {connectError ? (
        <p className="workspace-body-text text-destructive" role="alert" data-testid="intake-connection-error">
          {connectError}
        </p>
      ) : null}
      <ConnectionList
        organizationId={organizationId}
        integrationsBasePath={integrationsBasePath}
        integrations={integrations}
        selectedId={binding.integrationId}
        loading={integrationsLoading}
        chooseCopy={intakeChooseConnectionCopy(sourceId)}
        onSelect={(integrationId) => onBindingChange({ integrationId, resourceId: "" })}
      />
      {showReconnect ? (
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onReconnect}
          data-testid="intake-connection-reconnect"
        >
          {reconnectLabel}
        </Button>
      ) : null}
      {selected ? (
        <ProjectList
          projects={projects}
          selectedId={binding.resourceId}
          loading={projectsLoading}
          error={projectsError}
          onSelect={(resourceId) => onBindingChange({ ...binding, resourceId })}
          onRetry={onRetryProjects}
        />
      ) : null}
    </section>
  );
}

const integrationRowClassName =
  "flex h-auto w-full min-w-0 items-center justify-start gap-3 whitespace-normal rounded-lg border px-3 py-2.5 text-left text-foreground no-underline transition-colors";

function ConnectionList({
  organizationId,
  integrationsBasePath: integrationsBasePathProp,
  integrations,
  selectedId,
  loading,
  chooseCopy,
  onSelect,
}: {
  organizationId?: string;
  integrationsBasePath?: string;
  integrations: OrganizationsIntegration[];
  selectedId: string;
  loading: boolean;
  chooseCopy: string;
  onSelect: (id: string) => void;
}) {
  const integrationsBasePathFallback = useIntegrationsBasePath(organizationId ?? "");
  const integrationsBasePath = integrationsBasePathProp || integrationsBasePathFallback;
  const singleIntegration = integrations.length === 1 ? integrations[0] : undefined;
  const singleIntegrationId = singleIntegration?.metadata?.id ?? "";

  useEffect(() => {
    if (loading || selectedId || !singleIntegrationId) {
      return;
    }
    onSelect(singleIntegrationId);
  }, [loading, onSelect, selectedId, singleIntegrationId]);

  if (loading) {
    return (
      <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        {INTAKE_CONNECTION_COPY.loadingConnections}
      </p>
    );
  }
  if (integrations.length === 0) {
    return null;
  }

  const selectedMatchesOnlyConnection = selectedId === singleIntegrationId;
  const showCompactConnection = Boolean(singleIntegration) && (!selectedId || selectedMatchesOnlyConnection);

  if (showCompactConnection && singleIntegration) {
    const settingsHref = singleIntegrationId
      ? integrationDetailPath(integrationsBasePath, singleIntegrationId)
      : undefined;
    const identity = <IntakeIntegrationIdentity integration={singleIntegration} />;

    if (settingsHref) {
      return (
        <Link
          href={settingsHref}
          className={cn(integrationRowClassName, "group border-border hover:bg-accent/30")}
          data-testid={`intake-connection-${singleIntegrationId}`}
        >
          {identity}
          <ChevronRight className="size-4 shrink-0 text-muted-foreground group-hover:text-foreground" aria-hidden />
        </Link>
      );
    }

    return (
      <div
        className={cn(integrationRowClassName, "border-border")}
        data-testid={`intake-connection-${singleIntegrationId}`}
      >
        {identity}
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <p className="text-[13px] font-medium">{chooseCopy}</p>
      {integrations.map((integration) => {
        const id = integration.metadata?.id ?? "";
        const selected = id === selectedId;
        return (
          <Button
            key={id}
            type="button"
            variant="ghost"
            onClick={() => onSelect(id)}
            data-testid={`intake-connection-${id}`}
            className={cn(
              integrationRowClassName,
              selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30",
            )}
          >
            <IntakeIntegrationIdentity integration={integration} />
            {selected ? <Check className="size-4 shrink-0" aria-hidden /> : null}
          </Button>
        );
      })}
    </div>
  );
}

function IntakeIntegrationIdentity({ integration }: { integration: OrganizationsIntegration }): ReactNode {
  const name = intakeIntegrationInstanceName(integration);
  return (
    <>
      <span className="flex size-8 shrink-0 items-center justify-center rounded-md border border-border bg-muted/40">
        <IntegrationIcon integrationName={integration.metadata?.integrationName} className="size-4" size={16} />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-[13px] font-medium text-foreground">{name}</span>
        <span className="block text-[12px] text-muted-foreground">{INTAKE_CONNECTION_COPY.integration}</span>
      </span>
    </>
  );
}

function ProjectList({
  projects,
  selectedId,
  loading,
  error,
  onSelect,
  onRetry,
}: {
  projects: Array<{ id?: string; name?: string }>;
  selectedId: string;
  loading: boolean;
  error: boolean;
  onSelect: (id: string) => void;
  onRetry?: () => void;
}) {
  const options = useMemo(
    () =>
      projects
        .map((project) => {
          const id = project.id ?? "";
          const name = project.name?.trim() || id;
          return { value: id, label: name };
        })
        .filter((option) => option.value.length > 0),
    [projects],
  );

  if (loading) {
    return (
      <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        {INTAKE_CONNECTION_COPY.loadingProjects}
      </p>
    );
  }
  if (error) {
    return (
      <div className="space-y-2">
        <p className="workspace-body-text text-destructive">{INTAKE_CONNECTION_COPY.projectsError}</p>
        {onRetry ? (
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            {INTAKE_CONNECTION_COPY.retry}
          </Button>
        ) : null}
      </div>
    );
  }
  if (projects.length === 0) {
    return <p className="workspace-body-text text-muted-foreground">{INTAKE_CONNECTION_COPY.projectsEmpty}</p>;
  }

  return (
    <div className="space-y-2" data-testid="intake-connection-project-select">
      <Label htmlFor="intake-connection-project">{INTAKE_CONNECTION_COPY.project}</Label>
      <AutoCompleteSelect
        id="intake-connection-project"
        options={options}
        value={selectedId}
        onChange={onSelect}
        placeholder="Search projects"
      />
    </div>
  );
}
