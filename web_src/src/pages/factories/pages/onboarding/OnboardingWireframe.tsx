import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Check, Search } from "lucide-react";
import { useMemo, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { factoryOverviewPath, factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { ONBOARDING_AVAILABLE_REPOS, providerLabel, type GitProvider, type OnboardingRepo } from "./onboardingMocks";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

function ProviderButton({
  provider,
  connected,
  onConnect,
}: {
  provider: GitProvider;
  connected: boolean;
  onConnect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onConnect}
      disabled={connected}
      className={cn(
        "flex h-10 flex-1 items-center justify-center gap-2 rounded-md border px-3 text-[13px] font-medium transition-colors",
        connected
          ? "cursor-default border-emerald-700/40 bg-emerald-50 text-emerald-800 dark:border-emerald-500/40 dark:bg-emerald-950/40 dark:text-emerald-300"
          : "cursor-pointer border-border bg-background text-foreground hover:bg-accent",
      )}
      data-testid={`onboarding-connect-${provider}`}
    >
      {connected ? <Check className="size-3.5" strokeWidth={2} aria-hidden /> : null}
      {connected ? `${providerLabel(provider)} connected` : `Connect ${providerLabel(provider)}`}
    </button>
  );
}

function RepoRow({
  repo,
  selected,
  onToggle,
}: {
  repo: OnboardingRepo;
  selected: boolean;
  onToggle: (repo: OnboardingRepo) => void;
}) {
  return (
    <li>
      <button
        type="button"
        onClick={() => onToggle(repo)}
        className="flex w-full cursor-pointer items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-accent/50"
      >
        <span
          className={cn(
            "flex size-4 shrink-0 items-center justify-center rounded border",
            selected ? "border-foreground bg-foreground text-background" : "border-border bg-background",
          )}
          aria-hidden
        >
          {selected ? <Check className="size-3" strokeWidth={2.5} /> : null}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-[13px] font-medium tracking-[-0.01em] text-foreground">
            {repo.org}/{repo.name}
          </span>
          <span className="mt-0.5 block text-[12px] text-muted-foreground">{providerLabel(repo.provider)}</span>
        </span>
      </button>
    </li>
  );
}

function RepositoryPicker({
  integrationsPath,
  query,
  onQueryChange,
  visibleRepos,
  selectedIds,
  selectedRepos,
  onToggleRepo,
}: {
  integrationsPath: string;
  query: string;
  onQueryChange: (value: string) => void;
  visibleRepos: OnboardingRepo[];
  selectedIds: string[];
  selectedRepos: OnboardingRepo[];
  onToggleRepo: (repo: OnboardingRepo) => void;
}) {
  return (
    <div className="mt-8">
      <div className="flex items-end justify-between gap-3">
        <div>
          <div className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Repositories</div>
          <p className="mt-0.5 text-[12px] text-muted-foreground">Select one or more repositories to enable.</p>
        </div>
        <Link
          to={integrationsPath}
          className="shrink-0 text-[12px] text-blue-600 transition-colors hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
        >
          Configure permissions
        </Link>
      </div>

      <div className="relative mt-3">
        <Search
          className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground"
          strokeWidth={1.75}
          aria-hidden
        />
        <Input
          value={query}
          onChange={(event) => onQueryChange(event.target.value)}
          placeholder="Search repositories..."
          className="h-9 bg-background pl-9 text-[13px] shadow-none"
          aria-label="Search repositories"
        />
      </div>

      <div className="mt-3 overflow-hidden rounded-lg border border-border">
        {visibleRepos.length === 0 ? (
          <p className="px-4 py-8 text-center text-[13px] text-muted-foreground">
            No repositories match. Try another search or configure permissions.
          </p>
        ) : (
          <ul className="divide-y divide-border">
            {visibleRepos.map((repo) => (
              <RepoRow key={repo.id} repo={repo} selected={selectedIds.includes(repo.id)} onToggle={onToggleRepo} />
            ))}
          </ul>
        )}
      </div>

      {selectedRepos.length > 0 ? (
        <div className="mt-3 flex flex-wrap gap-1.5">
          {selectedRepos.map((repo) => (
            <span
              key={repo.id}
              className="rounded-md border border-border bg-accent px-2 py-1 text-[12px] text-foreground"
            >
              {repo.org}/{repo.name}
            </span>
          ))}
        </div>
      ) : (
        <p className="mt-3 text-[12px] text-muted-foreground">Select at least one repository to continue.</p>
      )}
    </div>
  );
}

/**
 * Storybook-only factory onboarding wireframe (v3 parity).
 * Not mounted on production app routes.
 */
export function OnboardingWireframe() {
  const navigate = useNavigate();
  const { organizationId = "", factoryId = "" } = useParams<{ organizationId: string; factoryId: string }>();
  const layout = useFactoriesLayout();
  const onboarding = useOnboardingStorybook();

  const workspaceId = onboarding?.pending?.workspaceId ?? factoryId;
  const workspaceName = onboarding?.pending?.workspaceName ?? layout.factory?.name ?? "New workspace";

  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [query, setQuery] = useState("");

  const providers = useMemo(() => onboarding?.connectedProviders ?? [], [onboarding?.connectedProviders]);
  const connected = providers.length > 0;

  const visibleRepos = useMemo(() => {
    const allowed = new Set(providers);
    const repos = ONBOARDING_AVAILABLE_REPOS.filter((repo) => allowed.has(repo.provider));
    const normalized = query.trim().toLowerCase();
    if (!normalized) return repos;
    return repos.filter(
      (repo) =>
        repo.name.toLowerCase().includes(normalized) ||
        repo.org.toLowerCase().includes(normalized) ||
        repo.provider.includes(normalized),
    );
  }, [providers, query]);

  const selectedRepos = useMemo(
    () => ONBOARDING_AVAILABLE_REPOS.filter((repo) => selectedIds.includes(repo.id)),
    [selectedIds],
  );

  const canContinue = selectedIds.length > 0;
  const integrationsPath = factorySettingsSectionPath(organizationId, workspaceId, "integrations");

  const toggleRepo = (repo: OnboardingRepo) => {
    setSelectedIds((current) =>
      current.includes(repo.id) ? current.filter((id) => id !== repo.id) : [...current, repo.id],
    );
  };

  return (
    <div className="mx-auto flex w-full max-w-[560px] flex-col px-8 py-10" data-testid="onboarding-wireframe">
      <h1 className="text-[22px] font-semibold tracking-[-0.02em] text-foreground">Set up {workspaceName}</h1>
      <p className="mt-1.5 text-[13px] text-muted-foreground">
        Connect at least one repository so this factory can open PRs and run work orders.
      </p>

      <div className="mt-8 flex flex-col gap-2 sm:flex-row">
        <ProviderButton
          provider="github"
          connected={providers.includes("github")}
          onConnect={() => onboarding?.connectProvider("github")}
        />
        <ProviderButton
          provider="gitlab"
          connected={providers.includes("gitlab")}
          onConnect={() => onboarding?.connectProvider("gitlab")}
        />
      </div>

      {!connected ? (
        <p className="mt-6 text-[13px] text-muted-foreground">
          Connect GitHub or GitLab to see repositories you can add to this factory.
        </p>
      ) : (
        <RepositoryPicker
          integrationsPath={integrationsPath}
          query={query}
          onQueryChange={setQuery}
          visibleRepos={visibleRepos}
          selectedIds={selectedIds}
          selectedRepos={selectedRepos}
          onToggleRepo={toggleRepo}
        />
      )}

      <div className="mt-8 flex justify-end border-t border-border pt-5">
        <Button
          type="button"
          size="sm"
          disabled={!canContinue}
          onClick={() => {
            if (!canContinue || !onboarding) return;
            onboarding.completeOnboarding(workspaceId, selectedRepos);
            navigate(factoryOverviewPath(organizationId, workspaceId), { replace: true });
          }}
          data-testid="onboarding-continue"
        >
          Continue
        </Button>
      </div>
    </div>
  );
}
