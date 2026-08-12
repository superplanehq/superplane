import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { TooltipProvider } from "@/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Check, Copy, ListTodo, Loader2, Search } from "lucide-react";
import { useEffect, useMemo, useState, type ReactNode } from "react";

import { useFactoriesThemeClass } from "../../../lib/useFactoriesThemeClass";
import {
  AGENT_OPTIONS,
  FIXTURE_INVITE_URL,
  FIXTURE_REPOS,
  VCS_OPTIONS,
  integrationLabel,
  vcsLabel,
  type AgentHarnessId,
  type IntegrationId,
  type IssuesChoiceId,
  type VcsHostId,
} from "./redesignFixtures";
import type { RedesignSetupApi } from "./useRedesignSetupState";

export function Shell({ children, className }: { children: ReactNode; className?: string }) {
  useFactoriesThemeClass();
  const [queryClient] = useState(() => new QueryClient({ defaultOptions: { queries: { retry: false } } }));

  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider delayDuration={150}>
        <div className={cn("w-full bg-background text-foreground", className)} data-testid="onboarding-shell">
          {children}
        </div>
      </TooltipProvider>
    </QueryClientProvider>
  );
}

function IntegrationChoiceIcon({ name, size = 20 }: { name: IntegrationId | VcsHostId; size?: number }) {
  return <IntegrationIcon integrationName={name} className="size-5" size={size} />;
}

/**
 * Shared row for version control, issues, and coding agent options.
 * Connectable rows always show Connect / Connected on the right.
 */
function ConnectOptionRow({
  icon,
  title,
  detail,
  selected,
  meta,
  connectLabel,
  connected,
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
  onSelect: () => void;
  onConnect?: () => void;
}) {
  const needsConnect = Boolean(connectLabel) && !connected;

  const select = () => {
    onSelect();
    if (needsConnect) onConnect?.();
  };

  return (
    <div
      className={cn(
        "flex flex-wrap items-start gap-3 rounded-lg border px-4 py-3 transition-colors",
        selected ? "border-foreground bg-accent/40" : "border-border bg-background",
      )}
    >
      <button type="button" className="flex min-w-0 flex-1 items-start gap-3 text-left" onClick={select}>
        <span className="mt-0.5 shrink-0">{icon}</span>
        <span className="min-w-0 flex-1">
          <span className="flex flex-wrap items-center gap-2">
            <span className="text-[13px] font-medium tracking-[-0.01em]">{title}</span>
            {meta ? <span className="text-[11px] text-muted-foreground">{meta}</span> : null}
            {selected && !needsConnect ? (
              <Check className="size-3.5 text-foreground" strokeWidth={2.5} aria-hidden />
            ) : null}
          </span>
          <span className="mt-0.5 block text-[12px] text-muted-foreground">{detail}</span>
        </span>
      </button>
      {connectLabel ? (
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
  );
}

export function NameInviteStep({ setup }: { setup: RedesignSetupApi }) {
  return (
    <div className="space-y-6">
      <div>
        <label htmlFor="workspace-name" className="text-[13px] font-medium">
          Workspace name
        </label>
        <Input
          id="workspace-name"
          value={setup.workspaceName}
          onChange={(event) => setup.setWorkspaceName(event.target.value)}
          placeholder="Tempered Loom"
          className="mt-2 h-10"
          autoFocus
        />
        <p className="mt-1.5 text-[12px] text-muted-foreground">
          Starter name is editable. Use a short name for the app or product area this workspace will improve.
        </p>
      </div>

      <div className="rounded-lg border border-border p-4">
        <div className="text-[13px] font-medium">Invite teammates</div>
        <p className="mt-1 text-[12px] text-muted-foreground">
          Optional. Share the organization invite link so teammates can review agent pull requests. People join as
          viewers. You can change roles later.
        </p>
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <code className="max-w-full truncate rounded-md border border-border bg-accent/40 px-2 py-1.5 text-[12px]">
            {FIXTURE_INVITE_URL}
          </code>
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => {
              void navigator.clipboard?.writeText(FIXTURE_INVITE_URL);
              setup.setInviteCopied(true);
            }}
          >
            <Copy className="size-3.5" aria-hidden />
            {setup.inviteCopied ? "Copied" : "Copy link"}
          </Button>
        </div>
      </div>
    </div>
  );
}

function RepositoryPicker({
  host,
  repos,
  selectedRepo,
  onSelect,
}: {
  host: VcsHostId;
  repos: string[];
  selectedRepo: string | null;
  onSelect: (repo: string) => void;
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
    <div className="space-y-3 rounded-lg border border-border p-4">
      <div className="text-[13px] font-medium">Select repository</div>
      <p className="text-[12px] text-muted-foreground">
        Choose the app repository SuperPlane will analyze and agents will change. Pull requests open here.
      </p>
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

export function RepoStep({
  setup,
  onRequestConnect,
}: {
  setup: RedesignSetupApi;
  onRequestConnect: (id: IntegrationId) => void;
}) {
  const host = setup.vcsHost;
  const hostConnected = host ? setup.connected.has(host) : false;
  const repos = host && hostConnected ? FIXTURE_REPOS[host] : [];

  return (
    <div className="space-y-4">
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
              meta={option.id === "github" ? "Recommended" : undefined}
              connectLabel={vcsLabel(optionId)}
              connected={connected}
              onSelect={() => setup.selectVcsHost(optionId)}
              onConnect={() => onRequestConnect(optionId)}
            />
          );
        })}
      </div>

      {host && hostConnected ? (
        <RepositoryPicker host={host} repos={repos} selectedRepo={setup.selectedRepo} onSelect={setup.selectRepo} />
      ) : null}
    </div>
  );
}

