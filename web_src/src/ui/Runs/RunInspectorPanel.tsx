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
import { buildNodeMap, buildRunPresentation } from "./runPresentation";
import {
  buildRunInspectorNodeSections,
  findRunInspectorErrorSummaries,
  type RunInspectorCurrentUser,
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

export function RunInspectorPanel({
  canvasId,
  organizationId,
  run,
  workflowNodes,
  workflowEdges,
  componentDefinitions,
  triggerDefinitions,
  componentIconMap = {},
  currentUser,
  selectedNodeId = null,
  onSelectNode,
  onClearSelectedNode,
  onEditNode,
  onRerunCreated,
  runNavigation,
  onNavigateRun,
  onNavigateOlder,
  onClose,
}: RunInspectorPanelProps) {
  const { account } = useAccount();
  const { data: me } = useMe(true, organizationId ?? null);
  const resolvedCurrentUser = useMemo(() => resolveCurrentUser(currentUser, me, account), [account, currentUser, me]);
  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const runVersionQuery = useCanvasVersion(organizationId ?? "", canvasId, run.versionId ?? "", Boolean(run.versionId));
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);
  const shouldUseRunVersion = Boolean(run.versionId);
  const versionWorkflowNodes = runVersionQuery.data?.spec?.nodes;
  const versionWorkflowEdges = runVersionQuery.data?.spec?.edges;
  const hasRunVersionSpec = shouldUseRunVersion && hasWorkflowNodes(versionWorkflowNodes);
  const inspectorWorkflowNodes = useMemo(
    () => selectInspectorWorkflowNodes(shouldUseRunVersion, hasRunVersionSpec, versionWorkflowNodes, workflowNodes),
    [hasRunVersionSpec, shouldUseRunVersion, versionWorkflowNodes, workflowNodes],
  );
  const inspectorWorkflowEdges = hasRunVersionSpec ? versionWorkflowEdges : workflowEdges;
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
    sections,
    executionsLoading: executionsQuery.isLoading,
    onRerunCreated,
  });
  const accordionValue = useMemo(
    () => resolveSelectedSectionValue(sections, selectedNodeId, selectedSectionValue),
    [sections, selectedNodeId, selectedSectionValue],
  );

  const handleValueChange = (value: string) => {
    if (value) {
      const section = sections.find((item) => item.sectionValue === value);
      setSelectedSectionValue(value);
      return onSelectNode(section?.nodeId ?? value);
    }

    setSelectedSectionValue(null);
    onClearSelectedNode?.();
  };

  const jumpToErrorOutput = (nodeId: string) => {
    setErrorScrollRequest({ nodeId, requestId: Date.now() });
    onSelectNode(nodeId);
  };

  return (
    <aside
      className={cn(
        "relative z-20 flex h-full shrink-0 flex-col border-l bg-white shadow-sm dark:bg-gray-950",
        appDarkModeClasses.sidebarEdge,
      )}
      style={{ width: inspectorWidth.width }}
      data-testid="run-inspector-panel"
      aria-label="Run inspector"
    >
      <ResizeHandle onPointerDown={inspectorWidth.startResize} isResizing={inspectorWidth.isResizing} />
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
        title={presentation.title}
        stepCount={sections.length || run.executions?.length || 0}
        onAction={() => (presentation.status === "running" ? actions.stop() : actions.rerun())}
        actionPending={presentation.status === "running" ? actions.stopPending : actions.rerunPending}
        actionDisabled={presentation.status === "running" ? actions.stopDisabled : !run.rootEvent?.id}
      />

      <RunInspectorStepsList
        errorSummaries={errorSummaries}
        status={presentation.status}
        sections={sections}
        isLoading={executionsQuery.isLoading}
        selectedValue={accordionValue}
        componentIconMap={componentIconMap}
        organizationId={organizationId}
        canShowExpressionTemplates={hasRunVersionSpec}
        onValueChange={handleValueChange}
        onJumpToError={jumpToErrorOutput}
        onRerun={actions.rerun}
        onEditNode={onEditNode}
        rerunPending={actions.rerunPending}
        actions={actions}
        currentUser={resolvedCurrentUser}
        errorScrollRequest={errorScrollRequest}
        onErrorScrolled={() => setErrorScrollRequest(null)}
      />
    </aside>
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
