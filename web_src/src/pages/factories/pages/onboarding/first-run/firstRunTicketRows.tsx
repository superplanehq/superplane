import { LoadingButton } from "@/components/ui/loading-button";
import { isIntakeSelectable, type IntakeSurfaceEntry, type IntakeSurfaceState } from "@/lib/intakeCatalog";
import { findIntakePresentation } from "@/lib/intakePresentation";

import { JiraProjectStep } from "../../JiraIntakeSetupSteps";
import { LinearProjectPicker } from "../../LinearProjectPicker";
import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunPanel } from "./FirstRunShell";
import type { FirstRunFlaggedChoiceBlock, FirstRunJiraProject } from "./FirstRunTicketsScreen";
import type { FirstRunTicketSource } from "./firstRunTypes";

function choiceNoticeCopy(block: FirstRunFlaggedChoiceBlock | null, loading: string, failed: string): string | null {
  if (block === "lookup-failed") return failed;
  if (block === "loading") return loading;
  return null;
}

const DEFAULT_TICKET_ROWS = ["github-issues", "jira-issues", "linear-issues"];

function ticketRows(ticketIntakes: IntakeSurfaceEntry[] | null | undefined) {
  if (!ticketIntakes) {
    return DEFAULT_TICKET_ROWS.map((key) => ({ key, entry: undefined as IntakeSurfaceEntry | undefined }));
  }
  return ticketIntakes.map((entry) => ({ key: entry.key, entry }));
}

type FirstRunTicketBodyProps = {
  ticketSource: FirstRunTicketSource | null;
  intakeState: (key: string) => IntakeSurfaceState | undefined;
  intakesLoading: boolean;
  ticketIntakes?: IntakeSurfaceEntry[] | null;
  jiraChoiceBlock: FirstRunFlaggedChoiceBlock | null;
  linearChoiceBlock: FirstRunFlaggedChoiceBlock | null;
  linearSelectable: boolean;
  linearLoading: boolean;
  jiraAvailable: boolean;
  continueLabel: string;
  continueDisabled: boolean;
  continueLoading: boolean;
  continueLoadingText: string;
  saving: boolean;
  jiraConnected: boolean;
  jiraProjects: FirstRunJiraProject[];
  jiraProjectsLoading: boolean;
  jiraProjectsError: boolean;
  jiraProjectId: string;
  linearConnected: boolean;
  linearProjects: FirstRunJiraProject[];
  linearProjectsLoading: boolean;
  linearProjectsError: boolean;
  linearProjectIds: string[];
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onAnalyzeTickets: () => void;
  onConnectJira?: () => void;
  onSelectJiraProject?: (id: string) => void;
  onRetryJiraProjects?: () => void;
  onConnectLinear?: () => void;
  onToggleLinearProject?: (id: string) => void;
  onRetryLinearProjects?: () => void;
};

export function FirstRunTicketBody(props: FirstRunTicketBodyProps) {
  const copy = FIRST_RUN_COPY.tickets;
  return (
    <div className="mt-8 space-y-4">
      <FirstRunPanel>
        <div className="space-y-3">
          {ticketRows(props.ticketIntakes).map(({ key, entry }) => (
            <FirstRunTicketRow
              key={key}
              intakeKey={key}
              entry={entry}
              state={props.intakeState(key)}
              intakesLoading={props.intakesLoading}
              ticketSource={props.ticketSource}
              saving={props.saving}
              jiraConnected={props.jiraConnected}
              linearSelectable={props.linearSelectable}
              linearLoading={props.linearLoading}
              linearConnected={props.linearConnected}
              onSelectTicketSource={props.onSelectTicketSource}
              onConnectJira={props.onConnectJira}
              onConnectLinear={props.onConnectLinear}
            />
          ))}
        </div>
        <FirstRunJiraProjectFields
          jiraAvailable={props.jiraAvailable}
          visible={props.ticketSource === "jira" && props.jiraConnected}
          copy={copy}
          saving={props.saving}
          jiraProjects={props.jiraProjects}
          jiraProjectsLoading={props.jiraProjectsLoading}
          jiraProjectsError={props.jiraProjectsError}
          jiraProjectId={props.jiraProjectId}
          onSelectJiraProject={props.onSelectJiraProject}
          onRetryJiraProjects={props.onRetryJiraProjects}
        />
        <FirstRunLinearProjectFields
          linearAvailable={props.linearSelectable}
          visible={props.ticketSource === "linear" && props.linearConnected}
          copy={copy}
          saving={props.saving}
          linearProjects={props.linearProjects}
          linearProjectsLoading={props.linearProjectsLoading}
          linearProjectsError={props.linearProjectsError}
          linearProjectIds={props.linearProjectIds}
          onToggleLinearProject={props.onToggleLinearProject}
          onRetryLinearProjects={props.onRetryLinearProjects}
        />
      </FirstRunPanel>
      <FirstRunChoiceNotice
        block={props.jiraChoiceBlock}
        loading={copy.jiraLookupLoading}
        failed={copy.jiraLookupFailed}
        testId="first-run-jira-choice-notice"
      />
      <FirstRunChoiceNotice
        block={props.linearChoiceBlock}
        loading={copy.linearLookupLoading}
        failed={copy.linearLookupFailed}
        testId="first-run-linear-choice-notice"
      />
      <div className="space-y-3">
        <LoadingButton
          type="button"
          className="w-full"
          disabled={props.continueDisabled}
          loading={props.continueLoading}
          loadingText={props.continueLoadingText}
          onClick={props.onAnalyzeTickets}
          data-testid="first-run-analyze-tickets"
        >
          {props.continueLabel}
        </LoadingButton>
      </div>
    </div>
  );
}