export function IssuesStep({
  setup,
  onRequestConnect,
  autoDiscover,
}: {
  setup: RedesignSetupApi;
  onRequestConnect: (id: IntegrationId) => void;
  autoDiscover?: boolean;
}) {
  const { selectedRepo, issuesDiscovered, issuesDiscovering, issuesChoice, startIssuesDiscovery } = setup;

  useEffect(() => {
    if (!autoDiscover) return;
    if (!selectedRepo || issuesDiscovered || issuesDiscovering || issuesChoice) return;
    startIssuesDiscovery();
  }, [autoDiscover, selectedRepo, issuesDiscovered, issuesDiscovering, issuesChoice, startIssuesDiscovery]);

  const host = setup.vcsHost;
  if (!host || !setup.selectedRepo) {
    return <p className="text-[13px] text-muted-foreground">Select an app repository first.</p>;
  }

  return (
    <div className="space-y-4">
      {(setup.issuesDiscovering || (!setup.issuesDiscovered && !setup.issuesChoice)) && (
        <div className="flex items-center gap-2 rounded-lg border border-border bg-accent/30 px-4 py-3 text-[13px]">
          <Loader2 className="size-3.5 animate-spin" aria-hidden />
          Looking for backlog issues on {setup.selectedRepo}…
        </div>
      )}

      {setup.issuesDiscovered || setup.issuesChoice ? (
        <>
          <div className="rounded-lg border border-border px-4 py-3">
            <div className="flex items-center gap-2 text-[13px] font-medium">
              <IntegrationChoiceIcon name={host} />
              Found {setup.issueCount} open issues on {vcsLabel(host)}
            </div>
            <p className="mt-1 text-[12px] text-muted-foreground">
              SuperPlane can score these for autonomous work. Use this backlog, connect another tracker, or skip.
            </p>
          </div>

          <div className="grid gap-2">
            <ConnectOptionRow
              icon={<IntegrationChoiceIcon name={host} />}
              title={`Use ${vcsLabel(host)} Issues`}
              detail={`Find agent-ready work in open issues on ${setup.selectedRepo}.`}
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

export function AgentStep({
  setup,
  onRequestConnect,
}: {
  setup: RedesignSetupApi;
  onRequestConnect: (id: IntegrationId) => void;
}) {
  return (
    <div className="grid gap-2">
      {AGENT_OPTIONS.map((option) => {
        const connected = setup.connected.has(option.integrationId);
        return (
          <ConnectOptionRow
            key={option.id}
            icon={<IntegrationChoiceIcon name={option.integrationId} />}
            title={option.label}
            detail={option.detail}
            selected={setup.agent === option.id}
            meta={option.recommended ? "Recommended" : undefined}
            connectLabel={integrationLabel(option.integrationId)}
            connected={connected}
            onSelect={() => setup.setAgent(option.id as AgentHarnessId)}
            onConnect={() => onRequestConnect(option.integrationId)}
          />
        );
      })}
    </div>
  );
}

export function DonePanel({ setup }: { setup: RedesignSetupApi }) {
  const issuesLabel = (choice: IssuesChoiceId | null) => {
    if (choice === "vcs" && setup.vcsHost) return `${vcsLabel(setup.vcsHost)} Issues`;
    if (choice === "linear") return "Linear";
    if (choice === "jira") return "Jira";
    if (choice === "skip") return "Manual work orders";
    return "Not set";
  };

  return (
    <div className="space-y-4 rounded-lg border border-border p-5">
      <div>
        <h2 className="text-[18px] font-semibold tracking-[-0.02em]">Workspace ready</h2>
        <p className="mt-1 text-[13px] text-muted-foreground">
          SuperPlane can keep analyzing the app and backlog. Create a work order to hand the first task to your coding
          agent.
        </p>
      </div>
      <ul className="space-y-2 text-[13px]">
        <li>
          <span className="text-muted-foreground">Workspace</span> · {setup.summary.workspaceName}
        </li>
        <li>
          <span className="text-muted-foreground">App repository</span> · {setup.summary.selectedRepo}
        </li>
        <li>
          <span className="text-muted-foreground">Backlog</span> · {issuesLabel(setup.issuesChoice)}
        </li>
        <li>
          <span className="text-muted-foreground">Coding agent</span> ·{" "}
          {AGENT_OPTIONS.find((option) => option.id === setup.agent)?.label ?? "—"}
        </li>
      </ul>
    </div>
  );
}
