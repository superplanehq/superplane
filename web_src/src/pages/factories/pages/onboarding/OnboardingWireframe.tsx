import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Check, ChevronDown } from "lucide-react";
import { useState, type ReactNode } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { factoryOverviewPath } from "../../lib/factoryPagePaths";
import { AnalysisSidePanel } from "./redesign/AnalysisSidePanel";
import type { IntegrationId } from "./redesign/redesignFixtures";
import { AgentStep, DonePanel, IssuesStep, NameInviteStep, RepoStep, Shell } from "./redesign/redesignShared";
import { useConnectDialog } from "./redesign/useConnectDialog";
import { useRedesignSetupState } from "./redesign/useRedesignSetupState";
import { useOnboardingStorybook } from "./useOnboardingStorybook";

type SectionId = "name" | "repo" | "issues" | "agent";

function Section({
  id,
  title,
  summary,
  open,
  complete,
  locked,
  onOpen,
  children,
}: {
  id: SectionId;
  title: string;
  summary?: string;
  open: boolean;
  complete: boolean;
  locked?: boolean;
  onOpen: (id: SectionId) => void;
  children: ReactNode;
}) {
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
            {summary && !open ? (
              <span className="block truncate text-[12px] text-muted-foreground">{summary}</span>
            ) : null}
          </span>
        </span>
        <ChevronDown className={cn("size-4 shrink-0 text-muted-foreground transition", open && "rotate-180")} />
      </button>
      {open ? <div className="border-t border-border px-4 py-4">{children}</div> : null}
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
  setup: ReturnType<typeof useRedesignSetupState>;
  openSection: SectionId;
  setOpenSection: (id: SectionId) => void;
  requestConnect: (id: IntegrationId) => void;
  onFinish: () => void;
}) {
  return (
    <div className="space-y-3">
      <Section
        id="name"
        title="Name & invite"
        summary={setup.nameReady ? setup.workspaceName.trim() : undefined}
        open={openSection === "name"}
        complete={setup.nameReady}
        onOpen={setOpenSection}
      >
        <NameInviteStep setup={setup} />
        <div className="mt-4">
          <Button type="button" size="sm" disabled={!setup.nameReady} onClick={() => setOpenSection("repo")}>
            Continue to repository
          </Button>
        </div>
      </Section>

      <Section
        id="repo"
        title="Repository"
        summary={setup.selectedRepo ?? undefined}
        open={openSection === "repo"}
        complete={setup.repoReady}
        locked={!setup.nameReady}
        onOpen={setOpenSection}
      >
        <RepoStep setup={setup} onRequestConnect={requestConnect} />
        <div className="mt-4">
          <Button type="button" size="sm" disabled={!setup.repoReady} onClick={() => setOpenSection("issues")}>
            Continue to issues
          </Button>
        </div>
      </Section>

      <Section
        id="issues"
        title="Issues"
        summary={
          setup.issuesChoice === "skip"
            ? "Skipped. Manual work orders."
            : setup.issuesChoice
              ? "Source selected"
              : undefined
        }
        open={openSection === "issues"}
        complete={setup.issuesReady}
        locked={!setup.repoReady}
        onOpen={setOpenSection}
      >
        <IssuesStep setup={setup} onRequestConnect={requestConnect} autoDiscover />
        <div className="mt-4">
          <Button type="button" size="sm" disabled={!setup.issuesReady} onClick={() => setOpenSection("agent")}>
            Continue to agent
          </Button>
        </div>
      </Section>

      <Section
        id="agent"
        title="Agent"
        summary={setup.agentReady ? (setup.agent ?? undefined) : undefined}
        open={openSection === "agent"}
        complete={setup.agentReady}
        locked={!setup.issuesReady}
        onOpen={setOpenSection}
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
  const { organizationId = "", factoryId = "" } = useParams<{ organizationId: string; factoryId: string }>();
  const onboarding = useOnboardingStorybook();
  const setup = useRedesignSetupState(onboarding?.pending?.workspaceName ?? "");
  const { requestConnect, dialog } = useConnectDialog(setup);
  const [openSection, setOpenSection] = useState<SectionId>("name");

  const finishSetup = () => {
    if (onboarding && factoryId) {
      onboarding.completeOnboarding(factoryId, []);
    }
    if (organizationId && factoryId) {
      navigate(factoryOverviewPath(organizationId, factoryId), { replace: true });
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
          Finish each section to unlock the next. The setup log on the right follows each step.
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
                repoReady: setup.repoReady,
                issuesChoice: setup.issuesChoice,
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
