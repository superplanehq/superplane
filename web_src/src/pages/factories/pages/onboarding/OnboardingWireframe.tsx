import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Check, ChevronDown } from "lucide-react";
import { useState, type ReactNode } from "react";
import { useNavigate } from "react-router";

import { useFactoriesLayout } from "../../layout/factoriesLayoutContext";
import { factoryOverviewPath } from "../../lib/factoryPagePaths";
import type { OnboardingRepo } from "./onboardingMocks";
import { AnalysisSidePanel } from "./AnalysisSidePanel";
import { AGENT_OPTIONS, type IntegrationId, type VcsHostId } from "./onboardingFixtures";
import { AgentStep, DonePanel, IssuesStep, NameInviteStep, RepoStep, Shell } from "./onboardingSteps";
import { useConnectDialog } from "./useConnectDialog";
import { useOnboardingSetupState, type OnboardingSetupApi } from "./useOnboardingSetupState";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

type SectionId = "name" | "repo" | "issues" | "agent";

function issuesSectionSummary(setup: OnboardingSetupApi): string | undefined {
  if (setup.issuesChoice === "skip") {
    return "Skipped. Create work orders yourself.";
  }
  if (setup.issuesChoice === "vcs" && setup.vcsHost) {
    const hostLabel = setup.vcsHost === "github" ? "GitHub" : "GitLab";
    return `${hostLabel} Issues · ${setup.issuesRepo ?? setup.selectedRepo}`;
  }
  if (setup.issuesChoice === "linear") {
    return "Linear";
  }
  if (setup.issuesChoice === "jira") {
    return "Jira";
  }
  return undefined;
}

function onboardingReposFromSetup(selectedRepo: string | null, vcsHost: VcsHostId | null): OnboardingRepo[] {
  if (!selectedRepo || !vcsHost) {
    return [];
  }
  const [org, name] = selectedRepo.split("/");
  if (!org || !name) {
    return [];
  }
  return [{ id: `${vcsHost}-${org}-${name}`, name, org, provider: vcsHost }];
}

function Section({
  id,
  title,
  purpose,
  summary,
  open,
  complete,
  locked,
  onOpen,
  children,
}: {
  id: SectionId;
  title: string;
  /** Short why text. Shown when open, and when collapsed if there is no summary. */
  purpose: string;
  summary?: string;
  open: boolean;
  complete: boolean;
  locked?: boolean;
  onOpen: (id: SectionId) => void;
  children: ReactNode;
}) {
  const collapsedHint = summary ?? purpose;

  return (
    <section
      className={cn(
        "overflow-hidden rounded-lg border",
        open ? "border-foreground/40" : "border-border",
        locked && "opacity-50",
      )}
    >
      <button
        type="button"
        disabled={locked}
        onClick={() => onOpen(id)}
        className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left"
      >
        <span className="flex min-w-0 items-center gap-2">
          <span
            className={cn(
              "flex size-5 shrink-0 items-center justify-center rounded-full border text-[11px]",
              complete
                ? "border-emerald-600 bg-emerald-600 text-white"
                : "border-border bg-background text-muted-foreground",
            )}
          >
            {complete ? <Check className="size-3" strokeWidth={2.5} aria-hidden /> : null}
          </span>
          <span className="min-w-0">
            <span className="block text-[13px] font-medium">{title}</span>
            {!open ? <span className="mt-0.5 block text-[12px] text-muted-foreground">{collapsedHint}</span> : null}
          </span>
        </span>
        <ChevronDown className={cn("size-4 shrink-0 text-muted-foreground transition", open && "rotate-180")} />
      </button>
      {open ? (
        <div className="border-t border-border px-4 py-4">
          <p className="mb-4 text-[13px] text-muted-foreground">{purpose}</p>
          {children}
        </div>
      ) : null}
    </section>
  );
}

