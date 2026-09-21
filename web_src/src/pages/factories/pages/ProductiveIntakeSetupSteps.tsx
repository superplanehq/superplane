import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Check, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";

import { PRODUCTIVE_INTAKE_SETUP_COPY } from "./productiveIntakeSetupCopy";

export function ProductiveConnectionStep({
  integrations,
  selectedId,
  loading,
  onSelect,
}: {
  integrations: OrganizationsIntegration[];
  selectedId: string;
  loading: boolean;
  onSelect: (id: string) => void;
}) {
  if (loading) return <LoadingMessage message={PRODUCTIVE_INTAKE_SETUP_COPY.wizardConnectionsLoading} />;
  if (integrations.length === 0) {
    return null;
  }

  return <ConnectionOptions integrations={integrations} selectedId={selectedId} onSelect={onSelect} />;
}

function ConnectionOptions({
  integrations,
  selectedId,
  onSelect,
}: {
  integrations: OrganizationsIntegration[];
  selectedId: string;
  onSelect: (id: string) => void;
}) {
  return (
    <div className="space-y-2">
      <p className="text-[13px] font-medium">{PRODUCTIVE_INTAKE_SETUP_COPY.wizardStepConnectExisting}</p>
      {integrations.map((integration) => {
        const id = integration.metadata?.id ?? "";
        const selected = id === selectedId;
        return (
          <Button
            key={id}
            type="button"
            variant="ghost"
            onClick={() => onSelect(id)}
            data-testid={`productive-connection-${id}`}
            className={cn(
              "h-auto w-full justify-between rounded-lg border px-3 py-3 text-left text-[13px] font-medium",
              selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30",
            )}
          >
            <span className="min-w-0 flex-1 truncate">{integration.metadata?.name || "Productive.io"}</span>
            {selected ? <Check className="size-4 shrink-0" aria-hidden /> : null}
          </Button>
        );
      })}
    </div>
  );
}

export function ProductiveProjectStep({
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
  onRetry: () => void;
}) {
  if (loading) return <LoadingMessage message={PRODUCTIVE_INTAKE_SETUP_COPY.wizardProjectsLoading} />;
  if (error) return <RetryMessage message={PRODUCTIVE_INTAKE_SETUP_COPY.wizardProjectsError} onRetry={onRetry} />;
  if (projects.length === 0) {
    return (
      <p className="workspace-body-text text-muted-foreground">{PRODUCTIVE_INTAKE_SETUP_COPY.wizardProjectsEmpty}</p>
    );
  }
  return <ProjectPicker projects={projects} selectedId={selectedId} onSelect={onSelect} />;
}

function ProjectPicker({
  projects,
  selectedId,
  onSelect,
}: {
  projects: Array<{ id?: string; name?: string }>;
  selectedId: string;
  onSelect: (id: string) => void;
}) {
  const [query, setQuery] = useState("");
  const filtered = useMemo(() => {
    const term = query.trim().toLowerCase();
    if (!term) return projects;
    return projects.filter((project) => (project.name ?? "").toLowerCase().includes(term));
  }, [projects, query]);

  return (
    <div className="space-y-3">
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
      <div className="max-h-56 overflow-y-auto rounded-lg border border-border" role="listbox" aria-label="Projects">
        {filtered.length === 0 ? (
          <p className="px-3 py-6 text-center text-[13px] text-muted-foreground">No matching projects.</p>
        ) : (
          <ul className="divide-y divide-border">
            {filtered.map((project) => (
              <ProjectOption
                key={project.id}
                project={project}
                selected={project.id === selectedId}
                onSelect={onSelect}
              />
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function ProjectOption({
  project,
  selected,
  onSelect,
}: {
  project: { id?: string; name?: string };
  selected: boolean;
  onSelect: (id: string) => void;
}) {
  const id = project.id ?? "";
  return (
    <li>
      <Button
        type="button"
        variant="ghost"
        role="option"
        aria-selected={selected}
        onClick={() => onSelect(id)}
        data-testid={`productive-project-${id}`}
        className={cn(
          "h-auto w-full justify-start gap-3 rounded-none px-3 py-2.5 text-left text-[13px] font-medium",
          selected ? "bg-accent/50" : "hover:bg-accent/30",
        )}
      >
        <span className="min-w-0 flex-1 truncate">{project.name || "Untitled project"}</span>
        {selected ? <Check className="size-3.5 shrink-0 text-foreground" strokeWidth={2.5} aria-hidden /> : null}
      </Button>
    </li>
  );
}

function LoadingMessage({ message }: { message: string }) {
  return (
    <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
      <Loader2 className="size-4 animate-spin" aria-hidden />
      {message}
    </p>
  );
}

function RetryMessage({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="space-y-3">
      <p className="workspace-body-text text-destructive">{message}</p>
      <Button type="button" variant="outline" size="sm" onClick={onRetry}>
        {PRODUCTIVE_INTAKE_SETUP_COPY.wizardRetry}
      </Button>
    </div>
  );
}
