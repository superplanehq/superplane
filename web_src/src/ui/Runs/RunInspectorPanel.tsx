import { Loader2 } from "lucide-react";
import { useMemo, useState } from "react";
import {
  type ActionsAction,
  type CanvasesCanvasRun,
  type ComponentsEdge,
  type SuperplaneMeUser,
  type SuperplaneComponentsNode as ComponentsNode,
  type TriggersTrigger,
} from "@/api-client";
import { useAccount } from "@/contexts/useAccount";
import { useCanvasVersion, useEventExecutions } from "@/hooks/useCanvasData";
import { useMe } from "@/hooks/useMe";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { FactorySidebarHeading } from "@/ui/factoryNodeChrome";
import { FactorySidebarCloseButton, FactorySidebarCloseRow } from "./FactorySidebarClose";
import { RunInspectorChrome } from "./RunInspectorChrome";
import { RunInspectorHeader } from "./RunInspectorHeader";
import { RunInspectorNodeActions } from "./RunInspectorNodeAccordion";
import { ResizeHandle } from "./RunInspectorResize";
import { RunInspectorStepTimeline } from "./RunInspectorStepTimeline";
import { RunInspectorStepsList } from "./RunInspectorStepsList";
import { RunErrorsCard } from "./RunErrorsCard";
import { buildNodeMap, buildRunPresentation, type RUN_STATUS_META } from "./runPresentation";
import { normalizeRunErrors } from "./runErrors";
import type { RunInspectorCurrentUser, RunInspectorErrorSummary, RunInspectorNodeSection } from "./types";
import { useResizableInspectorWidth } from "./useResizableInspectorWidth";
import { useRunInspectorActions } from "./useRunInspectorActions";
import { buildRunInspectorNodeSections, findRunInspectorErrorSummaries } from "./runNodeDetailModel";

export interface RunInspectorPanelProps {
  canvasId: string;
  organizationId?: string;
  run: CanvasesCanvasRun;
  workflowNodes: ComponentsNode[];
  workflowEdges?: ComponentsEdge[];
  componentDefinitions?: ActionsAction[];
  triggerDefinitions?: TriggersTrigger[];
  componentIconMap?: Record<string, string>;
  currentUser?: RunInspectorCurrentUser;
  selectedNodeId?: string | null;
  onSelectNode: (nodeId: string) => void;
  onClearSelectedNode?: () => void;
  onEditNode?: (nodeId: string) => void;
  onRerunCreated?: (eventId: string) => void | Promise<void>;
  onReplayCreated?: (runId: string) => void;
  runNavigation?: { newerRunId?: string | null; olderRunId?: string | null; canNavigateOlder?: boolean } | null;
  onNavigateRun?: (runId: string) => void;
  onNavigateOlder?: () => void;
  onClose: () => void;
  /** Factory canvas/automation: Close-only chrome (no newer/older/copy link). */
  factoryContext?: boolean;
}

type AccountFallback = {
  id: string;
  email: string;
  roles?: string[];
  groups?: string[];
} | null;

export function RunInspectorPanel(props: RunInspectorPanelProps) {
  const {
    componentIconMap = {},
    factoryContext = false,
    onClose,
    onEditNode,
    onNavigateOlder,
    onNavigateRun,
    organizationId,
    run,
    runNavigation,
  } = props;
  const model = useRunInspectorPanelModel(props);

  return (
    <aside
      className={cn(
        "relative z-20 flex h-full shrink-0 flex-col border-l bg-white shadow-sm dark:bg-gray-950",
        appDarkModeClasses.sidebarEdge,
      )}
      style={{ width: model.inspectorWidth.width }}
      data-testid="run-inspector-panel"
      data-factory-context={factoryContext ? "true" : undefined}
      aria-label="Run inspector"
    >
      <ResizeHandle onPointerDown={model.inspectorWidth.startResize} isResizing={model.inspectorWidth.isResizing} />
      {!factoryContext ? (
        <RunInspectorChrome
          runId={run.id}
          newerRunId={runNavigation?.newerRunId}
          olderRunId={runNavigation?.olderRunId}
          canNavigateOlder={runNavigation?.canNavigateOlder}
          onNavigateRun={onNavigateRun}
          onNavigateOlder={onNavigateOlder}
          onClose={onClose}
        />
      ) : null}
      <RunInspectorPanelBody
        factoryContext={factoryContext}
        organizationId={organizationId}
        run={run}
        model={model}
        componentIconMap={componentIconMap}
        onEditNode={onEditNode}
        onClose={onClose}
      />
    </aside>
  );
}

