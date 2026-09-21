import type { FactoryIntakeHealth, OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Check, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";

import {
  INTAKE_CONNECTION_COPY,
  intakeChooseConnectionCopy,
  intakeConnectLabel,
  intakeHealthBanner,
  intakeProviderDisplayName,
  intakeReconnectLabel,
  selectedIntakeIntegration,
  type IntakeConnectionBinding,
} from "./intakeConnectionModel";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export type IntakeConnectionFieldsProps = {
  sourceId: LineIntakeSourceId;
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
  const banner = intakeHealthBanner(health);
  const selected = selectedIntakeIntegration(integrations, binding.integrationId);
  const selectedReady = selected?.status?.state === "ready";
  const reconnectLabel = intakeReconnectLabel(selected);
  const showReconnect = Boolean(selected && !selectedReady && onReconnect);

  return (
    <section className="flex flex-col gap-4" data-testid="intake-connection">
      <div className="flex items-start justify-between gap-3">
        <h3 className="workspace-section-title">{INTAKE_CONNECTION_COPY.section}</h3>
        <Button
          type="button"
          variant={integrations.length > 0 ? "outline" : "default"}
          size="sm"
          disabled={connecting}
          onClick={onConnect}
          data-testid="intake-connection-connect"
        >
          {connecting ? INTAKE_CONNECTION_COPY.connecting : intakeConnectLabel(integrations.length > 0, providerName)}
        </Button>
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
      {binding.integrationId ? (
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

function ConnectionList({
  integrations,
  selectedId,
  loading,
  chooseCopy,
  onSelect,
}: {
  integrations: OrganizationsIntegration[];
  selectedId: string;
  loading: boolean;
  chooseCopy: string;
  onSelect: (id: string) => void;
}) {
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
              "h-auto w-full justify-between rounded-lg border px-3 py-3 text-left text-[13px] font-medium",
              selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30",
            )}
          >
            <span className="min-w-0 flex-1 truncate">{integration.metadata?.name || "Connection"}</span>
            {selected ? <Check className="size-4 shrink-0" aria-hidden /> : null}
          </Button>
        );
      })}
    </div>
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
  const [query, setQuery] = useState("");
  const filtered = useMemo(() => {
    const term = query.trim().toLowerCase();
    if (!term) return projects;
    return projects.filter((project) => (project.name ?? project.id ?? "").toLowerCase().includes(term));
  }, [projects, query]);

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
    <div className="space-y-3">
      <p className="text-[13px] font-medium">{INTAKE_CONNECTION_COPY.project}</p>
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search projects"
          className="h-9 pl-9"
          aria-label="Search projects"
        />
      </div>
      <div className="space-y-2">
        {filtered.map((project) => {
          const id = project.id ?? "";
          const selected = id === selectedId;
          return (
            <Button
              key={id}
              type="button"
              variant="ghost"
              onClick={() => onSelect(id)}
              data-testid={`intake-connection-project-${id}`}
              className={cn(
                "h-auto w-full justify-between rounded-lg border px-3 py-3 text-left text-[13px] font-medium",
                selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30",
              )}
            >
              <span className="min-w-0 flex-1 truncate">{project.name || id}</span>
              {selected ? <Check className="size-4 shrink-0" aria-hidden /> : null}
            </Button>
          );
        })}
      </div>
    </div>
  );
}
