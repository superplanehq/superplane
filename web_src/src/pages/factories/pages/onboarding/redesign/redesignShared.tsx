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

export function ChoiceCard({
  title,
  detail,
  selected,
  disabled,
  badge,
  icon,
  onClick,
}: {
  title: string;
  detail: string;
  selected?: boolean;
  disabled?: boolean;
  badge?: string;
  icon?: ReactNode;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={cn(
        "flex w-full items-start gap-3 rounded-lg border px-4 py-3 text-left transition-colors",
        selected ? "border-foreground bg-accent/40" : "border-border bg-background hover:bg-accent/30",
        disabled && "cursor-not-allowed opacity-50 hover:bg-background",
      )}
    >
      {icon ? <span className="mt-0.5 shrink-0">{icon}</span> : null}
      <span className="min-w-0 flex-1">
        <span className="flex w-full items-center justify-between gap-2">
          <span className="text-[13px] font-medium tracking-[-0.01em]">{title}</span>
          <span className="flex shrink-0 items-center gap-2">
            {badge ? <span className="text-[11px] text-muted-foreground">{badge}</span> : null}
            {selected ? <Check className="size-3.5" strokeWidth={2.5} aria-hidden /> : null}
          </span>
        </span>
        <span className="mt-0.5 block text-[12px] text-muted-foreground">{detail}</span>
      </span>
    </button>
  );
}

