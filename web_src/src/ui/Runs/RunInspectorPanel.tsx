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
import { RunInspectorChrome } from "./RunInspectorChrome";
import { RunInspectorHeader } from "./RunInspectorHeader";
import { ResizeHandle } from "./RunInspectorResize";
import { RunInspectorStepsList } from "./RunInspectorStepsList";
import { buildNodeMap, buildRunPresentation, type RUN_STATUS_META } from "./runPresentation";
import {
  buildRunInspectorNodeSections,
  findRunInspectorErrorSummaries,
  type RunInspectorCurrentUser,
  type RunInspectorErrorSummary,
  type RunInspectorNodeSection,
} from "./runNodeDetailModel";
import { useResizableInspectorWidth } from "./useResizableInspectorWidth";
import { useRunInspectorActions } from "./useRunInspectorActions";

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
  runNavigation?: { newerRunId?: string | null; olderRunId?: string | null; canNavigateOlder?: boolean } | null;
  onNavigateRun?: (runId: string) => void;
  onNavigateOlder?: () => void;
  onClose: () => void;
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
      aria-label="Run inspector"
    >
      <ResizeHandle onPointerDown={model.inspectorWidth.startResize} isResizing={model.inspectorWidth.isResizing} />
      <RunInspectorChrome
        runId={run.id}
        newerRunId={runNavigation?.newerRunId}
        olderRunId={runNavigation?.olderRunId}
        canNavigateOlder={runNavigation?.canNavigateOlder}
        onNavigateRun={onNavigateRun}
        onNavigateOlder={onNavigateOlder}
        onClose={onClose}
      />
      <RunInspectorHeader
        run={run}
        title={model.presentation.title}
        stepCount={model.sections.length || run.executions?.length || 0}
        onAction={() => (model.presentation.status === "running" ? model.actions.stop() : model.actions.rerun())}
        actionPending={model.presentation.status === "running" ? model.actions.stopPending : model.actions.rerunPending}
        actionDisabled={model.presentation.status === "running" ? model.actions.stopDisabled : !run.rootEvent?.id}
      />

      <RunInspectorContent
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
    </aside>
  );
}

function useRunInspectorPanelModel({
  canvasId,
  componentDefinitions,
  currentUser,
  onClearSelectedNode,
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
  const inspectorWidth = useResizableInspectorWidth();
  const [errorScrollRequest, setErrorScrollRequest] = useState<{ nodeId: string; requestId: number } | null>(null);
  const [selectedSectionValue, setSelectedSectionValue] = useState<string | null>(null);
  const actions = useRunInspectorActions({
    canvasId,
    run,
    onRerunCreated,
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
