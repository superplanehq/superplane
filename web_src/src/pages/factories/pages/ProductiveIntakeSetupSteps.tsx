import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Check, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";

export function ProductiveConnectionStep({
  integrations,
  selectedId,
  loading,
  onSelect,
  onConnect,
}: {
  integrations: OrganizationsIntegration[];
  selectedId: string;
  loading: boolean;
  onSelect: (id: string) => void;
  onConnect: () => void;
}) {
  if (loading) return <LoadingMessage message="Loading Productive.io connections…" />;
  return (
    <div className="space-y-4">
      <p className="workspace-body-text text-muted-foreground">
        SuperPlane uses a personal API token to read the tasks of your projects.
      </p>
      {integrations.length > 0 ? (
        <ConnectionOptions integrations={integrations} selectedId={selectedId} onSelect={onSelect} />
      ) : (
        <ProductiveConnectionInstructions />
      )}
      <Button type="button" variant={integrations.length > 0 ? "outline" : "default"} onClick={onConnect}>
        {integrations.length > 0 ? "Connect another account" : "Connect Productive.io"}
      </Button>
    </div>
  );
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
      <p className="text-[13px] font-medium">Choose a connection</p>
      {integrations.map((integration) => {
        const id = integration.metadata?.id ?? "";
        const selected = id === selectedId;
        return (
          <button
            key={id}
            type="button"
            onClick={() => onSelect(id)}
            className={`flex w-full items-center justify-between rounded-lg border px-3 py-3 text-left ${
              selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30"
            }`}
          >
            <span className="text-[13px] font-medium">{integration.metadata?.name || "Productive.io"}</span>
            {selected ? <Check className="size-4" aria-hidden /> : null}
          </button>
        );
      })}
    </div>
  );
}

function ProductiveConnectionInstructions() {
  return (
    <div className="rounded-lg border border-border bg-muted/30 p-4">
      <p className="text-[13px] font-medium">Connect your Productive.io account</p>
      <ol className="workspace-body-text mt-2 list-decimal space-y-1 pl-5 text-muted-foreground">
        <li>Create a personal access token in Productive.io API integrations.</li>
        <li>Copy the organization ID from your Productive.io URL.</li>
        <li>Enter both values in the next form.</li>
      </ol>
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
  if (loading) return <LoadingMessage message="Loading Productive.io projects…" />;
  if (error) return <RetryMessage message="SuperPlane could not load Productive.io projects." onRetry={onRetry} />;
  if (projects.length === 0) {
    return <p className="workspace-body-text text-muted-foreground">This connection has no available projects.</p>;
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
      <button
        type="button"
        role="option"
        aria-selected={selected}
        onClick={() => onSelect(id)}
        data-testid={`productive-project-${id}`}
        className={cn(
          "flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors",
          selected ? "bg-accent/50" : "hover:bg-accent/30",
        )}
      >
        <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{project.name || "Untitled project"}</span>
        {selected ? <Check className="size-3.5 shrink-0 text-foreground" strokeWidth={2.5} aria-hidden /> : null}
      </button>
    </li>
  );
}

export function ProductiveCompleteStep() {
  return (
    <div className="flex flex-col items-center py-6 text-center">
      <span className="flex size-10 items-center justify-center rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300">
        <Check className="size-5" aria-hidden />
      </span>
      <p className="mt-3 text-[15px] font-semibold">Productive.io intake is ready</p>
      <p className="workspace-body-text mt-1 text-muted-foreground">
        SuperPlane is adding the newest open tasks to the Backlog. SuperPlane checks the project every minute, so later
        tasks arrive shortly after your team creates them.
      </p>
    </div>
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
        Try again
      </Button>
    </div>
  );
}
