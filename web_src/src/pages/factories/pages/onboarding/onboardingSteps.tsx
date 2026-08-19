import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { Check, ListTodo, Loader2, Search } from "lucide-react";
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
        Use a short name for the app or product area this workspace will improve.
      </p>
    </div>
  );
}

export function StartStep({ setup }: { setup: OnboardingSetupApi }) {
  return (
    <div className="space-y-4">
      <div>
        <Label htmlFor="start-work-order-title" className="text-[13px] font-medium">
          Work order title
        </Label>
        <Input
          id="start-work-order-title"
          value={setup.workOrderTitle}
          onChange={(event) => setup.setWorkOrderTitle(event.target.value)}
          className="mt-2 h-10"
          autoFocus
        />
      </div>
      <div>
        <Label htmlFor="start-work-order-description" className="text-[13px] font-medium">
          Description
        </Label>
        <Textarea
          id="start-work-order-description"
          value={setup.workOrderDescription}
          onChange={(event) => setup.setWorkOrderDescription(event.target.value)}
          className="mt-2 min-h-40"
        />
        <p className="mt-1.5 text-[12px] text-muted-foreground">
          The agent uses this description when it improves AGENTS.md in the app repository.
        </p>
      </div>
    </div>
  );
}

function RepositoryPicker({
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
      <p className="text-[11px] text-muted-foreground">
        {filtered.length} of {repos.length} repositories
      </p>
    </div>
  );
}

export function VcsStep({
  setup,
  onRequestConnect,
}: {
  setup: OnboardingSetupApi;
  onRequestConnect: (id: IntegrationId) => void;
}) {
  return (
    <div className="grid gap-2">
      {VCS_OPTIONS.map((option) => {
        const optionId = option.id as VcsHostId;
        const connected = setup.connected.has(optionId);
        const selected = setup.vcsHost === optionId;
        return (
          <ConnectOptionRow
            key={option.id}
            icon={<IntegrationChoiceIcon name={option.id} />}
            title={option.label}
            detail={option.detail}
            selected={selected}
            connectLabel={vcsLabel(optionId)}
            connected={connected}
            soon={option.soon}
            onSelect={() => setup.selectVcsHost(optionId)}
            onConnect={() => onRequestConnect(optionId)}
          />
        );
      })}
    </div>
  );
}

export function RepositoryStep({
  setup,
  repos,
  onSelect,
}: {
  setup: OnboardingSetupApi;
  repos?: string[];
  /** Selecting a repository is the answer for this step, so it also continues. */
  onSelect: (repo: string) => void;
}) {
  const host = setup.vcsHost;
  if (!host || !setup.connected.has(host)) {
    return <p className="text-[13px] text-muted-foreground">Connect version control first.</p>;
  }

  const availableRepos = repos ?? FIXTURE_REPOS[host];

  return <RepositoryPicker host={host} repos={availableRepos} selectedRepo={setup.selectedRepo} onSelect={onSelect} />;
}

