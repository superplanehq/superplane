import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import { ArrowLeft, Check, Loader2, Search } from "lucide-react";
import { useMemo, useState } from "react";

import { factoryPageTitleClassName } from "./factoryPageLayoutStyles";
import { PRFeedbackSetupWizardShell } from "./PRFeedbackSetupWizardChrome";
import { SentryIntakeSetupPreview } from "./SentryIntakeSetupPreview";
import { SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";
import { useSentryIntakeSetup, type SentryIntakeSetupModel, type SentrySetupStep } from "./useSentryIntakeSetup";

interface SentryIntakeSetupDialogProps {
  organizationId: string;
  factoryId: string;
  onClose: () => void;
  onCreated: () => void;
}

export function SentryIntakeSetupDialog(props: SentryIntakeSetupDialogProps) {
  const setup = useSentryIntakeSetup(props.organizationId, props.factoryId);
  const issueNames = (setup.issuesQuery.data ?? []).map((issue) => issue.name?.trim() ?? "").filter(Boolean);

  return (
    <>
      <PRFeedbackSetupWizardShell
        testId="sentry-intake-setup"
        preview={<SentryIntakeSetupPreview issueNames={issueNames} hasProject={Boolean(setup.projectId)} />}
      >
        <SetupHeader
          step={setup.step}
          onBack={() => {
            if (setup.step === "project") {
              setup.setStep("connection");
              return;
            }
            props.onClose();
          }}
        />
        <div>
          <SetupStepBody setup={setup} />
          {setup.error ? (
            <p className="workspace-body-text mt-4 text-destructive" role="alert">
              {setup.error}
            </p>
          ) : null}
        </div>
        <SetupFooter setup={setup} onCreated={props.onCreated} />
      </PRFeedbackSetupWizardShell>
      <IntegrationCreateDialog
        open={setup.connectOpen}
        onOpenChange={setup.setConnectOpen}
        integrationDefinition={setup.sentryDefinition}
        organizationId={props.organizationId}
        onCreateIntegration={async (payload) => {
          const response = await setup.createIntegration.mutateAsync(payload);
          return response.data;
        }}
        onReset={setup.createIntegration.reset}
        defaultName="Sentry"
        existingIntegrationNames={setup.existingNames}
        onCreated={setup.completeConnection}
      />
    </>
  );
}

function SetupHeader({ step, onBack }: { step: SentrySetupStep; onBack: () => void }) {
  return (
    <header className="text-left">
      <button
        type="button"
        className="mb-6 inline-flex items-center gap-1.5 text-[13px] text-muted-foreground hover:text-foreground"
        onClick={onBack}
        data-testid="sentry-setup-back"
      >
        <ArrowLeft className="size-4 shrink-0" aria-hidden />
        <span>
          {step === "project" ? SENTRY_INTAKE_SETUP_COPY.wizardBack : SENTRY_INTAKE_SETUP_COPY.wizardBackToBoard}
        </span>
      </button>
      <h1 className={factoryPageTitleClassName}>
        {step === "connection"
          ? SENTRY_INTAKE_SETUP_COPY.wizardStepConnect
          : SENTRY_INTAKE_SETUP_COPY.wizardStepProject}
      </h1>
      <p className="workspace-body-text mt-2 text-muted-foreground">
        {step === "connection"
          ? SENTRY_INTAKE_SETUP_COPY.wizardStepConnectHelper
          : SENTRY_INTAKE_SETUP_COPY.wizardStepProjectHelper}
      </p>
    </header>
  );
}

function SetupStepBody({ setup }: { setup: SentryIntakeSetupModel }) {
  if (setup.step === "connection") {
    return (
      <ConnectionStep
        integrations={setup.sentryIntegrations}
        selectedId={setup.integrationId}
        loading={setup.connectedQuery.isLoading}
        connecting={setup.connecting}
        onSelect={setup.setIntegrationId}
        onConnect={() => void setup.connectSentry()}
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
  connecting,
  onSelect,
  onConnect,
}: {
  integrations: OrganizationsIntegration[];
  selectedId: string;
  loading: boolean;
  connecting: boolean;
  onSelect: (id: string) => void;
  onConnect: () => void;
}) {
  if (loading) {
    return (
      <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
        <Loader2 className="size-4 animate-spin" aria-hidden />
        Loading Sentry connections...
      </p>
    );
  }

  return (
    <div className="space-y-4">
      {integrations.length > 0 ? (
        <div className="space-y-2">
          <p className="text-[13px] font-medium">{SENTRY_INTAKE_SETUP_COPY.wizardStepConnectExisting}</p>
          {integrations.map((integration) => {
            const id = integration.metadata?.id ?? "";
            const selected = id === selectedId;
            return (
              <button
                key={id}
                type="button"
                onClick={() => onSelect(id)}
                className={`flex w-full items-center justify-between rounded-lg border px-3 py-3 text-left ${
                  selected ? "border-foreground bg-accent/40" : "border-border hover:bg-accent/30"
                }`}
              >
                <span className="text-[13px] font-medium">{integration.metadata?.name || "Sentry"}</span>
                {selected ? <Check className="size-4" aria-hidden /> : null}
              </button>
            );
          })}
        </div>
      ) : null}
      <Button
        type="button"
        variant={integrations.length > 0 ? "outline" : "default"}
        disabled={connecting}
        onClick={onConnect}
        data-testid="sentry-setup-connect"
      >
        {connecting
          ? "Connecting..."
          : integrations.length > 0
            ? SENTRY_INTAKE_SETUP_COPY.wizardConnectAnother
            : SENTRY_INTAKE_SETUP_COPY.wizardConnect}
      </Button>
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
        {SENTRY_INTAKE_SETUP_COPY.wizardProjectsLoading}
      </p>
    );
  }
  if (error) {
    return (
      <div className="space-y-3">
        <p className="workspace-body-text text-destructive">{SENTRY_INTAKE_SETUP_COPY.wizardProjectsError}</p>
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          {SENTRY_INTAKE_SETUP_COPY.wizardRetry}
        </Button>
      </div>
    );
  }
  if (projects.length === 0) {
    return <p className="workspace-body-text text-muted-foreground">{SENTRY_INTAKE_SETUP_COPY.wizardProjectsEmpty}</p>;
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
                  <button
                    type="button"
                    role="option"
                    aria-selected={selected}
                    onClick={() => onSelect(id)}
                    data-testid={`sentry-project-${id}`}
                    className={cn(
                      "flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors",
                      selected ? "bg-accent/50" : "hover:bg-accent/30",
                    )}
                  >
                    <span className="min-w-0 flex-1 truncate text-[13px] font-medium">
                      {project.name || "Untitled project"}
                    </span>
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
    </div>
  );
}

function SetupFooter({ setup, onCreated }: { setup: SentryIntakeSetupModel; onCreated: () => void }) {
  if (setup.step === "connection") {
    return (
      <div className="space-y-3">
        <Button
          type="button"
          disabled={!setup.integrationId}
          onClick={() => setup.setStep("project")}
          data-testid="sentry-setup-continue"
        >
          {SENTRY_INTAKE_SETUP_COPY.wizardContinue}
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <Button
        type="button"
        disabled={!setup.projectId || setup.createIntake.isPending}
        onClick={() => {
          void setup.createBoundIntake().then((created) => {
            if (created) {
              onCreated();
            }
          });
        }}
        data-testid="sentry-setup-finish"
      >
        {setup.createIntake.isPending
          ? SENTRY_INTAKE_SETUP_COPY.wizardFinishing
          : SENTRY_INTAKE_SETUP_COPY.wizardFinish}
      </Button>
    </div>
  );
}
