import { AGENT_OPTIONS, VCS_OPTIONS, type IssuesChoiceId } from "./onboardingFixtures";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

function issuesLabel(setup: OnboardingSetupApi, choice: IssuesChoiceId | null): string {
  if (choice === "vcs" && setup.vcsHost) {
    const backlog = setup.issuesRepo ?? setup.selectedRepo;
    const host = VCS_OPTIONS.find((option) => option.id === setup.vcsHost)?.label ?? setup.vcsHost;
    return backlog ? `${host} Issues · ${backlog}` : `${host} Issues`;
  }
  if (choice === "linear") return "Linear";
  if (choice === "jira") return "Jira";
  if (choice === "skip") return "Manual work orders";
  return "Not set";
}

export function DonePanel({ setup }: { setup: OnboardingSetupApi }) {
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
          <span className="text-muted-foreground">Backlog</span> · {issuesLabel(setup, setup.issuesChoice)}
        </li>
        <li>
          <span className="text-muted-foreground">Coding agent</span> ·{" "}
          {AGENT_OPTIONS.find((option) => option.id === setup.agent)?.label ?? "—"}
        </li>
      </ul>
    </div>
  );
}