function SetupSections({
  setup,
  openSection,
  setOpenSection,
  requestConnect,
  onFinish,
}: {
  setup: OnboardingSetupApi;
  openSection: SectionId;
  setOpenSection: (id: SectionId) => void;
  requestConnect: (id: IntegrationId) => void;
  onFinish: () => void;
}) {
  return (
    <div className="space-y-3">
      <Section
        id="name"
        title="Name and invite"
        purpose="Name this workspace for continuous AI work on your app. Invite people who will collaborate on work orders."
        summary={setup.nameReady ? setup.workspaceName.trim() : undefined}
        open={openSection === "name"}
        complete={setup.nameReady}
        onOpen={setOpenSection}
      >
        <NameInviteStep setup={setup} />
        <div className="mt-4">
          <Button type="button" size="sm" disabled={!setup.nameReady} onClick={() => setOpenSection("repo")}>
            Continue to version control
          </Button>
        </div>
      </Section>

      <Section
        id="repo"
        title="Version control"
        purpose="Connect version control and pick the app repository. SuperPlane analyzes that codebase. Agents change it and open pull requests."
        summary={setup.selectedRepo ?? undefined}
        open={openSection === "repo"}
        complete={setup.repoReady}
        locked={!setup.nameReady}
        onOpen={setOpenSection}
      >
        <RepoStep setup={setup} onRequestConnect={requestConnect} />
        <div className="mt-4">
          <Button
            type="button"
            size="sm"
            disabled={!setup.repoReady}
            onClick={() => {
              setup.commitRepoStep();
              setOpenSection("issues");
            }}
          >
            Continue to issues
          </Button>
        </div>
      </Section>

      <Section
        id="issues"
        title="Issues"
        purpose="Optional. Point SuperPlane at a backlog so it can find small work that agents can solve. Or skip and create work orders yourself."
        summary={issuesSectionSummary(setup)}
        open={openSection === "issues"}
        complete={setup.issuesReady}
        locked={!setup.repoReady}
        onOpen={(id) => {
          if (id === "issues" && setup.repoReady) {
            setup.commitRepoStep();
          }
          setOpenSection(id);
        }}
      >
        <IssuesStep setup={setup} onRequestConnect={requestConnect} autoDiscover />
        <div className="mt-4">
          <Button
            type="button"
            size="sm"
            disabled={!setup.issuesReady}
            onClick={() => {
              setup.commitIssuesStep();
              setOpenSection("agent");
            }}
          >
            Continue to coding agent
          </Button>
        </div>
      </Section>

      <Section
        id="agent"
        title="Coding agent"
        purpose="Connect a cloud coding agent. It works issues in the app repository and opens pull requests without engineers watching each run."
        summary={
          setup.agentReady ? (AGENT_OPTIONS.find((option) => option.id === setup.agent)?.label ?? undefined) : undefined
        }
        open={openSection === "agent"}
        complete={setup.agentReady}
        locked={!setup.issuesReady}
        onOpen={(id) => {
          if (id === "agent" && setup.issuesReady) {
            setup.commitIssuesStep();
          }
          setOpenSection(id);
        }}
      >
        <AgentStep setup={setup} onRequestConnect={requestConnect} />
        <div className="mt-4">
          <Button type="button" size="sm" disabled={!setup.canFinish} onClick={onFinish}>
            Finish setup
          </Button>
        </div>
      </Section>
    </div>
  );
}

/**
 * Storybook-only workspace onboarding: progressive stack + setup.log side panel.
 * Connect uses the real IntegrationCreateDialog. Not mounted on production routes.
 */
export function OnboardingWireframe() {
  const navigate = useNavigate();
  const { organizationId, factoryId, factoryKey } = useFactoriesLayout();
  const onboarding = useOnboardingStorybook();
  const setup = useOnboardingSetupState(onboarding?.pending?.workspaceName ?? "");
  const { requestConnect, dialog } = useConnectDialog(setup);
  const [openSection, setOpenSection] = useState<SectionId>("name");

  const finishSetup = () => {
    if (onboarding && factoryId) {
      onboarding.completeOnboarding(factoryId, onboardingReposFromSetup(setup.selectedRepo, setup.vcsHost));
    }
    if (organizationId && factoryKey) {
      navigate(factoryOverviewPath(organizationId, factoryKey), { replace: true });
      return;
    }
    setup.setFinished(true);
  };

  if (setup.finished) {
    return (
      <Shell className="w-full">
        <div className="mx-auto max-w-lg px-8 py-14">
          <DonePanel setup={setup} />
        </div>
        {dialog}
      </Shell>
    );
  }

  return (
    <Shell className="w-full">
      <div className="mx-auto w-full max-w-6xl px-6 py-8 lg:px-8">
        <h1 className="text-[22px] font-semibold tracking-[-0.02em]">Set up your workspace</h1>
        <p className="mt-1.5 max-w-2xl text-[13px] text-muted-foreground">
          Hand off small engineering work to AI. SuperPlane finds candidate work in your app and backlog, then a coding
          agent opens pull requests. Finish one section to unlock the next.
        </p>

        <div className="mt-8 grid items-start gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(320px,400px)]">
          <SetupSections
            setup={setup}
            openSection={openSection}
            setOpenSection={setOpenSection}
            requestConnect={requestConnect}
            onFinish={finishSetup}
          />
          <div className="lg:sticky lg:top-6 lg:self-start">
            <AnalysisSidePanel
              progress={{
                workspaceName: setup.workspaceName,
                nameReady: setup.nameReady,
                selectedRepo: setup.selectedRepo,
                vcsHost: setup.vcsHost,
                repoCommitted: setup.repoCommitted,
                issuesChoice: setup.issuesChoice,
                issuesCommitted: setup.issuesCommitted,
                agent: setup.agent,
                agentReady: setup.agentReady,
              }}
            />
          </div>
        </div>
      </div>
      {dialog}
    </Shell>
  );
}
