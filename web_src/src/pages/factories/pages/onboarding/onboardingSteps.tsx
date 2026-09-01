import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import type { IntegrationInstanceSummary } from "@/pages/home/homeIntegrationStatus";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { Check, Plus, Search } from "lucide-react";
import { useEffect, useMemo, useState, type ReactNode } from "react";

import { FIXTURE_REPOS, VCS_OPTIONS, vcsLabel, type IntegrationId, type VcsHostId } from "./onboardingFixtures";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

export function IntegrationChoiceIcon({ name, size = 20 }: { name: IntegrationId | VcsHostId; size?: number }) {
  return <IntegrationIcon integrationName={name} className={size <= 16 ? "size-3.5" : "size-5"} size={size} />;
}

function ComingSoonRibbon() {
  // Nested clip box so the diagonal band can extend past the corner while the
  // label stays fully inside the visible triangle.
  return (
    <span className="pointer-events-none absolute -right-px -top-px z-10 size-[5.5rem] overflow-hidden" aria-hidden>
      <span className="absolute top-[1.35rem] -right-8 w-[8.5rem] rotate-45 bg-muted py-1 text-center text-[9px] font-medium uppercase tracking-[0.06em] text-muted-foreground">
        Coming soon
      </span>
    </span>
  );
}

function connectOptionRowTone(soon: boolean, selected: boolean): string {
  if (soon) {
    return "overflow-hidden border-border/70 bg-muted/20 opacity-70";
  }
  if (selected) {
    return "border-foreground bg-accent/40";
  }
  return "border-border bg-background";
}

/**
 * Shared row for version control, issues, and coding agent options.
 * Connectable rows always show Connect / Connected on the right.
 */
export function ConnectOptionRow({
  icon,
  title,
  detail,
  selected,
  meta,
  connectLabel,
  connected,
  soon,
  onSelect,
  onConnect,
}: {
  icon: ReactNode;
  title: string;
  detail: string;
  selected?: boolean;
  meta?: string;
  /** When set, show Connect / Connected on the right. */
  connectLabel?: string;
  connected?: boolean;
  soon?: boolean;
  onSelect: () => void;
  onConnect?: () => void;
}) {
  const needsConnect = Boolean(connectLabel) && !connected && !soon;
  const rowToneClass = connectOptionRowTone(Boolean(soon), Boolean(selected));

  const select = () => {
    if (soon) return;
    onSelect();
    if (needsConnect) onConnect?.();
  };

  return (
    <div
      className={cn("relative rounded-lg border px-4 py-3 transition-colors", rowToneClass)}
      aria-disabled={soon || undefined}
      data-soon={soon ? "true" : undefined}
    >
      {soon ? <ComingSoonRibbon /> : null}
      <div className="flex flex-wrap items-start gap-3">
        <button
          type="button"
          className={cn("flex min-w-0 flex-1 items-start gap-3 text-left", soon && "cursor-not-allowed")}
          onClick={select}
          disabled={soon}
        >
          <span className="mt-0.5 shrink-0">{icon}</span>
          <span className="min-w-0 flex-1">
            <span className="flex flex-wrap items-center gap-2">
              <span className="text-[13px] font-medium tracking-[-0.01em]">{title}</span>
              {meta && !soon ? <span className="text-[11px] text-muted-foreground">{meta}</span> : null}
              {selected && !needsConnect && !soon ? (
                <Check className="size-3.5 text-foreground" strokeWidth={2.5} aria-hidden />
              ) : null}
            </span>
            <span className="mt-0.5 block text-[12px] text-muted-foreground">{detail}</span>
          </span>
        </button>
        {connectLabel && !soon ? (
          <div className="shrink-0 self-center">
            {connected ? (
              <span className="inline-flex items-center gap-1.5 text-[12px] text-emerald-700 dark:text-emerald-300">
                <Check className="size-3.5" strokeWidth={2.5} aria-hidden />
                Connected
              </span>
            ) : (
              <Button
                type="button"
                size="sm"
                onClick={() => {
                  onSelect();
                  onConnect?.();
                }}
              >
                Connect {connectLabel}
              </Button>
            )}
          </div>
        ) : null}
      </div>
    </div>
  );
}

export function NameStep({ setup }: { setup: OnboardingSetupApi }) {
  return (
    <div>
      <Label htmlFor="workspace-name" className="text-[13px] font-medium">
        Workspace name
      </Label>
      <Input
        id="workspace-name"
        value={setup.workspaceName}
        onChange={(event) => setup.editWorkspaceName(event.target.value)}
        placeholder="Payments Service"
        className="mt-2 h-10"
        autoFocus
      />
      <p className="mt-1.5 text-[12px] text-muted-foreground">
        Use a name for the app or product area this workspace will improve.
      </p>
    </div>
  );
}

