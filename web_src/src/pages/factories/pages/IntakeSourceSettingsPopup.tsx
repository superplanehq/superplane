import type { RunsSidebarHrefForRun } from "@/components/CanvasToolSidebar/runsSidebarHref";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import {
  forwardRef,
  useEffect,
  useMemo,
  useRef,
  useState,
  type Dispatch,
  type ReactNode,
  type SetStateAction,
} from "react";

import { IntakeConnectionFields, type IntakeConnectionFieldsProps } from "./IntakeConnectionFields";
import { IntakeSettingsLifecycle } from "./IntakeSourceSettingsFooter";
import { IntakeSettingsSidebar } from "./IntakeSettingsSidebar";
import { IntakeSettingsTableOfContents } from "./IntakeSettingsTableOfContents";
import {
  INTAKE_SETTINGS_COPY,
  intakeSettingsSections,
  intakeSettingsTabs,
  intakeSettingsTitle,
  resolveIntakeSettingsTab,
  type IntakeSettingsTab,
  type IntakeSourceSettings,
} from "./intakeSourceSettingsModel";
import { GitHubIntakeFilterFields } from "./GitHubIntakeFilterFields";
import { JiraIntakeFilterFields } from "./JiraIntakeFilterFields";
import { ProductiveIntakeFilterFields } from "./ProductiveIntakeFilterFields";
import { SentryIntakeFilterFields } from "./SentryIntakeFilterFields";
import { PlanningReviewEditor, type PlanningReviewAgentSlot } from "./PlanningReviewEditor";
import { SettingsAutomationCanvasEdit, SettingsAutomationWorkspace } from "./SettingsAutomationWorkspace";
import { PopupShell } from "./work-order-popup-redesign/popupShared";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export type IntakeSettingsConnection = Omit<IntakeConnectionFieldsProps, "sourceId"> & {
  saveDisabled?: boolean;
};

interface IntakeSourceSettingsPopupProps {
  settings: IntakeSourceSettings;
  sourceId?: LineIntakeSourceId;
  organizationId?: string;
  integrationId?: string;
  resourceId?: string;
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
  automationGraph?: IntakeAutomationGraph;
  automationLoading?: boolean;
  automationError?: boolean;
  onRetryAutomation?: () => void;
  onSave: (next: IntakeSourceSettings) => Promise<void> | void;
  savePending?: boolean;
  saveError?: string;
  paused?: boolean;
  pausePending?: boolean;
  deletePending?: boolean;
  pauseError?: string;
  deleteError?: string;
  onPause?: () => Promise<void> | void;
  onResume?: () => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
  editAutomationHref?: string;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  agent?: PlanningReviewAgentSlot;
  onClose: () => void;
  fixed?: boolean;
  initialTab?: IntakeSettingsTab;
  connection?: IntakeSettingsConnection;
}

export function IntakeSourceSettingsPopup({
  settings,
  sourceId = "github-issues",
  organizationId,
  integrationId,
  resourceId,
  labelOptions,
  labelOptionsLoading,
  automationGraph,
  automationLoading = false,
  automationError = false,
  onRetryAutomation,
  onSave,
  savePending = false,
  saveError,
  paused = false,
  pausePending = false,
  deletePending = false,
  pauseError,
  deleteError,
  onPause,
  onResume,
  onDelete,
  editAutomationHref,
  canvasId,
  runHrefFor,
  agent,
  onClose,
  fixed = true,
  initialTab = "general",
  connection,
}: IntakeSourceSettingsPopupProps) {
  const tabs = useMemo(() => intakeSettingsTabs(Boolean(agent)), [agent]);
  const [draft, setDraft] = useState(settings);
  const [tab, setTab] = useState<IntakeSettingsTab>(() => resolveIntakeSettingsTab(tabs, initialTab));

  useEffect(() => {
    setDraft(settings);
  }, [settings]);

  useEffect(() => {
    setTab((current) => resolveIntakeSettingsTab(tabs, current));
  }, [tabs]);

  return (
    <PopupShell
      testId="intake-source-settings"
      canvas
      fixed={fixed}
      onDismiss={onClose}
      className="bg-sidebar! text-sidebar-foreground"
    >
      <div className="flex min-h-0 min-w-0 flex-1 flex-col bg-sidebar">
        <IntakeSettingsSidebar
          sourceId={sourceId}
          title={intakeSettingsTitle(sourceId)}
          tabs={tabs}
          active={tab}
          onSelect={setTab}
          onClose={onClose}
          save={
            tab === "general"
              ? {
                  draft,
                  savePending,
                  saveError,
                  onSave,
                  onClose,
                  saveDisabled: connection?.saveDisabled,
                }
              : undefined
          }
        />
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          <IntakeSettingsTabPanel
            tab={tab}
            sourceId={sourceId}
            organizationId={organizationId}
            integrationId={integrationId}
            resourceId={resourceId}
            labelOptions={labelOptions}
            labelOptionsLoading={labelOptionsLoading}
            draft={draft}
            agent={agent}
            automationGraph={automationGraph}
            automationLoading={automationLoading}
            automationError={automationError}
            onRetryAutomation={onRetryAutomation}
            canvasId={canvasId}
            runHrefFor={runHrefFor}
            editAutomationHref={editAutomationHref}
            onDraftChange={setDraft}
            connection={connection}
            paused={paused}
            pausePending={pausePending}
            deletePending={deletePending}
            pauseError={pauseError}
            deleteError={deleteError}
            onPause={onPause}
            onResume={onResume}
            onDelete={onDelete}
          />
        </div>
      </div>
    </PopupShell>
  );
}

