import { useEffect, useMemo, useState } from "react";
import {
  type ActionsAction,
  type CanvasesCanvasRun,
  type ComponentsEdge,
  type SuperplaneMeUser,
  type SuperplaneComponentsNode as ComponentsNode,
  type TriggersTrigger,
} from "@/api-client";
import { useAccount } from "@/contexts/useAccount";
import { useEventExecutions } from "@/hooks/useCanvasData";
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
  onClose: () => void;
}

type AccountFallback = {
  id: string;
  email: string;
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
  onClose,
}: RunInspectorPanelProps) {
  const { account } = useAccount();
  const { data: me } = useMe(true, organizationId ?? null);
  const resolvedCurrentUser = useMemo(() => resolveCurrentUser(currentUser, me, account), [account, currentUser, me]);
  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [nodeMap, run]);
  const sections = useMemo(
    () =>
      buildRunInspectorNodeSections({
        run,
        executions,
        workflowNodes,
        workflowEdges,
        componentDefinitions,
        triggerDefinitions,
      }),
    [componentDefinitions, executions, run, triggerDefinitions, workflowEdges, workflowNodes],
  );
  const errorSummaries = useMemo(() => findRunInspectorErrorSummaries(sections), [sections]);
  const inspectorWidth = useResizableInspectorWidth();
  const selectedValue = selectedNodeId ?? "";
  const [pendingErrorScrollNodeId, setPendingErrorScrollNodeId] = useState<string | null>(null);
  const actions = useRunInspectorActions({
    canvasId,
    run,
    sections,
    executionsLoading: executionsQuery.isLoading,
  });

  const handleValueChange = (value: string) => {
    if (value) {
      onSelectNode(value);
      return;
    }

    onClearSelectedNode?.();
  };

  const jumpToErrorOutput = (nodeId: string) => {
    setPendingErrorScrollNodeId(nodeId);
    onSelectNode(nodeId);
  };

  useEffect(() => {
    if (!pendingErrorScrollNodeId || selectedNodeId !== pendingErrorScrollNodeId) return;

    const frame = window.requestAnimationFrame(() => {
      const errorOutput = document.querySelector(`[data-run-error-output-node-id="${pendingErrorScrollNodeId}"]`);
      errorOutput?.scrollIntoView({ block: "center", behavior: "smooth" });
      setPendingErrorScrollNodeId(null);
    });

    return () => window.cancelAnimationFrame(frame);
  }, [pendingErrorScrollNodeId, selectedNodeId]);

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
      <RunInspectorChrome onClose={onClose} />
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
        selectedValue={selectedValue}
        componentIconMap={componentIconMap}
        organizationId={organizationId}
        onValueChange={handleValueChange}
        onJumpToError={jumpToErrorOutput}
        onRerun={actions.rerun}
        rerunPending={actions.rerunPending}
        actions={actions}
        currentUser={resolvedCurrentUser}
      />
    </aside>
  );
}

function resolveCurrentUser(
  currentUser: RunInspectorCurrentUser | undefined,
  me: SuperplaneMeUser | null | undefined,
  account: AccountFallback,
): RunInspectorCurrentUser | undefined {
  if (currentUser) return currentUser;
  if (me) return { id: me.id ?? "", email: me.email ?? "", roles: me.roles, groups: me.groups };
  if (account) return { id: account.id, email: account.email, groups: account.groups };
  return undefined;
}