function FirstRunChoiceNotice({
  block,
  loading,
  failed,
  testId,
}: {
  block: FirstRunFlaggedChoiceBlock | null;
  loading: string;
  failed: string;
  testId: string;
}) {
  const notice = choiceNoticeCopy(block, loading, failed);
  if (!notice) return null;
  const lookupFailed = block === "lookup-failed";
  return (
    <p
      className={lookupFailed ? "text-[13px] text-destructive" : "text-[13px] text-muted-foreground"}
      role={lookupFailed ? "alert" : "status"}
      data-testid={testId}
    >
      {notice}
    </p>
  );
}

type FirstRunTicketRowProps = {
  intakeKey: string;
  entry?: IntakeSurfaceEntry;
  state: IntakeSurfaceState | undefined;
  intakesLoading: boolean;
  ticketSource: FirstRunTicketSource | null;
  saving: boolean;
  jiraConnected: boolean;
  linearSelectable: boolean;
  linearLoading: boolean;
  linearConnected: boolean;
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onConnectJira?: () => void;
  onConnectLinear?: () => void;
};

function FirstRunTicketRow(props: FirstRunTicketRowProps) {
  const copy = FIRST_RUN_COPY.tickets;
  const { intakeKey, state, saving, ticketSource } = props;
  const meta = state === "beta" ? copy.beta : undefined;

  if (intakeKey === "github-issues") {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="github" />}
        title={copy.githubIssues}
        detail={copy.githubIssuesHelper}
        meta={meta}
        selected={ticketSource === "github-issues"}
        soon={state === "soon"}
        disabled={saving}
        onSelect={() => props.onSelectTicketSource("github-issues")}
      />
    );
  }
  if (intakeKey === "jira-issues") {
    return <FirstRunJiraTicketRow {...props} meta={meta} />;
  }
  if (intakeKey === "linear-issues") {
    return <FirstRunLinearTicketRow {...props} meta={meta} />;
  }
  return <FirstRunSoonTicketRow intakeKey={intakeKey} entry={props.entry} saving={saving} />;
}

function FirstRunSoonTicketRow({
  intakeKey,
  entry,
  saving,
}: {
  intakeKey: string;
  entry?: IntakeSurfaceEntry;
  saving: boolean;
}) {
  const copy = FIRST_RUN_COPY.tickets;
  const name = entry?.name ?? findIntakePresentation(intakeKey)?.name ?? intakeKey;
  const iconSrc = entry?.iconSrc ?? findIntakePresentation(intakeKey)?.iconSrc;
  return (
    <ConnectOptionRow
      icon={<TicketIntakeGlyph name={name} iconSrc={iconSrc} />}
      title={name}
      detail={copy.intakeSoonHelper}
      soon
      disabled={saving}
      onSelect={() => undefined}
    />
  );
}

function TicketIntakeGlyph({ name, iconSrc }: { name: string; iconSrc?: string }) {
  if (iconSrc) {
    return <img src={iconSrc} alt="" className="size-5" />;
  }
  return (
    <span className="flex size-5 items-center justify-center rounded-md bg-muted text-[11px] font-medium text-muted-foreground">
      {name.charAt(0).toUpperCase()}
    </span>
  );
}