function IntakeSettingsTabPanel({
  tab,
  sourceId,
  organizationId,
  integrationId,
  resourceId,
  labelOptions,
  labelOptionsLoading,
  draft,
  agent,
  automationGraph,
  automationLoading,
  automationError,
  onRetryAutomation,
  canvasId,
  runHrefFor,
  editAutomationHref,
  onDraftChange,
  connection,
  paused,
  pausePending,
  deletePending,
  pauseError,
  deleteError,
  onPause,
  onResume,
  onDelete,
}: {
  tab: IntakeSettingsTab;
  sourceId: LineIntakeSourceId;
  organizationId?: string;
  integrationId?: string;
  resourceId?: string;
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
  draft: IntakeSourceSettings;
  agent?: PlanningReviewAgentSlot;
  automationGraph?: IntakeAutomationGraph;
  automationLoading: boolean;
  automationError: boolean;
  onRetryAutomation?: () => void;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  editAutomationHref?: string;
  onDraftChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  connection?: IntakeSettingsConnection;
  paused: boolean;
  pausePending: boolean;
  deletePending: boolean;
  pauseError?: string;
  deleteError?: string;
  onPause?: () => Promise<void> | void;
  onResume?: () => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
}) {
  if (tab === "automation") {
    return (
      <IntakeAutomationTab
        graph={automationGraph}
        canvasId={canvasId}
        runHrefFor={runHrefFor}
        editHref={editAutomationHref}
        loading={automationLoading}
        error={automationError}
        onRetry={onRetryAutomation}
      />
    );
  }
  if (tab === "agent" && agent) {
    return (
      <PlanningReviewEditor
        key={agent.draft?.components[0]?.id ?? "agent"}
        initialDraft={agent.draft}
        onSave={agent.onSave}
        organizationId={agent.organizationId}
        factoryId={agent.factoryId}
        factoryKey={agent.factoryKey}
        isLoading={agent.isLoading}
        showAutomationNote={false}
        showCancel={false}
      />
    );
  }
  return (
    <IntakeSettingsGeneralPanel
      sourceId={sourceId}
      organizationId={organizationId}
      integrationId={integrationId}
      resourceId={resourceId}
      labelOptions={labelOptions}
      labelOptionsLoading={labelOptionsLoading}
      draft={draft}
      onDraftChange={onDraftChange}
      connection={connection}
      paused={paused}
      pausePending={pausePending}
      deletePending={deletePending}
      pauseError={pauseError}
      deleteError={deleteError}
      onPause={onPause}
      onResume={onResume}
      onDelete={onDelete}
    />
  );
}

