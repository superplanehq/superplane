import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Check, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";

export function NotionConnectionStep({
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
  if (loading) return <LoadingMessage message="Loading Notion connections…" />;
  return (
    <div className="space-y-4">
      <p className="workspace-body-text text-muted-foreground">
        SuperPlane uses an internal integration token to read the pages of a shared database.
      </p>
      {integrations.length > 0 ? (
        <ConnectionOptions integrations={integrations} selectedId={selectedId} onSelect={onSelect} />
      ) : (
        <NotionConnectionInstructions />
      )}
      <Button type="button" variant={integrations.length > 0 ? "outline" : "default"} onClick={onConnect}>
        {integrations.length > 0 ? "Connect another account" : "Connect Notion"}
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
            <span className="text-[13px] font-medium">{integration.metadata?.name || "Notion"}</span>
            {selected ? <Check className="size-4" aria-hidden /> : null}
          </button>
        );
      })}
    </div>
  );
}

function NotionConnectionInstructions() {
  return (
    <div className="rounded-lg border border-border bg-muted/30 p-4">
      <p className="text-[13px] font-medium">Connect your Notion workspace</p>
      <ol className="workspace-body-text mt-2 list-decimal space-y-1 pl-5 text-muted-foreground">
        <li>Create an internal integration in Notion's My integrations page.</li>
        <li>Copy the integration's internal integration secret.</li>
        <li>Enter the token in the next form.</li>
      </ol>
    </div>
  );
}

export function NotionDatabaseStep({
  databases,
  selectedId,
  loading,
  error,
  onSelect,
  onRetry,
}: {
  databases: Array<{ id?: string; name?: string }>;
  selectedId: string;
  loading: boolean;
  error: boolean;
  onSelect: (id: string) => void;
  onRetry: () => void;
}) {
  if (loading) return <LoadingMessage message="Loading Notion databases…" />;
  if (error) return <RetryMessage message="SuperPlane could not load Notion databases." onRetry={onRetry} />;
  if (databases.length === 0) {
    return (
      <p className="workspace-body-text text-muted-foreground">
        This connection has no shared databases. Share a database with the integration in Notion, then try again.
      </p>
    );
  }
  return <DatabasePicker databases={databases} selectedId={selectedId} onSelect={onSelect} />;
}

function DatabasePicker({
  databases,
  selectedId,
  onSelect,
}: {
  databases: Array<{ id?: string; name?: string }>;
  selectedId: string;
  onSelect: (id: string) => void;
}) {
  const [query, setQuery] = useState("");
  const filtered = useMemo(() => {
    const term = query.trim().toLowerCase();
    if (!term) return databases;
    return databases.filter((database) => (database.name ?? "").toLowerCase().includes(term));
  }, [databases, query]);

  return (
    <div className="space-y-3">
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search databases"
          className="h-9 pl-9"
          aria-label="Search databases"
        />
      </div>
      <div className="max-h-56 overflow-y-auto rounded-lg border border-border" role="listbox" aria-label="Databases">
        {filtered.length === 0 ? (
          <p className="px-3 py-6 text-center text-[13px] text-muted-foreground">No matching databases.</p>
        ) : (
          <ul className="divide-y divide-border">
            {filtered.map((database) => (
              <DatabaseOption
                key={database.id}
                database={database}
                selected={database.id === selectedId}
                onSelect={onSelect}
              />
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function DatabaseOption({
  database,
  selected,
  onSelect,
}: {
  database: { id?: string; name?: string };
  selected: boolean;
  onSelect: (id: string) => void;
}) {
  const id = database.id ?? "";
  return (
    <li>
      <button
        type="button"
        role="option"
        aria-selected={selected}
        onClick={() => onSelect(id)}
        data-testid={`notion-database-${id}`}
        className={cn(
          "flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors",
          selected ? "bg-accent/50" : "hover:bg-accent/30",
        )}
      >
        <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{database.name || "Untitled database"}</span>
        {selected ? <Check className="size-3.5 shrink-0 text-foreground" strokeWidth={2.5} aria-hidden /> : null}
      </button>
    </li>
  );
}

export function NotionCompleteStep() {
  return (
    <div className="flex flex-col items-center py-6 text-center">
      <span className="flex size-10 items-center justify-center rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300">
        <Check className="size-5" aria-hidden />
      </span>
      <p className="mt-3 text-[15px] font-semibold">Notion intake is ready</p>
      <p className="workspace-body-text mt-1 text-muted-foreground">
        SuperPlane is adding the newest pages of the database to the Backlog. SuperPlane checks the database every
        minute, so later pages arrive shortly after your team adds them.
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