export function IssuesStep({
  setup,
  onRequestConnect,
  autoDiscover,
  repos,
}: {
  setup: OnboardingSetupApi;
  onRequestConnect: (id: IntegrationId) => void;
  autoDiscover?: boolean;
  repos?: string[];
}) {
  const {
    selectedRepo,
    issuesRepo,
    issuesDiscovered,
    issuesDiscovering,
    issuesChoice,
    startIssuesDiscovery,
    selectIssuesRepo,
  } = setup;
  const [pickingBacklogRepo, setPickingBacklogRepo] = useState(false);

  useEffect(() => {
    if (!autoDiscover) return;
    if (!selectedRepo || issuesDiscovered || issuesDiscovering || issuesChoice) return;
    startIssuesDiscovery();
  }, [autoDiscover, selectedRepo, issuesDiscovered, issuesDiscovering, issuesChoice, startIssuesDiscovery]);

  useEffect(() => {
    setPickingBacklogRepo(false);
  }, [selectedRepo]);

  const host = setup.vcsHost;
  if (!host || !selectedRepo) {
    return <p className="text-[13px] text-muted-foreground">Select an app repository first.</p>;
  }

  const backlogRepo = issuesRepo ?? selectedRepo;
  const availableRepos = repos ?? FIXTURE_REPOS[host];
  const showDiscoveryResult = setup.issuesDiscovered || Boolean(setup.issuesChoice);

  return (
    <div className="space-y-4">
      {(setup.issuesDiscovering || (!showDiscoveryResult && !pickingBacklogRepo)) && (
        <div className="flex items-center gap-2 rounded-lg border border-border bg-accent/30 px-4 py-3 text-[13px]">
          <Loader2 className="size-3.5 animate-spin" aria-hidden />
          Looking for backlog issues on {backlogRepo}…
        </div>
      )}

      {showDiscoveryResult && !setup.issuesDiscovering ? (
        <>
          <div className="rounded-lg border border-border px-4 py-3">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="text-[13px] font-medium">
                  {setup.issueCount === undefined
                    ? `Use ${vcsLabel(host)} Issues`
                    : `Found ${setup.issueCount} open issues on ${vcsLabel(host)}`}
                </div>
                <p className="mt-1 text-[12px] text-muted-foreground">
                  Select the repository that SuperPlane will use for work orders.
                </p>
              </div>
              <button
                type="button"
                onClick={() => setPickingBacklogRepo((open) => !open)}
                className={cn(
                  "inline-flex max-w-[55%] shrink-0 items-center gap-1.5 rounded-md border border-border bg-accent/40 px-2 py-1 text-left text-[12px] font-medium tracking-[-0.01em] transition-colors hover:bg-accent",
                  pickingBacklogRepo && "border-foreground bg-accent",
                )}
                aria-label="Change backlog repository"
                aria-expanded={pickingBacklogRepo}
              >
                <IntegrationChoiceIcon name={host} size={14} />
                <span className="truncate">{backlogRepo}</span>
              </button>
            </div>
            {pickingBacklogRepo ? (
              <div className="mt-3">
                <RepositoryPicker
                  host={host}
                  repos={availableRepos}
                  selectedRepo={backlogRepo}
                  title="Select backlog repository"
                  description="Choose the repository that holds the issue backlog. The app repository does not change."
                  onSelect={(repo) => {
                    selectIssuesRepo(repo);
                    setPickingBacklogRepo(false);
                  }}
                />
              </div>
            ) : null}
          </div>

          <div className="grid gap-2">
            <ConnectOptionRow
              icon={<IntegrationChoiceIcon name={host} />}
              title={`Use ${vcsLabel(host)} Issues`}
              detail={`Find agent-ready work in open issues on ${backlogRepo}.`}
              selected={setup.issuesChoice === "vcs"}
              connectLabel={vcsLabel(host)}
              connected
              onSelect={() => setup.setIssuesChoice("vcs")}
            />
            <ConnectOptionRow
              icon={<IntegrationChoiceIcon name="linear" />}
              title="Linear"
              detail="Find agent-ready work in your Linear backlog."
              selected={setup.issuesChoice === "linear"}
              connectLabel="Linear"
              connected={setup.connected.has("linear")}
              soon
              onSelect={() => setup.setIssuesChoice("linear")}
              onConnect={() => onRequestConnect("linear")}
            />
            <ConnectOptionRow
              icon={<IntegrationChoiceIcon name="jira" />}
              title="Jira"
              detail="Find agent-ready work in your Jira backlog."
              selected={setup.issuesChoice === "jira"}
              connectLabel="Jira"
              connected={setup.connected.has("jira")}
              soon
              onSelect={() => setup.setIssuesChoice("jira")}
              onConnect={() => onRequestConnect("jira")}
            />
            <ConnectOptionRow
              icon={<ListTodo className="size-5 text-muted-foreground" aria-hidden />}
              title="Skip for now"
              detail="Do not import a backlog. Create work orders yourself for selected tasks."
              selected={setup.issuesChoice === "skip"}
              onSelect={() => setup.setIssuesChoice("skip")}
            />
          </div>
        </>
      ) : null}
    </div>
  );
}