export function RepositoryPicker({
  host,
  repos,
  selectedRepo,
  onSelect,
  title,
  description,
}: {
  host: VcsHostId;
  repos: string[];
  selectedRepo: string | null;
  onSelect: (repo: string) => void;
  /** Only for pickers that need their own heading, such as the backlog picker. */
  title?: string;
  description?: string;
}) {
  const [query, setQuery] = useState("");

  useEffect(() => {
    setQuery("");
  }, [host]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return repos;
    return repos.filter((repo) => repo.toLowerCase().includes(q));
  }, [repos, query]);

  return (
    <div className="space-y-3">
      {title ? <div className="text-[13px] font-medium">{title}</div> : null}
      {description ? <p className="text-[12px] text-muted-foreground">{description}</p> : null}
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search repositories"
          className="h-9 pl-9"
          aria-label="Search repositories"
        />
      </div>
      <div
        className="max-h-56 overflow-y-auto rounded-lg border border-border"
        role="listbox"
        aria-label="Repositories"
      >
        {filtered.length === 0 ? (
          <p className="px-3 py-6 text-center text-[13px] text-muted-foreground">No matching repositories.</p>
        ) : (
          <ul className="divide-y divide-border">
            {filtered.map((repo) => {
              const selected = selectedRepo === repo;
              return (
                <li key={repo}>
                  <button
                    type="button"
                    role="option"
                    aria-selected={selected}
                    onClick={() => onSelect(repo)}
                    className={cn(
                      "flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors",
                      selected ? "bg-accent/50" : "hover:bg-accent/30",
                    )}
                  >
                    <IntegrationChoiceIcon name={host} />
                    <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{repo}</span>
                    {selected ? (
                      <Check className="size-3.5 shrink-0 text-foreground" strokeWidth={2.5} aria-hidden />
                    ) : null}
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}

export function VcsStep({
  github,
  selectedConnectionId,
  onSelectConnection,
  onCreateConnection,
}: {
  github: IntegrationInstanceSummary;
  selectedConnectionId?: string;
  onSelectConnection: (id: string, name: string) => void;
  onCreateConnection: () => void;
}) {
  const githubOption = VCS_OPTIONS.find((option) => option.id === "github");
  const gitlabOption = VCS_OPTIONS.find((option) => option.id === "gitlab");

  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <section className="rounded-lg border border-border bg-background p-5">
        <div className="flex items-center gap-3">
          <IntegrationIcon integrationName="github" className="size-10" size={40} />
          <div>
            <h2 className="text-[22px] font-semibold tracking-[-0.02em]">{githubOption?.label ?? "GitHub"}</h2>
            <p className="text-[12px] text-muted-foreground">
              {github.readyInstances.length > 0
                ? "Choose a connection or connect a new one."
                : "Connect GitHub to continue."}
            </p>
          </div>
        </div>

        {github.readyInstances.length > 0 ? (
          <div className="mt-5 space-y-2">
            <p className="text-[12px] font-medium text-muted-foreground">Existing connections</p>
            {github.readyInstances.map((connection) => {
              const id = connection.metadata?.id;
              const name = connection.metadata?.name || "Unnamed connection";
              if (!id) return null;

              const selected = id === selectedConnectionId;
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => onSelectConnection(id, name)}
                  className={cn(
                    "flex w-full items-center gap-3 rounded-md border px-3 py-2.5 text-left transition-colors",
                    selected
                      ? "border-foreground bg-accent/50"
                      : "border-border bg-background hover:border-foreground hover:bg-accent/30",
                  )}
                >
                  <IntegrationIcon integrationName="github" className="size-4 shrink-0" size={16} />
                  <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{name}</span>
                  {selected ? <Check className="size-4 shrink-0" aria-hidden /> : null}
                </button>
              );
            })}
            <Button type="button" variant="outline" className="mt-3 w-full" onClick={onCreateConnection}>
              <Plus className="size-4" aria-hidden />
              Connect new GitHub
            </Button>
          </div>
        ) : (
          <Button type="button" className="mt-5 w-full" onClick={onCreateConnection}>
            Connect GitHub
          </Button>
        )}
      </section>

      {gitlabOption ? (
        <div
          className="relative flex min-h-44 flex-col items-center justify-center gap-3 overflow-hidden rounded-lg border border-border/70 bg-muted/20 px-6 py-12 opacity-70"
          aria-disabled="true"
        >
          <ComingSoonRibbon />
          <IntegrationIcon integrationName="gitlab" className="size-10" size={40} />
          <span className="text-[22px] font-semibold tracking-[-0.02em] text-muted-foreground">
            {gitlabOption.label}
          </span>
        </div>
      ) : null}
    </div>
  );
}

export function RepositoryStep({
  setup,
  repos,
  onSelect,
  onEditConnection,
}: {
  setup: OnboardingSetupApi;
  repos?: string[];
  /** Selecting a repository is the answer for this step, so it also continues. */
  onSelect: (repo: string) => void;
  onEditConnection: () => void;
}) {
  const host = setup.vcsHost;
  if (!host || !setup.connected.has(host)) {
    return <p className="text-[13px] text-muted-foreground">Connect version control first.</p>;
  }

  const availableRepos = repos ?? FIXTURE_REPOS[host];

  return (
    <div className="space-y-3">
      <RepositoryPicker host={host} repos={availableRepos} selectedRepo={setup.selectedRepo} onSelect={onSelect} />
      <p className="text-[13px] text-muted-foreground">
        Do not see your repo?{" "}
        <button
          type="button"
          onClick={onEditConnection}
          className="font-medium text-foreground underline underline-offset-2 hover:no-underline"
        >
          Edit {vcsLabel(host)} connection
        </button>
      </p>
    </div>
  );
}