function IntegrationChoiceIcon({ name, size = 20 }: { name: IntegrationId | VcsHostId; size?: number }) {
  return <IntegrationIcon integrationName={name} className="size-5" size={size} />;
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
          placeholder="Refunds"
          className="mt-2 h-10"
          autoFocus
        />
        <p className="mt-1.5 text-[12px] text-muted-foreground">Short name for this workspace.</p>
      </div>

      <div className="rounded-lg border border-border p-4">
        <div className="text-[13px] font-medium">Invite teammates</div>
        <p className="mt-1 text-[12px] text-muted-foreground">
          Optional. Share the organization invite link. People join as viewers. You can change roles later.
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
    <div className="space-y-3">
      <div className="text-[13px] font-medium">Select repository</div>
      <p className="text-[12px] text-muted-foreground">
        This workspace opens pull requests and writes output in this repository.
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
  const connected = host ? setup.connected.has(host) : false;
  const repos = host ? FIXTURE_REPOS[host] : [];

  return (
    <div className="space-y-5">
      <div className="grid gap-2 sm:grid-cols-2">
        {VCS_OPTIONS.map((option) => (
          <ChoiceCard
            key={option.id}
            icon={<IntegrationChoiceIcon name={option.id} />}
            title={option.label}
            detail={option.detail}
            selected={setup.vcsHost === option.id}
            onClick={() => setup.selectVcsHost(option.id as VcsHostId)}
          />
        ))}
      </div>

      {host ? (
        <div className="rounded-lg border border-border p-4">
          {!connected ? (
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex min-w-0 items-start gap-3">
                <IntegrationChoiceIcon name={host} />
                <div>
                  <div className="text-[13px] font-medium">Connect {vcsLabel(host)}</div>
                  <p className="mt-0.5 text-[12px] text-muted-foreground">
                    Required before you can choose a repository.
                  </p>
                </div>
              </div>
              <Button type="button" size="sm" onClick={() => onRequestConnect(host)}>
                Connect {vcsLabel(host)}
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              <div className="flex items-center gap-2 text-[13px] text-emerald-700 dark:text-emerald-300">
                <IntegrationChoiceIcon name={host} />
                <Check className="size-3.5" strokeWidth={2.5} aria-hidden />
                {vcsLabel(host)} connected
              </div>
              <RepositoryPicker
                host={host}
                repos={repos}
                selectedRepo={setup.selectedRepo}
                onSelect={setup.selectRepo}
              />
            </div>
          )}
        </div>
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
    return <p className="text-[13px] text-muted-foreground">Connect a repository first.</p>;
  }

  return (
    <div className="space-y-4">
      {(setup.issuesDiscovering || (!setup.issuesDiscovered && !setup.issuesChoice)) && (
        <div className="flex items-center gap-2 rounded-lg border border-border bg-accent/30 px-4 py-3 text-[13px]">
          <Loader2 className="size-3.5 animate-spin" aria-hidden />
          Looking for issues on {setup.selectedRepo}…
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
              Use these issues for work orders, connect another tracker, or skip and create work orders yourself.
            </p>
          </div>

          <div className="grid gap-2">
            <ChoiceCard
              icon={<IntegrationChoiceIcon name={host} />}
              title={`Use ${vcsLabel(host)} Issues`}
              detail={`Use the ${vcsLabel(host)} integration you already connected.`}
              selected={setup.issuesChoice === "vcs"}
              onClick={() => setup.setIssuesChoice("vcs")}
            />
            <ChoiceCard
              icon={<IntegrationChoiceIcon name="linear" />}
              title="Linear"
              detail={
                setup.connected.has("linear")
                  ? "Linear is connected. Import issues from Linear."
                  : "Connect Linear if your issues are in Linear."
              }
              selected={setup.issuesChoice === "linear"}
              onClick={() => {
                if (!setup.connected.has("linear")) {
                  onRequestConnect("linear");
                  return;
                }
                setup.setIssuesChoice("linear");
              }}
              badge={setup.connected.has("linear") ? "Connected" : "Connect"}
            />
            <ChoiceCard
              icon={<IntegrationChoiceIcon name="jira" />}
              title="Jira"
              detail={
                setup.connected.has("jira")
                  ? "Jira is connected. Import issues from Jira."
                  : "Connect Jira if your issues are in Jira."
              }
              selected={setup.issuesChoice === "jira"}
              onClick={() => {
                if (!setup.connected.has("jira")) {
                  onRequestConnect("jira");
                  return;
                }
                setup.setIssuesChoice("jira");
              }}
              badge={setup.connected.has("jira") ? "Connected" : "Connect"}
            />
            <ChoiceCard
              icon={<ListTodo className="size-5 text-muted-foreground" aria-hidden />}
              title="Skip for now"
              detail="Create work orders yourself. No issue analysis."
              selected={setup.issuesChoice === "skip"}
              onClick={() => setup.setIssuesChoice("skip")}
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
        const selected = setup.agent === option.id;
        return (
          <div key={option.id} className="rounded-lg border border-border p-3">
            <ChoiceCard
              icon={<IntegrationChoiceIcon name={option.integrationId} />}
              title={option.label}
              detail={option.detail}
              selected={selected}
              badge={option.recommended ? "Recommended" : undefined}
              onClick={() => setup.setAgent(option.id as AgentHarnessId)}
            />
            {selected ? (
              <div className="mt-3 flex flex-wrap items-center justify-between gap-2 px-1">
                {connected ? (
                  <span className="inline-flex items-center gap-1.5 text-[12px] text-emerald-700 dark:text-emerald-300">
                    <Check className="size-3.5" strokeWidth={2.5} aria-hidden />
                    {integrationLabel(option.integrationId)} connected
                  </span>
                ) : (
                  <>
                    <span className="text-[12px] text-muted-foreground">
                      Connect {integrationLabel(option.integrationId)} to use {option.label}.
                    </span>
                    <Button type="button" size="sm" onClick={() => onRequestConnect(option.integrationId)}>
                      Connect {integrationLabel(option.integrationId)}
                    </Button>
                  </>
                )}
              </div>
            ) : null}
          </div>
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
          Setup is complete. Background analysis can continue from the steps you finished.
        </p>
      </div>
      <ul className="space-y-2 text-[13px]">
        <li>
          <span className="text-muted-foreground">Workspace</span> · {setup.summary.workspaceName}
        </li>
        <li>
          <span className="text-muted-foreground">Repository</span> · {setup.summary.selectedRepo}
        </li>
        <li>
          <span className="text-muted-foreground">Issues</span> · {issuesLabel(setup.issuesChoice)}
        </li>
        <li>
          <span className="text-muted-foreground">Agent</span> ·{" "}
          {AGENT_OPTIONS.find((option) => option.id === setup.agent)?.label ?? "—"}
        </li>
      </ul>
    </div>
  );
}