function FirstRunJiraTicketRow({
  state,
  intakesLoading,
  ticketSource,
  saving,
  jiraConnected,
  meta,
  onSelectTicketSource,
  onConnectJira,
}: FirstRunTicketRowProps & { meta?: string }) {
  const copy = FIRST_RUN_COPY.tickets;
  if (state === undefined && intakesLoading) {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="jira" />}
        title={copy.jira}
        detail={copy.jiraLookupLoading}
        disabled
        onSelect={() => undefined}
      />
    );
  }
  if (!isIntakeSelectable(state)) {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="jira" />}
        title={copy.jira}
        detail={copy.jiraSoonHelper}
        soon
        disabled={saving}
        onSelect={() => undefined}
      />
    );
  }
  return (
    <ConnectOptionRow
      icon={<IntegrationChoiceIcon name="jira" />}
      title={copy.jira}
      detail={copy.jiraHelper}
      meta={meta}
      selected={ticketSource === "jira"}
      connectLabel={copy.jira}
      connected={jiraConnected}
      disabled={saving}
      onSelect={() => onSelectTicketSource("jira")}
      onConnect={onConnectJira}
    />
  );
}

function FirstRunLinearTicketRow({
  ticketSource,
  saving,
  linearSelectable,
  linearLoading,
  linearConnected,
  meta,
  onSelectTicketSource,
  onConnectLinear,
}: FirstRunTicketRowProps & { meta?: string }) {
  const copy = FIRST_RUN_COPY.tickets;
  if (linearLoading) {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="linear" />}
        title={copy.linear}
        detail={copy.linearLookupLoading}
        disabled
        onSelect={() => undefined}
      />
    );
  }
  if (!linearSelectable) {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="linear" />}
        title={copy.linear}
        detail={copy.linearSoonHelper}
        soon
        disabled={saving}
        onSelect={() => undefined}
      />
    );
  }
  return (
    <ConnectOptionRow
      icon={<IntegrationChoiceIcon name="linear" />}
      title={copy.linear}
      detail={copy.linearHelper}
      meta={meta}
      selected={ticketSource === "linear"}
      connectLabel={copy.linear}
      connected={linearConnected}
      disabled={saving}
      onSelect={() => onSelectTicketSource("linear")}
      onConnect={onConnectLinear}
    />
  );
}

function FirstRunJiraProjectFields({
  jiraAvailable,
  visible,
  copy,
  saving,
  jiraProjects,
  jiraProjectsLoading,
  jiraProjectsError,
  jiraProjectId,
  onSelectJiraProject,
  onRetryJiraProjects,
}: {
  jiraAvailable: boolean;
  visible: boolean;
  copy: (typeof FIRST_RUN_COPY)["tickets"];
  saving: boolean;
  jiraProjects: FirstRunJiraProject[];
  jiraProjectsLoading: boolean;
  jiraProjectsError: boolean;
  jiraProjectId: string;
  onSelectJiraProject?: (id: string) => void;
  onRetryJiraProjects?: () => void;
}) {
  if (!jiraAvailable || !visible) {
    return null;
  }

  return (
    <div className="mt-4 border-t border-border pt-4" data-testid="first-run-jira-projects">
      <p className="mb-3 text-[13px] font-medium">{copy.jiraProjectHeading}</p>
      <fieldset disabled={saving} className="contents">
        <JiraProjectStep
          projects={jiraProjects}
          selectedId={jiraProjectId}
          loading={jiraProjectsLoading}
          error={jiraProjectsError}
          onSelect={(id) => onSelectJiraProject?.(id)}
          onRetry={() => onRetryJiraProjects?.()}
        />
      </fieldset>
    </div>
  );
}

function FirstRunLinearProjectFields({
  linearAvailable,
  visible,
  copy,
  saving,
  linearProjects,
  linearProjectsLoading,
  linearProjectsError,
  linearProjectIds,
  onToggleLinearProject,
  onRetryLinearProjects,
}: {
  linearAvailable: boolean;
  visible: boolean;
  copy: (typeof FIRST_RUN_COPY)["tickets"];
  saving: boolean;
  linearProjects: FirstRunJiraProject[];
  linearProjectsLoading: boolean;
  linearProjectsError: boolean;
  linearProjectIds: string[];
  onToggleLinearProject?: (id: string) => void;
  onRetryLinearProjects?: () => void;
}) {
  if (!linearAvailable || !visible) {
    return null;
  }

  return (
    <div className="mt-4 border-t border-border pt-4" data-testid="first-run-linear-projects">
      <p className="mb-3 text-[13px] font-medium">{copy.linearProjectHeading}</p>
      <fieldset disabled={saving} className="contents">
        <LinearProjectPicker
          projects={linearProjects}
          selectedIds={linearProjectIds}
          loading={linearProjectsLoading}
          error={linearProjectsError}
          onToggle={(id) => onToggleLinearProject?.(id)}
          onRetry={() => onRetryLinearProjects?.()}
        />
      </fieldset>
    </div>
  );
}
