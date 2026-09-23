import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { Check, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";
import { useLocation } from "react-router";

import { IntakeSetupWizard, type IntakeSetupStepItem } from "./IntakeSetupWizard";
import { IntakeSkipInitialImportField } from "./IntakeSkipInitialImportField";
import { JiraCompletionColumnFields } from "./JiraCompletionColumnFields";
import { JIRA_COMPLETION_COLUMN_COPY } from "./jiraCompletionColumnCopy";
import { JIRA_INTAKE_SETUP_COPY } from "./jiraIntakeSetupCopy";
import { useJiraIntakeSetup, type JiraIntakeSetupModel } from "./useJiraIntakeSetup";

const JIRA_INTAKE_STEPS: IntakeSetupStepItem[] = [
  { id: "connection", label: "Connect Jira" },
  { id: "project", label: "Choose project" },
  { id: "completion", label: JIRA_INTAKE_SETUP_COPY.wizardStepColumn },
];

/** Full-width wizard action. Disabled stays a solid control, not a faded pill. */
const JIRA_INTAKE_ACTION_CLASS =
  "h-10 w-full rounded-lg disabled:bg-muted disabled:text-muted-foreground disabled:opacity-100";

interface JiraIntakeSetupDialogProps {
  organizationId: string;
  factoryId: string;
  onClose: () => void;
  onCreated: () => void;
  selectIntegrationId?: string;
}

export function JiraIntakeSetupDialog(props: JiraIntakeSetupDialogProps) {
  const location = useLocation();
  const setup = useJiraIntakeSetup(props.organizationId, props.factoryId, props.selectIntegrationId);
  const copy = jiraIntakeWizardCopy(setup);
  const showConnectAction =
    setup.step === "connection" && !setup.connectedQuery.isLoading && setup.jiraIntegrations.length === 0;

  return (
    <>
      <IntakeSetupWizard
        testId="jira-intake-setup"
        integrationName="Jira"
        step={setup.step}
        steps={JIRA_INTAKE_STEPS}
        plain={setup.step === "completion"}
        title={copy.title}
        helper={copy.helper}
        stepAction={
          showConnectAction ? (
            <Button
              type="button"
              disabled={setup.connecting}
              onClick={() => void setup.connectJira()}
              data-testid="jira-setup-connect"
            >
              {setup.connecting ? JIRA_INTAKE_SETUP_COPY.wizardConnecting : JIRA_INTAKE_SETUP_COPY.wizardConnect}
            </Button>
          ) : undefined
        }
        footer={<SetupFooter setup={setup} onCreated={props.onCreated} />}
        onBack={() => leaveJiraIntakeStep(setup, props.onClose)}
      >
        {showConnectAction && !setup.error ? null : (
          <div>
            <SetupStepBody organizationId={props.organizationId} setup={setup} />
            {setup.error ? (
              <p className="workspace-body-text mt-4 text-destructive" role="alert">
                {setup.error}
              </p>
            ) : null}
          </div>
        )}
      </IntakeSetupWizard>
      <IntegrationCreateDialog
        open={setup.connectOpen}
        onOpenChange={setup.setConnectOpen}
        integrationDefinition={setup.jiraDefinition}
        organizationId={props.organizationId}
        onCreateIntegration={async (payload) => {
          const response = await setup.createIntegration.mutateAsync(payload);
          return response.data;
        }}
        onReset={setup.createIntegration.reset}
        defaultName="Jira"
        existingIntegrationNames={setup.existingNames}
        onCreated={setup.completeConnection}
        setupReturnTo={location.pathname}
      />
    </>
  );
}

function jiraIntakeWizardCopy(setup: JiraIntakeSetupModel): { title: string; helper: string } {
  if (setup.step === "connection") {
    return {
      title: JIRA_INTAKE_SETUP_COPY.wizardStepConnect,
      helper: JIRA_INTAKE_SETUP_COPY.wizardStepConnectHelper,
    };
  }
  if (setup.step === "completion") {
    return {
      title: JIRA_COMPLETION_COLUMN_COPY.section,
      helper: JIRA_INTAKE_SETUP_COPY.wizardStepCompletionHelper,
    };
  }
  return {
    title: JIRA_INTAKE_SETUP_COPY.wizardStepProject,
    helper: setup.skipInitialImport
      ? JIRA_INTAKE_SETUP_COPY.wizardStepProjectHelperSkip
      : JIRA_INTAKE_SETUP_COPY.wizardStepProjectHelper,
  };
}

function leaveJiraIntakeStep(setup: JiraIntakeSetupModel, onClose: () => void) {
  if (setup.step === "completion") {
    setup.setStep("project");
    return;
  }
  if (setup.step === "project") {
    setup.returnToConnection();
    return;
  }
  onClose();
}

function SetupStepBody({ organizationId, setup }: { organizationId: string; setup: JiraIntakeSetupModel }) {
  if (setup.step === "connection") {
    return (
      <ConnectionStep
        integrations={setup.jiraIntegrations}
        selectedId={setup.integrationId}
        loading={setup.connectedQuery.isLoading}
        onSelect={setup.setIntegrationId}
      />
    );
  }

  if (setup.step === "completion") {
    return (
      <JiraCompletionColumnFields
        organizationId={organizationId}
        integrationId={setup.integrationId}
        projectId={setup.projectId}
        value={setup.jiraCompletion}
        onChange={setup.setJiraCompletion}
        layout="plain"
        showSection={false}
      />
    );
  }

  return (
    <ProjectStep
      projects={setup.projectsQuery.data ?? []}
      selectedId={setup.projectId}
      loading={setup.projectsQuery.isLoading}
      error={setup.projectsQuery.isError}
      onSelect={setup.setProjectId}
      onRetry={() => void setup.projectsQuery.refetch()}
    />
  );
}

function ConnectionStep({
  integrations,
  selectedId,
  loading,
  onSelect,
}: {
  integrations: OrganizationsIntegration[];
  selectedId: string;
  loading: boolean;
  onSelect: (id: string) => void;
}) {
  if (loading) {
    return (
      <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        {JIRA_INTAKE_SETUP_COPY.wizardConnectionsLoading}
      </p>
    );
  }
  if (integrations.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2">
      <p className="text-[13px] font-medium">{JIRA_INTAKE_SETUP_COPY.wizardStepConnectExisting}</p>
      {integrations.map((integration) => {
        const id = integration.metadata?.id ?? "";
        const selected = id === selectedId;
        return (
          <Button
            key={id}
            type="button"
            variant="ghost"
            onClick={() => onSelect(id)}
            data-testid={`jira-connection-${id}`}
            className={cn(
              "h-auto w-full justify-between rounded-lg border px-3 py-3 text-left text-[13px] font-medium",
              selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30",
            )}
          >
            <span className="min-w-0 flex-1 truncate">{integration.metadata?.name || "Jira"}</span>
            {selected ? <Check className="size-4 shrink-0" aria-hidden /> : null}
          </Button>
        );
      })}
    </div>
  );
}

function ProjectStep({
  projects,
  selectedId,
  loading,
  error,
  onSelect,
  onRetry,
}: {
  projects: Array<{ id?: string; name?: string }>;
  selectedId: string;
  loading: boolean;
  error: boolean;
  onSelect: (id: string) => void;
  onRetry: () => void;
}) {
  if (loading) {
    return (
      <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        {JIRA_INTAKE_SETUP_COPY.wizardProjectsLoading}
      </p>
    );
  }
  if (error) {
    return (
      <div className="space-y-3">
        <p className="workspace-body-text text-destructive">{JIRA_INTAKE_SETUP_COPY.wizardProjectsError}</p>
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          {JIRA_INTAKE_SETUP_COPY.wizardRetry}
        </Button>
      </div>
    );
  }
  if (projects.length === 0) {
    return <p className="workspace-body-text text-muted-foreground">{JIRA_INTAKE_SETUP_COPY.wizardProjectsEmpty}</p>;
  }
  return <ProjectPicker projects={projects} selectedId={selectedId} onSelect={onSelect} />;
}

function ProjectPicker({
  projects,
  selectedId,
  onSelect,
}: {
  projects: Array<{ id?: string; name?: string }>;
  selectedId: string;
  onSelect: (id: string) => void;
}) {
  const [query, setQuery] = useState("");
  const filtered = useMemo(() => {
    const term = query.trim().toLowerCase();
    if (!term) return projects;
    return projects.filter((project) => (project.name ?? "").toLowerCase().includes(term));
  }, [projects, query]);

  return (
    <div className="space-y-3">
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Search projects"
          className="h-9 pl-9"
          aria-label="Search projects"
        />
      </div>
      <div className="max-h-56 overflow-y-auto rounded-lg border border-border" role="listbox" aria-label="Projects">
        {filtered.length === 0 ? (
          <p className="px-3 py-6 text-center text-[13px] text-muted-foreground">No matching projects.</p>
        ) : (
          <ul className="divide-y divide-border">
            {filtered.map((project) => {
              const id = project.id ?? "";
              const selected = id === selectedId;
              return (
                <li key={id}>
                  <Button
                    type="button"
                    variant="ghost"
                    role="option"
                    aria-selected={selected}
                    onClick={() => onSelect(id)}
                    data-testid={`jira-project-${id}`}
                    className={cn(
                      "h-auto w-full justify-start gap-3 rounded-none px-3 py-2.5 text-left text-[13px] font-medium",
                      selected ? "bg-accent/50" : "hover:bg-accent/30",
                    )}
                  >
                    <span className="min-w-0 flex-1 truncate">{project.name || "Untitled project"}</span>
                    {selected ? (
                      <Check className="size-3.5 shrink-0 text-foreground" strokeWidth={2.5} aria-hidden />
                    ) : null}
                  </Button>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}

function SetupFooter({ setup, onCreated }: { setup: JiraIntakeSetupModel; onCreated: () => void }) {
  if (setup.step === "connection") {
    return (
      <div>
        <Button
          type="button"
          className={JIRA_INTAKE_ACTION_CLASS}
          disabled={!setup.integrationId}
          onClick={() => setup.setStep("project")}
          data-testid="jira-setup-continue"
        >
          {JIRA_INTAKE_SETUP_COPY.wizardContinue}
        </Button>
      </div>
    );
  }

  if (setup.step === "project") {
    return (
      <div className="space-y-4">
        <IntakeSkipInitialImportField
          checked={setup.skipInitialImport}
          onCheckedChange={setup.setSkipInitialImport}
          testId="jira-skip-initial-import"
        />
        <Button
          type="button"
          className={JIRA_INTAKE_ACTION_CLASS}
          disabled={!setup.projectId}
          onClick={() => setup.setStep("completion")}
          data-testid="jira-setup-continue"
        >
          {JIRA_INTAKE_SETUP_COPY.wizardContinue}
        </Button>
      </div>
    );
  }

  return (
    <div>
      <Button
        type="button"
        className={JIRA_INTAKE_ACTION_CLASS}
        disabled={!setup.projectId || setup.createIntake.isPending}
        onClick={() => {
          void setup.createBoundIntake().then((created) => {
            if (created) {
              onCreated();
            }
          });
        }}
        data-testid="jira-setup-finish"
      >
        {setup.createIntake.isPending ? JIRA_INTAKE_SETUP_COPY.wizardFinishing : JIRA_INTAKE_SETUP_COPY.wizardFinish}
      </Button>
    </div>
  );
}
