import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Switch } from "@/components/ui/switch";
import type { ReactNode } from "react";

import { JiraProjectStep } from "../../JiraIntakeSetupSteps";
import { LinearProjectPicker } from "../../LinearProjectPicker";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunStepper } from "./FirstRunStepper";
import {
  currentTicketStep,
  type FirstRunTicketProjectState,
  ticketIntakeSteps,
  ticketProjectChoiceComplete,
  ticketSourceName,
} from "./firstRunTicketIntakeSteps";
import type { FirstRunTicketSource } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.tickets;

export type FirstRunIntakeSettingsProps = {
  /** How many existing issues the first import takes. */
  initialImportLimit: number;
  skipInitialImport: boolean;
  /** Absent on the first workspace: that import always runs. */
  onSkipInitialImportChange?: (skip: boolean) => void;
  automaticIntake: boolean;
  onAutomaticIntakeChange?: (on: boolean) => void;
};

export function FirstRunTicketIntakeStepper({
  source,
  connected,
  connecting,
  repository,
  projects,
  projectsConfirmed,
  saving,
  settings,
  finishButton,
  onConnect,
  onSelectJiraProject,
  onRetryJiraProjects,
  onToggleLinearProject,
  onRetryLinearProjects,
  onConfirmProjects,
}: {
  source: FirstRunTicketSource;
  connected: boolean;
  connecting: boolean;
  /** Repository the GitHub intake reads, e.g. "acme/payments". */
  repository?: string;
  projects: FirstRunTicketProjectState;
  projectsConfirmed: boolean;
  saving: boolean;
  settings: FirstRunIntakeSettingsProps;
  finishButton: ReactNode;
  onConnect?: () => void;
  onSelectJiraProject?: (id: string) => void;
  onRetryJiraProjects?: () => void;
  onToggleLinearProject?: (id: string) => void;
  onRetryLinearProjects?: () => void;
  onConfirmProjects: () => void;
}) {
  const current = currentTicketStep({ source, connected, projectsConfirmed });
  const steps = ticketIntakeSteps(source, repository, projects);
  const sourceName = ticketSourceName(source);

  if (current === "connect") {
    return (
      <FirstRunStepper
        steps={steps}
        current="connect"
        testId="first-run-ticket-stepper"
        action={
          <LoadingButton
            type="button"
            size="sm"
            onClick={onConnect}
            disabled={saving || !onConnect}
            loading={connecting}
            loadingText={copy.stepConnect(sourceName)}
            data-testid="first-run-connect-ticket-source"
          >
            {copy.stepConnect(sourceName)}
          </LoadingButton>
        }
      />
    );
  }

  if (current === "project") {
    return (
      <FirstRunStepper steps={steps} current="project" testId="first-run-ticket-stepper">
        <TicketProjectFields
          source={source}
          saving={saving}
          projects={projects}
          onSelectJiraProject={onSelectJiraProject}
          onRetryJiraProjects={onRetryJiraProjects}
          onToggleLinearProject={onToggleLinearProject}
          onRetryLinearProjects={onRetryLinearProjects}
        />
        <Button
          type="button"
          className="w-full"
          disabled={saving || !ticketProjectChoiceComplete(source, projects)}
          onClick={onConfirmProjects}
          data-testid="first-run-confirm-projects"
        >
          {copy.confirmProjects}
        </Button>
      </FirstRunStepper>
    );
  }

  return (
    <FirstRunStepper steps={steps} current="settings" testId="first-run-ticket-stepper">
      <FirstRunIntakeSettings {...settings} saving={saving} />
      {finishButton}
    </FirstRunStepper>
  );
}