function isStopOrCancelStatus(status: string) {
  return status === "running" || status === "cancelling";
}

function RunInspectorPanelBody({
  factoryContext,
  organizationId,
  run,
  model,
  componentIconMap,
  onEditNode,
  onClose,
}: {
  factoryContext: boolean;
  organizationId?: string;
  run: CanvasesCanvasRun;
  model: ReturnType<typeof useRunInspectorPanelModel>;
  componentIconMap: Record<string, string>;
  onEditNode?: (nodeId: string) => void;
  onClose: () => void;
}) {
  if (factoryContext) {
    return (
      <FactoryNodeDetailBody
        organizationId={organizationId}
        sections={model.sections}
        isLoading={model.isStepsLoading}
        selectedValue={model.accordionValue}
        componentIconMap={componentIconMap}
        canShowExpressionTemplates={model.hasRunVersionSpec}
        onEditNode={onEditNode}
        onClose={onClose}
        actions={model.actions}
        currentUser={model.resolvedCurrentUser}
        errorScrollRequest={model.errorScrollRequest}
        onErrorScrolled={model.clearErrorScrollRequest}
        runErrors={model.runErrors}
      />
    );
  }

  const stopping = isStopOrCancelStatus(model.presentation.status);
  return (
    <>
      <RunInspectorHeader
        run={run}
        title={model.presentation.title}
        stepCount={model.sections.length || run.executions?.length || 0}
        onAction={() => (stopping ? model.actions.stop() : model.actions.rerun())}
        actionPending={stopping ? model.actions.stopPending : model.actions.rerunPending}
        actionDisabled={stopping ? model.actions.stopDisabled : !run.rootEvent?.id}
      />
      <RunInspectorContent
        runErrors={model.runErrors}
        errorSummaries={model.errorSummaries}
        status={model.presentation.status}
        sections={model.sections}
        isLoading={model.isStepsLoading}
        selectedValue={model.accordionValue}
        componentIconMap={componentIconMap}
        organizationId={organizationId}
        canShowExpressionTemplates={model.hasRunVersionSpec}
        onValueChange={model.handleValueChange}
        onJumpToError={model.jumpToErrorOutput}
        onRerun={model.actions.rerun}
        onEditNode={onEditNode}
        rerunPending={model.actions.rerunPending}
        actions={model.actions}
        currentUser={model.resolvedCurrentUser}
        errorScrollRequest={model.errorScrollRequest}
        onErrorScrolled={model.clearErrorScrollRequest}
      />
    </>
  );
}

/**
 * Factory embed: title row (label · name + Close) + node actions +
 * that node's timeline only. No run chrome, run header, or Rerun.
 */
