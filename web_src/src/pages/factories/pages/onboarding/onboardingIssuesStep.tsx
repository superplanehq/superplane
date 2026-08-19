import { cn } from "@/lib/utils";
import { Loader2 } from "lucide-react";
import { useEffect, useState } from "react";

import { FIXTURE_REPOS, vcsLabel, type IntegrationId } from "./onboardingFixtures";
import { IssuesSourceOptions } from "./onboardingIssuesOptions";
import { IntegrationChoiceIcon, RepositoryPicker } from "./onboardingSteps";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

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
            <IssuesSourceOptions
              setup={setup}
              backlogRepo={backlogRepo}
              host={host}
              onRequestConnect={onRequestConnect}
            />
          </div>
        </>
      ) : null}
    </div>
  );
}
