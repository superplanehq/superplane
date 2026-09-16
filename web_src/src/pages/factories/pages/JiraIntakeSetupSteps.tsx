import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Check, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";

export function JiraConnectionStep({
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
  if (loading) return <LoadingMessage message="Loading Jira connections…" />;
  return (
    <div className="space-y-4">
      <p className="workspace-body-text text-muted-foreground">
        SuperPlane connects to Jira with an OAuth 2.0 app. Create the app in the Atlassian Developer Console, then paste
        the Client ID and Client Secret here.
      </p>
      {integrations.length > 0 ? (
        <ConnectionOptions integrations={integrations} selectedId={selectedId} onSelect={onSelect} />
      ) : (
        <JiraConnectionInstructions />
      )}
      <Button type="button" variant={integrations.length > 0 ? "outline" : "default"} onClick={onConnect}>
        {integrations.length > 0 ? "Connect another site" : "Connect Jira"}
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
          <Button
            key={id}
            type="button"
            variant="ghost"
            onClick={() => onSelect(id)}
            className={cn(
              "h-auto w-full justify-between rounded-lg border px-3 py-3 text-left",
              selected ? "border-foreground bg-accent/40 hover:bg-accent/40" : "border-border hover:bg-accent/30",
            )}
          >
            <span className="text-[13px] font-medium">{integration.metadata?.name || "Jira"}</span>
            {selected ? <Check className="size-4" aria-hidden /> : null}
          </Button>
        );
      })}
    </div>
  );
}

function JiraConnectionInstructions() {
  return (
    <div className="rounded-lg border border-border bg-muted/30 p-4">
      <p className="text-[13px] font-medium">Connect your Jira Cloud site</p>
      <ol className="workspace-body-text mt-2 list-decimal space-y-1 pl-5 text-muted-foreground">
        <li>Create an OAuth 2.0 (3LO) app in the Atlassian Developer Console.</li>
        <li>Add Jira API scopes, including manage:jira-webhook.</li>
        <li>Paste the callback URL from SuperPlane into the app settings.</li>
      </ol>
    </div>
  );
}

export function JiraProjectStep({
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
  if (loading) return <LoadingMessage message="Loading Jira projects…" />;
  if (error) return <RetryMessage message="SuperPlane could not load Jira projects." onRetry={onRetry} />;
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
      <Button
        type="button"
        variant="ghost"
        role="option"
        aria-selected={selected}
        onClick={() => onSelect(id)}
        data-testid={`jira-project-${id}`}
        className={cn(
          "h-auto w-full justify-start gap-3 rounded-none px-3 py-2.5 text-left",
          selected ? "bg-accent/50 hover:bg-accent/50" : "hover:bg-accent/30",
        )}
      >
        <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{project.name || "Untitled project"}</span>
        {selected ? <Check className="size-3.5 shrink-0 text-foreground" strokeWidth={2.5} aria-hidden /> : null}
      </Button>
    </li>
  );
}

export function JiraCompleteStep() {
  return (
    <div className="flex flex-col items-center py-6 text-center">
      <span className="flex size-10 items-center justify-center rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300">
        <Check className="size-5" aria-hidden />
      </span>
      <p className="mt-3 text-[15px] font-semibold">Jira intake is ready</p>
      <p className="workspace-body-text mt-1 text-muted-foreground">
        SuperPlane is adding the newest open issues to the Backlog. New and updated issues arrive through Jira webhooks.
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
