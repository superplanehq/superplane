import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Check, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";

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
  if (loading) {
    return (
      <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        Loading Jira projects...
      </p>
    );
  }
  if (error) {
    return (
      <div className="space-y-3">
        <p className="workspace-body-text text-destructive">SuperPlane could not load Jira projects.</p>
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          Try again
        </Button>
      </div>
    );
  }
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
