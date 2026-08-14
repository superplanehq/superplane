import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ListTodo } from "lucide-react";
import { useState, type ReactNode } from "react";

import { factoryCardClassName } from "../../factoryPageLayoutStyles";
import { FIXTURE_REPOS, type IssuesChoiceId, type VcsHostId, vcsLabel } from "../../onboarding/onboardingFixtures";
import { RepositoryPicker } from "../../onboarding/onboardingSteps";
import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { useWorkspaceConnections } from "./useWorkspaceConnections";
import { WorkspaceSettingsSection } from "./WorkspaceSettingsSection";
import { appRepositoryFullName, appRepositorySummary, backlogSummary, repoFromFullName } from "./workspaceConnections";

export function WorkspaceRepositoriesPage() {
  const { factoryId } = useFactorySettingsLayout();
  const { connections, updateConnections } = useWorkspaceConnections(factoryId);
  const appRepo = connections.repos[0];
  const host = (appRepo?.provider ?? "github") as VcsHostId;
  const backlogRepo = connections.issuesRepo ?? appRepositoryFullName(appRepo);
  const [changingAppRepo, setChangingAppRepo] = useState(false);
  const [changingBacklogRepo, setChangingBacklogRepo] = useState(false);

  return (
    <WorkspaceSettingsSection
      title="Repositories"
      description="Choose the app repository that agents analyze and change. Optionally connect a backlog so SuperPlane can find work."
    >
      <div className="max-w-2xl space-y-6">
        <section className={cn("p-6", factoryCardClassName)} data-testid="workspace-settings-app-repo">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">App repository</h2>
              <p className="mt-1 text-[12px] text-muted-foreground">
                SuperPlane analyzes this repository. Coding agents change the code here and open pull requests.
              </p>
              <p className="mt-3 flex items-center gap-2 text-[13px] font-medium">
                {appRepo ? <IntegrationIcon integrationName={appRepo.provider} className="size-4" size={16} /> : null}
                {appRepositorySummary(connections)}
              </p>
            </div>
            <Button type="button" size="sm" variant="outline" onClick={() => setChangingAppRepo((open) => !open)}>
              {changingAppRepo ? "Close" : "Change"}
            </Button>
          </div>
          {changingAppRepo ? (
            <div className="mt-4">
              <RepositoryPicker
                host={host}
                repos={FIXTURE_REPOS[host]}
                selectedRepo={appRepositoryFullName(appRepo)}
                onSelect={(fullName) => {
                  const nextRepo = repoFromFullName(fullName, host);
                  const nextBacklog =
                    connections.issuesRepo === appRepositoryFullName(appRepo) || !connections.issuesRepo
                      ? fullName
                      : connections.issuesRepo;
                  updateConnections({
                    ...connections,
                    repos: [nextRepo],
                    issuesRepo: connections.issuesChoice === "vcs" ? nextBacklog : connections.issuesRepo,
                  });
                  setChangingAppRepo(false);
                }}
              />
            </div>
          ) : null}
        </section>

        <section className={cn("p-6", factoryCardClassName)} data-testid="workspace-settings-backlog">
          <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Backlog</h2>
          <p className="mt-1 text-[12px] text-muted-foreground">
            SuperPlane looks here for small work that a coding agent can complete. You can also skip this and create
            work orders yourself.
          </p>
          <p className="mt-3 text-[13px] font-medium">{backlogSummary(connections)}</p>

          <div className="mt-4 grid gap-2">
            <BacklogChoice
              selected={connections.issuesChoice === "vcs"}
              icon={<IntegrationIcon integrationName={host} className="size-5" size={20} />}
              title={`Use ${vcsLabel(host)} Issues`}
              detail={`Use open issues on ${backlogRepo || "the selected repository"} as the source of work.`}
              onSelect={() =>
                updateConnections({
                  ...connections,
                  issuesChoice: "vcs",
                  issuesRepo: backlogRepo || appRepositoryFullName(appRepo),
                })
              }
            />
            <BacklogChoice
              selected={connections.issuesChoice === "linear"}
              soon
              icon={<IntegrationIcon integrationName="linear" className="size-5" size={20} />}
              title="Linear"
              detail="Use Linear issues as the source of work."
              onSelect={() => updateConnections({ ...connections, issuesChoice: "linear" as IssuesChoiceId })}
            />
            <BacklogChoice
              selected={connections.issuesChoice === "jira"}
              soon
              icon={<IntegrationIcon integrationName="jira" className="size-5" size={20} />}
              title="Jira"
              detail="Use Jira issues as the source of work."
              onSelect={() => updateConnections({ ...connections, issuesChoice: "jira" as IssuesChoiceId })}
            />
            <BacklogChoice
              selected={connections.issuesChoice === "skip"}
              icon={<ListTodo className="size-5 text-muted-foreground" aria-hidden />}
              title="No backlog"
              detail="Do not import issues. Create each work order yourself."
              onSelect={() => updateConnections({ ...connections, issuesChoice: "skip", issuesRepo: null })}
            />
          </div>

          {connections.issuesChoice === "vcs" ? (
            <div className="mt-4">
              <Button type="button" size="sm" variant="outline" onClick={() => setChangingBacklogRepo((open) => !open)}>
                {changingBacklogRepo ? "Close" : "Change backlog repository"}
              </Button>
              {changingBacklogRepo ? (
                <div className="mt-3">
                  <RepositoryPicker
                    host={host}
                    repos={FIXTURE_REPOS[host]}
                    selectedRepo={backlogRepo}
                    title="Select backlog repository"
                    description="Choose the repository that holds the issue backlog. This does not change the app repository."
                    onSelect={(fullName) => {
                      updateConnections({ ...connections, issuesChoice: "vcs", issuesRepo: fullName });
                      setChangingBacklogRepo(false);
                    }}
                  />
                </div>
              ) : null}
            </div>
          ) : null}
        </section>
      </div>
    </WorkspaceSettingsSection>
  );
}

function BacklogChoice({
  icon,
  title,
  detail,
  selected,
  soon,
  onSelect,
}: {
  icon: ReactNode;
  title: string;
  detail: string;
  selected: boolean;
  soon?: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      disabled={soon}
      onClick={onSelect}
      className={cn(
        "relative flex items-start gap-3 rounded-lg border px-4 py-3 text-left transition-colors",
        soon && "cursor-not-allowed border-border/70 bg-muted/20 opacity-70",
        !soon && selected && "border-foreground bg-accent/40",
        !soon && !selected && "border-border bg-background hover:bg-accent/30",
      )}
    >
      <span className="mt-0.5 shrink-0">{icon}</span>
      <span className="min-w-0">
        <span className="block text-[13px] font-medium">{title}</span>
        <span className="mt-0.5 block text-[12px] text-muted-foreground">{detail}</span>
      </span>
    </button>
  );
}