function FactoryNodeDetailBody({
  organizationId,
  sections,
  isLoading,
  selectedValue,
  componentIconMap,
  canShowExpressionTemplates,
  onEditNode,
  onClose,
  actions,
  currentUser,
  errorScrollRequest,
  onErrorScrolled,
  runErrors,
}: {
  organizationId?: string;
  sections: RunInspectorNodeSection[];
  isLoading: boolean;
  selectedValue: string;
  componentIconMap: Record<string, string>;
  canShowExpressionTemplates: boolean;
  onEditNode?: (nodeId: string) => void;
  onClose: () => void;
  actions: ReturnType<typeof useRunInspectorActions>;
  currentUser: RunInspectorCurrentUser | undefined;
  errorScrollRequest: { nodeId: string; requestId: number } | null;
  onErrorScrolled: () => void;
  runErrors: string[];
}) {
  const selectedSection = sections.find((section) => section.sectionValue === selectedValue) ?? null;
  const runErrorsCard = runErrors.length > 0 ? <RunErrorsCard errors={runErrors} /> : null;

  if (isLoading && !selectedSection) {
    return (
      <div className="flex min-h-0 flex-1 flex-col" data-testid="factory-run-node-detail">
        <FactorySidebarCloseRow onClose={onClose} />
        {runErrorsCard ? <div className="px-3 pt-3">{runErrorsCard}</div> : null}
        <div className="flex min-h-0 flex-1 items-center justify-center gap-2 px-4 py-8 text-sm text-slate-500 dark:text-gray-400">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading run steps...
        </div>
      </div>
    );
  }

  if (!selectedSection) {
    return (
      <div className="flex min-h-0 flex-1 flex-col" data-testid="factory-run-node-detail">
        <FactorySidebarCloseRow onClose={onClose} />
        {runErrorsCard ? <div className="px-3 pt-3">{runErrorsCard}</div> : null}
        <div className="px-4 py-8 text-sm text-slate-500 dark:text-gray-400" data-testid="factory-run-inspector-empty">
          Select a node to inspect this run.
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="factory-run-node-detail">
      <div className="flex shrink-0 items-center gap-2 border-b border-slate-950/10 px-3 py-2.5 dark:border-gray-800">
        <FactorySidebarCloseButton onClose={onClose} />
        <FactorySidebarHeading
          componentLabel={selectedSection.componentLabel}
          nodeName={selectedSection.nodeName}
          testId="factory-run-node-title"
        />
        <RunInspectorNodeActions
          section={selectedSection}
          actions={actions}
          currentUser={currentUser}
          className="pl-0"
        />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto bg-slate-50 px-3 py-3 dark:bg-gray-950">
        {runErrorsCard ? <div className="mb-3">{runErrorsCard}</div> : null}
        {selectedSection.isQueued ? (
          <p className="text-sm text-slate-500 dark:text-gray-400">This step is queued.</p>
        ) : (
          <RunInspectorStepTimeline
            section={selectedSection}
            componentIconMap={componentIconMap}
            organizationId={organizationId}
            canShowExpressionTemplates={canShowExpressionTemplates}
            onEditNode={onEditNode}
            errorScrollRequestId={
              errorScrollRequest?.nodeId === selectedSection.nodeId ? errorScrollRequest.requestId : null
            }
            onErrorScrolled={onErrorScrolled}
          />
        )}
      </div>
    </div>
  );
}

function useRunInspectorPanelModel({
  canvasId,
  componentDefinitions,
  currentUser,
  onClearSelectedNode,
  onReplayCreated,
  onRerunCreated,
  onSelectNode,
  organizationId,
  run,
  selectedNodeId = null,
  triggerDefinitions,
  workflowEdges,
  workflowNodes,
}: RunInspectorPanelProps) {
  const { account } = useAccount();
  const { data: me } = useMe(true, organizationId ?? null);
  const executionsQuery = useEventExecutions(canvasId, run.rootEvent?.id || null);
  const runVersionQuery = useCanvasVersion(organizationId ?? "", canvasId, run.versionId ?? "", Boolean(run.versionId));
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);
  const shouldUseRunVersion = Boolean(run.versionId);
  const versionWorkflowNodes = runVersionQuery.data?.spec?.nodes;
  const hasRunVersionSpec = shouldUseRunVersion && hasWorkflowNodes(versionWorkflowNodes);
  const inspectorWorkflowNodes = useMemo(
    () => selectInspectorWorkflowNodes(shouldUseRunVersion, hasRunVersionSpec, versionWorkflowNodes, workflowNodes),
    [hasRunVersionSpec, shouldUseRunVersion, versionWorkflowNodes, workflowNodes],
  );
  const inspectorWorkflowEdges = hasRunVersionSpec ? runVersionQuery.data?.spec?.edges : workflowEdges;
  const nodeMap = useMemo(() => buildNodeMap(inspectorWorkflowNodes), [inspectorWorkflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [nodeMap, run]);
  const sections = useMemo(
    () =>
      buildRunInspectorNodeSections({
        run,
        executions,
        workflowNodes: inspectorWorkflowNodes,
        workflowEdges: inspectorWorkflowEdges,
        componentDefinitions,
        triggerDefinitions,
      }),
    [componentDefinitions, executions, run, triggerDefinitions, inspectorWorkflowEdges, inspectorWorkflowNodes],
  );
  const errorSummaries = useMemo(() => findRunInspectorErrorSummaries(sections), [sections]);
  const runErrors = useMemo(() => normalizeRunErrors(run.errors), [run.errors]);
  const inspectorWidth = useResizableInspectorWidth();
  const [errorScrollRequest, setErrorScrollRequest] = useState<{ nodeId: string; requestId: number } | null>(null);
  const [selectedSectionValue, setSelectedSectionValue] = useState<string | null>(null);
  const actions = useRunInspectorActions({
    canvasId,
    run,
    sections,
    executionsLoading: executionsQuery.isLoading,
    onRerunCreated,
    onReplayCreated,
  });
  const accordionValue = useMemo(
    () => resolveSelectedSectionValue(sections, selectedNodeId, selectedSectionValue),
    [sections, selectedNodeId, selectedSectionValue],
  );
  const isStepsLoading = executionsQuery.isLoading && !sections.some((section) => section.isQueued);
  const resolvedCurrentUser = resolveCurrentUser(currentUser, me, account);

  return {
    accordionValue,
    actions,
    clearErrorScrollRequest: () => setErrorScrollRequest(null),
    errorScrollRequest,
    errorSummaries,
    runErrors,
    handleValueChange: (value: string) =>
      selectRunInspectorSection(value, sections, setSelectedSectionValue, onSelectNode, onClearSelectedNode),
    hasRunVersionSpec,
    inspectorWidth,
    isStepsLoading,
    jumpToErrorOutput: (nodeId: string) => {
      setErrorScrollRequest({ nodeId, requestId: Date.now() });
      setSelectedSectionValue(null);
      onSelectNode(nodeId);
    },
    presentation,
    resolvedCurrentUser,
    sections,
  };
}

function RunInspectorContent({
  runErrors,
  errorSummaries,
  status,
  sections,
  isLoading,
  selectedValue,
  componentIconMap,
  organizationId,
  canShowExpressionTemplates,
  onValueChange,
  onJumpToError,
  onRerun,
  onEditNode,
  rerunPending,
  actions,
  currentUser,
  errorScrollRequest,
  onErrorScrolled,
}: {
  runErrors: string[];
  errorSummaries: RunInspectorErrorSummary[];
  status: keyof typeof RUN_STATUS_META;
  sections: RunInspectorNodeSection[];
  isLoading: boolean;
  selectedValue: string;
  componentIconMap: Record<string, string>;
  organizationId?: string;
  canShowExpressionTemplates: boolean;
  onValueChange: (value: string) => void;
  onJumpToError: (nodeId: string) => void;
  onRerun: () => void;
  onEditNode?: (nodeId: string) => void;
  rerunPending: boolean;
  actions: ReturnType<typeof useRunInspectorActions>;
  currentUser: RunInspectorCurrentUser | undefined;
  errorScrollRequest: { nodeId: string; requestId: number } | null;
  onErrorScrolled: () => void;
}) {
  return (
    <RunInspectorStepsList
      runErrors={runErrors}
      errorSummaries={errorSummaries}
      status={status}
      sections={sections}
      isLoading={isLoading}
      selectedValue={selectedValue}
      componentIconMap={componentIconMap}
      organizationId={organizationId}
      canShowExpressionTemplates={canShowExpressionTemplates}
      onValueChange={onValueChange}
      onJumpToError={onJumpToError}
      onRerun={onRerun}
      onEditNode={onEditNode}
      rerunPending={rerunPending}
      actions={actions}
      currentUser={currentUser}
      errorScrollRequest={errorScrollRequest}
      onErrorScrolled={onErrorScrolled}
    />
  );
}

function resolveSelectedSectionValue(
  sections: ReturnType<typeof buildRunInspectorNodeSections>,
  selectedNodeId: string | null,
  selectedSectionValue: string | null,
): string {
  if (!selectedNodeId) return "";

  const selectedSection = selectedSectionValue
    ? sections.find((section) => section.sectionValue === selectedSectionValue && section.nodeId === selectedNodeId)
    : undefined;
  if (selectedSection) return selectedSection.sectionValue;

  return sections.find((section) => section.nodeId === selectedNodeId)?.sectionValue ?? "";
}

function selectRunInspectorSection(
  value: string,
  sections: RunInspectorNodeSection[],
  setSelectedSectionValue: (value: string | null) => void,
  onSelectNode: (nodeId: string) => void,
  onClearSelectedNode: (() => void) | undefined,
) {
  if (!value) {
    setSelectedSectionValue(null);
    onClearSelectedNode?.();
    return;
  }

  const section = sections.find((item) => item.sectionValue === value);
  setSelectedSectionValue(value);
  onSelectNode(section?.nodeId ?? value);
}

function selectInspectorWorkflowNodes(
  shouldUseRunVersion: boolean,
  hasRunVersionSpec: boolean,
  versionWorkflowNodes: ComponentsNode[] | undefined,
  workflowNodes: ComponentsNode[],
): ComponentsNode[] {
  if (!shouldUseRunVersion) return workflowNodes;
  if (hasRunVersionSpec) return versionWorkflowNodes ?? [];
  return workflowNodes.map((node) => ({ ...node, configuration: undefined }));
}

function hasWorkflowNodes(nodes: ComponentsNode[] | undefined): boolean {
  return Boolean(nodes?.length);
}

function resolveCurrentUser(
  currentUser: RunInspectorCurrentUser | undefined,
  me: SuperplaneMeUser | null | undefined,
  account: AccountFallback,
): RunInspectorCurrentUser | undefined {
  if (currentUser) return currentUser;
  if (me) return { id: me.id ?? "", email: me.email ?? "", roles: me.roles, groups: me.groups };
  if (account) return { id: account.id, email: account.email, roles: account.roles, groups: account.groups };
  return undefined;
}