function IntakeSettingsGeneralPanel({
  sourceId,
  organizationId,
  integrationId,
  resourceId,
  labelOptions,
  labelOptionsLoading,
  draft,
  onDraftChange,
  connection,
  paused,
  pausePending,
  deletePending,
  pauseError,
  deleteError,
  onPause,
  onResume,
  onDelete,
}: {
  sourceId: LineIntakeSourceId;
  organizationId?: string;
  integrationId?: string;
  resourceId?: string;
  labelOptions?: string[];
  labelOptionsLoading?: boolean;
  draft: IntakeSourceSettings;
  onDraftChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  connection?: IntakeSettingsConnection;
  paused: boolean;
  pausePending: boolean;
  deletePending: boolean;
  pauseError?: string;
  deleteError?: string;
  onPause?: () => Promise<void> | void;
  onResume?: () => Promise<void> | void;
  onDelete?: () => Promise<void> | void;
}) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const sections = useMemo(() => intakeSettingsSections(sourceId, Boolean(connection)), [connection, sourceId]);

  return (
    <div className="flex min-h-0 min-w-0 flex-1">
      <IntakeSettingsTableOfContents sections={sections} scrollContainerRef={scrollRef} />
      <IntakeSectionScroll ref={scrollRef} width={connection ? "connection" : "wide"}>
        <div className="flex flex-col gap-8">
          {connection ? (
            <IntakeConnectionFields
              sourceId={sourceId}
              organizationId={organizationId}
              integrationsBasePath={connection.integrationsBasePath}
              health={connection.health}
              binding={connection.binding}
              integrations={connection.integrations}
              integrationsLoading={connection.integrationsLoading}
              projects={connection.projects}
              projectsLoading={connection.projectsLoading}
              projectsError={connection.projectsError}
              connecting={connection.connecting}
              connectError={connection.connectError}
              onBindingChange={connection.onBindingChange}
              onConnect={connection.onConnect}
              onReconnect={connection.onReconnect}
              onRetryProjects={connection.onRetryProjects}
            />
          ) : null}
          <GitHubIntakeFilterFields
            sourceId={sourceId}
            settings={draft}
            onSettingsChange={onDraftChange}
            labelOptions={labelOptions}
            labelOptionsLoading={labelOptionsLoading}
            part="all"
            layout="grid"
          />
          <JiraIntakeFilterFields
            sourceId={sourceId}
            settings={draft}
            onSettingsChange={onDraftChange}
            organizationId={organizationId}
            integrationId={integrationId}
            projectId={resourceId}
            part="all"
            layout="grid"
          />
          <SentryIntakeFilterFields
            sourceId={sourceId}
            settings={draft}
            onSettingsChange={onDraftChange}
            layout="grid"
          />
          <ProductiveIntakeFilterFields sourceId={sourceId} settings={draft} onSettingsChange={onDraftChange} />
          <IntakeSettingsLifecycle
            sourceId={sourceId}
            paused={paused}
            pausePending={pausePending}
            deletePending={deletePending}
            pauseError={pauseError}
            deleteError={deleteError}
            onPause={onPause}
            onResume={onResume}
            onDelete={onDelete}
          />
        </div>
      </IntakeSectionScroll>
    </div>
  );
}

const IntakeSectionScroll = forwardRef<
  HTMLDivElement,
  {
    children: ReactNode;
    width?: "wide" | "connection" | "narrow";
  }
>(function IntakeSectionScroll({ children, width = "wide" }, ref) {
  return (
    <div ref={ref} className="min-h-0 flex-1 overflow-y-auto px-8 py-6">
      <div
        className={cn(
          "w-full",
          width === "wide" && "max-w-5xl",
          width === "connection" && "max-w-2xl",
          width === "narrow" && "max-w-xl",
        )}
      >
        {children}
      </div>
    </div>
  );
});

function IntakeAutomationTab({
  graph,
  canvasId,
  runHrefFor,
  editHref,
  loading,
  error,
  onRetry,
}: {
  graph?: IntakeAutomationGraph;
  canvasId?: string;
  runHrefFor?: RunsSidebarHrefForRun;
  editHref?: string;
  loading: boolean;
  error: boolean;
  onRetry?: () => void;
}) {
  if (!graph || graph.nodes.length === 0) {
    return (
      <IntakeAutomationEmpty
        message={automationEmptyMessage(loading, error)}
        onRetry={automationRetry(error, onRetry)}
        editHref={editHref}
      />
    );
  }

  return (
    <SettingsAutomationWorkspace
      graph={graph}
      testId="intake-source-automation"
      canvasId={canvasId}
      runHrefFor={runHrefFor}
      workflowNodes={graph.specNodes}
      editHref={editHref}
      editLabel={INTAKE_SETTINGS_COPY.editAutomation}
      editPlacement="belowClose"
    />
  );
}

function automationEmptyMessage(loading: boolean, error: boolean): string {
  if (loading) {
    return INTAKE_SETTINGS_COPY.automationLoading;
  }
  return error ? INTAKE_SETTINGS_COPY.automationError : INTAKE_SETTINGS_COPY.automationEmpty;
}

function automationRetry(error: boolean, onRetry: (() => void) | undefined): (() => void) | undefined {
  return error ? onRetry : undefined;
}

function IntakeAutomationEmpty({
  message,
  onRetry,
  editHref,
}: {
  message: string;
  onRetry?: () => void;
  editHref?: string;
}) {
  return (
    <section
      className="relative flex min-h-0 flex-1 flex-col items-start gap-3 px-6 py-6"
      aria-label="Automation"
      data-testid="intake-source-automation"
    >
      <p className="workspace-body-text text-muted-foreground">{message}</p>
      {onRetry ? (
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          {INTAKE_SETTINGS_COPY.retryAutomation}
        </Button>
      ) : null}
      {editHref ? (
        <SettingsAutomationCanvasEdit
          href={editHref}
          label={INTAKE_SETTINGS_COPY.editAutomation}
          testId="settings-automation-edit"
          placement="belowClose"
        />
      ) : null}
    </section>
  );
}