function TicketProjectFields({
  source,
  saving,
  projects,
  onSelectJiraProject,
  onRetryJiraProjects,
  onToggleLinearProject,
  onRetryLinearProjects,
}: {
  source: FirstRunTicketSource;
  saving: boolean;
  projects: FirstRunTicketProjectState;
  onSelectJiraProject?: (id: string) => void;
  onRetryJiraProjects?: () => void;
  onToggleLinearProject?: (id: string) => void;
  onRetryLinearProjects?: () => void;
}) {
  if (source === "jira") {
    return (
      <fieldset disabled={saving} className="contents" data-testid="first-run-jira-projects">
        <JiraProjectStep
          projects={projects.jiraProjects}
          selectedId={projects.jiraProjectId}
          loading={projects.jiraProjectsLoading}
          error={projects.jiraProjectsError}
          onSelect={(id) => onSelectJiraProject?.(id)}
          onRetry={() => onRetryJiraProjects?.()}
        />
      </fieldset>
    );
  }
  if (source === "linear") {
    return (
      <fieldset disabled={saving} className="contents" data-testid="first-run-linear-projects">
        <LinearProjectPicker
          projects={projects.linearProjects}
          selectedIds={projects.linearProjectIds}
          loading={projects.linearProjectsLoading}
          error={projects.linearProjectsError}
          onToggle={(id) => onToggleLinearProject?.(id)}
          onRetry={() => onRetryLinearProjects?.()}
        />
      </fieldset>
    );
  }
  return null;
}

function IntakeToggleRow({
  id,
  label,
  helper,
  checked,
  disabled,
  onCheckedChange,
  testId,
}: {
  id: string;
  label: string;
  helper: string;
  checked: boolean;
  disabled: boolean;
  onCheckedChange: (checked: boolean) => void;
  testId: string;
}) {
  return (
    <div className="flex items-start justify-between gap-6">
      <div className="flex min-w-0 flex-col gap-1">
        <Label htmlFor={id} className="text-[13px] font-medium tracking-[-0.01em] text-foreground">
          {label}
        </Label>
        <p id={`${id}-help`} className="text-[12px] text-muted-foreground">
          {helper}
        </p>
      </div>
      <Switch
        id={id}
        checked={checked}
        disabled={disabled}
        aria-describedby={`${id}-help`}
        onCheckedChange={onCheckedChange}
        data-testid={testId}
      />
    </div>
  );
}

function FirstRunIntakeSettings({
  initialImportLimit,
  skipInitialImport,
  onSkipInitialImportChange,
  automaticIntake,
  onAutomaticIntakeChange,
  saving,
}: FirstRunIntakeSettingsProps & { saving: boolean }) {
  const importOff = Boolean(onSkipInitialImportChange) && skipInitialImport;
  return (
    <div className="space-y-4" data-testid="first-run-intake-settings">
      {onSkipInitialImportChange ? (
        <IntakeToggleRow
          id="first-run-import-existing"
          label={copy.importExisting}
          helper={skipInitialImport ? copy.importExistingOff : copy.importExistingOn(initialImportLimit)}
          checked={!skipInitialImport}
          disabled={saving}
          onCheckedChange={(checked) => onSkipInitialImportChange(!checked)}
          testId="onboarding-import-existing-issues"
        />
      ) : (
        <p className="text-[12px] text-muted-foreground" data-testid="onboarding-initial-import-fixed">
          {copy.importFixed(initialImportLimit)}
        </p>
      )}
      {onAutomaticIntakeChange ? (
        <IntakeToggleRow
          id="first-run-automatic-intake"
          label={copy.automaticIntake}
          helper={automaticIntake ? copy.automaticIntakeOn : copy.automaticIntakeOff}
          checked={automaticIntake}
          disabled={saving}
          onCheckedChange={onAutomaticIntakeChange}
          testId="onboarding-automatic-intake"
        />
      ) : null}
      {importOff && !automaticIntake ? (
        <p className="text-[12px] text-muted-foreground" role="status" data-testid="onboarding-no-intake-hint">
          {copy.nothingImportedHint}
        </p>
      ) : null}
    </div>
  );
}
