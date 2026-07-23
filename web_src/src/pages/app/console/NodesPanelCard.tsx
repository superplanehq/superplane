import { useMemo, useState } from "react";
import { CircleDot, Network, Play } from "lucide-react";

import { LoadingButton } from "@/components/ui/loading-button";
import type { ConsolePanel } from "@/hooks/useCanvasData";

import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { WidgetEmptyState } from "./WidgetEmptyState";
import { useConsoleContext, resolveConsoleNode } from "./ConsoleContext";
import { isManualRunNode } from "./manualRunTriggers";
import { NodeRunConfirmDialog } from "./NodeRunConfirmDialog";
import { NodesPanelInlineRunForm } from "./NodesPanelInlineRunForm";
import { resolveStartTemplate } from "./consoleTriggerParameters";
import { useConsoleRunTrigger } from "./useConsoleRunTrigger";
import { useConsoleTriggerLock, type ConsoleTriggerLock } from "./useConsoleTriggerLock";
import type { NodesPanelContent, NodesPanelNode, NodesPanelFormMode } from "./nodesPanelContent";
import { NODES_PANEL_FORM_MODES, nodesPanelContentFromLegacyNode } from "./nodesPanelContent";
import { NodesPanelForm } from "./NodesPanelForm";

interface NodesPanelCardProps {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onEditingChange?: (editing: boolean) => void;
}

/**
 * Adaptive node panel: renders as a compact centered card when exactly one
 * node is configured (status + optional manual-run button) and as a row
 * list otherwise. Handles both the modern `type: "nodes"` shape and the
 * legacy `type: "node"` shape by folding the latter into a one-entry list.
 *
 * Resolution and trigger plumbing reuse {@link useConsoleContext} and the
 * shared {@link useConsoleRunTrigger} hook so authorization, manual-run
 * gating, and re-entry protection stay consistent everywhere the console
 * fires a trigger.
 */
export function NodesPanelCard({ panel, readOnly, onDelete, onChange, onEditingChange }: NodesPanelCardProps) {
  const [editing, setEditing] = useState(false);
  const content = normalizeContent(panel);
  const setEditingState = (next: boolean) => {
    setEditing(next);
    onEditingChange?.(next);
  };

  return (
    <>
      <TypedPanelShell
        title={content.title}
        fallbackTitle={panel.id}
        readOnly={readOnly}
        onEdit={() => setEditingState(true)}
        onDelete={onDelete}
        // Keep the body a flex column so inline prompt forms can stretch the
        // textarea to the panel height with the submit button pinned below.
        bodyClassName="flex min-h-0 flex-col overflow-hidden"
      >
        <NodesPanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<NodesPanelContent>
        open={editing}
        onOpenChange={setEditingState}
        panelId={panel.id}
        panelType="nodes"
        initialContent={content}
        onSave={(next) => onChange(next as unknown as Record<string, unknown>)}
        renderForm={({ value, onChange: setDraft }) => <NodesPanelForm value={value} onChange={setDraft} />}
      />
    </>
  );
}

function NodesPanelBody({ content }: { content: NodesPanelContent }) {
  const ctx = useConsoleContext();
  // One lock instance for the whole panel. Submissions inside
  // `useConsoleRunTrigger` are keyed by trigger node id, so two entries
  // pointing at the same trigger disable together the moment either fires —
  // per-entry lock instances would leave siblings clickable until the
  // websocket reports the run as STATE_STARTED.
  const runTriggerNodeIds = useMemo(
    () =>
      content.nodes
        .filter((entry) => entry.showRun && entry.node)
        .map((entry) => resolveConsoleNode(ctx, entry.node)?.node.id)
        .filter((id): id is string => Boolean(id)),
    [ctx, content.nodes],
  );
  const lock = useConsoleTriggerLock({ triggerNodeIds: runTriggerNodeIds });

  if (content.nodes.length === 0) {
    return (
      <WidgetEmptyState icon={Network} className="min-h-0" message="Add nodes from the editor to surface them here." />
    );
  }
  if (content.nodes.length === 1) {
    return <SingleNodeBody entry={content.nodes[0]} lock={lock} />;
  }
  return (
    <ul className="flex h-full flex-col divide-y divide-slate-100 dark:divide-gray-800" data-testid="nodes-panel-list">
      {content.nodes.map((entry, idx) => (
        <NodesPanelRow key={`${entry.node}-${idx}`} entry={entry} lock={lock} />
      ))}
    </ul>
  );
}

/**
 * Compact single-node presentation used when the panel holds exactly one
 * entry — matches the pre-merge single-node card so existing dashboards
 * keep their look after we consolidate the widget.
 */
function SingleNodeBody({ entry, lock }: { entry: NodesPanelNode; lock: ConsoleTriggerLock }) {
  const ctx = useConsoleContext();
  if (!entry.node.trim()) {
    return (
      <WidgetEmptyState
        icon={CircleDot}
        className="min-h-0"
        message="Pick a node from the editor to display it here."
      />
    );
  }
  const resolved = resolveConsoleNode(ctx, entry.node);
  const displayName = entry.label?.trim() || resolved?.label || entry.node || "—";
  const canManualRun = isManualRunNode(resolved?.node);
  const useInlineLayout = isInlineLayout(entry, canManualRun);
  const styles = singleNodeLayoutStyles(useInlineLayout);

  return (
    <div className={styles.container}>
      {entry.showNodeLabel !== false ? (
        <div className={styles.header} data-testid="node-panel-name">
          {displayName}
        </div>
      ) : null}
      {entry.description ? (
        <p className={styles.description} title={entry.description}>
          {entry.description}
        </p>
      ) : null}
      {entry.showRun && canManualRun ? (
        <div className={styles.runControl}>
          <NodesPanelRunControl
            entry={entry}
            resolved={resolved}
            lock={lock}
            testIds={{ button: "node-panel-run", dialog: "node-panel-run-dialog" }}
          />
        </div>
      ) : null}
      {!resolved ? (
        <p className="text-[13px] text-amber-600 dark:text-amber-400">
          Node {JSON.stringify(entry.node)} not found in this canvas.
        </p>
      ) : null}
    </div>
  );
}

function isInlineLayout(entry: NodesPanelNode, canManualRun: boolean): boolean {
  return entry.formMode === "inline" && Boolean(entry.showRun) && canManualRun;
}

function singleNodeLayoutStyles(useInlineLayout: boolean) {
  return {
    container: useInlineLayout
      ? "flex h-full min-h-0 flex-col items-stretch gap-3 p-4"
      : "flex h-full flex-col items-center justify-center gap-3 p-4",
    header: "shrink-0 text-[13px] font-semibold text-slate-800 dark:text-gray-100",
    description: useInlineLayout
      ? "shrink-0 text-[13px] text-slate-500 dark:text-gray-400"
      : "max-w-full truncate text-center text-[13px] text-slate-500 dark:text-gray-400",
    runControl: useInlineLayout ? "flex min-h-0 flex-1 flex-col" : undefined,
  };
}

function NodesPanelRow({ entry, lock }: { entry: NodesPanelNode; lock: ConsoleTriggerLock }) {
  const ctx = useConsoleContext();
  const configured = entry.node.trim().length > 0;
  const resolved = resolveConsoleNode(ctx, entry.node);
  const displayName = entry.label?.trim() || resolved?.label || entry.node;
  const canManualRun = isManualRunNode(resolved?.node);
  const useInlineLayout = isInlineLayout(entry, canManualRun);
  const styles = rowLayoutStyles(useInlineLayout);

  return (
    <li className={styles.row} data-testid="nodes-panel-row">
      <div className={styles.text}>
        {entry.showNodeLabel !== false ? (
          <div className={styles.name} data-testid="nodes-panel-row-name">
            {displayName}
          </div>
        ) : null}
        {entry.description ? (
          <p className={styles.description} title={entry.description}>
            {entry.description}
          </p>
        ) : null}
        {!configured ? (
          <p className="truncate text-[13px] text-slate-400 dark:text-gray-500">
            Pick a node from the editor to display it here.
          </p>
        ) : null}
        {configured && !resolved ? (
          <p className="truncate text-[13px] text-amber-600 dark:text-amber-400">
            Node {JSON.stringify(entry.node)} not found in this canvas.
          </p>
        ) : null}
      </div>
      {entry.showRun && canManualRun ? (
        <NodesPanelRunControl
          entry={entry}
          resolved={resolved}
          lock={lock}
          testIds={{ button: "nodes-panel-row-run", dialog: "nodes-panel-row-run-dialog" }}
          buttonClassName={useInlineLayout ? undefined : "shrink-0"}
        />
      ) : null}
    </li>
  );
}

function rowLayoutStyles(useInlineLayout: boolean) {
  return {
    row: useInlineLayout ? "flex flex-col gap-2 px-3 py-3" : "flex items-center gap-3 px-3 py-2",
    text: useInlineLayout ? "min-w-0" : "min-w-0 flex-1",
    name: useInlineLayout
      ? "text-[13px] font-medium text-slate-800 dark:text-gray-100"
      : "truncate text-[13px] font-medium text-slate-800 dark:text-gray-100",
    description: useInlineLayout
      ? "text-[13px] text-slate-500 dark:text-gray-400"
      : "truncate text-[13px] text-slate-500 dark:text-gray-400",
  };
}

interface RunControlTestIds {
  button: string;
  dialog: string;
}

function disabledTitleFor(reason: string | null): string | undefined {
  switch (reason) {
    case "uncommitted-canvas-changes":
      return "Commit canvas changes before running this node.";
    case "no-perm":
      return "You do not have permission to run this node";
    case "not-manual-run":
      return "Only trigger nodes with a manual run can be fired from the console.";
    case "run-in-flight":
      return "A run for this trigger is already in progress.";
    case "submitting":
      return "Submitting trigger…";
    default:
      return undefined;
  }
}

/**
 * Run button + confirm dialog for one entry. A template with input fields
 * always opens {@link NodeRunConfirmDialog} so the operator can fill them
 * in. A parameter-less template only prompts when the entry opts in via
 * `promptConfirmation`; otherwise the click fires the trigger directly.
 *
 * The button is locked while `useConsoleRunTrigger` reports a submission or
 * in-flight run for the trigger, so users can't queue duplicate runs while
 * the pipeline is still executing. The lock is shared across the panel and
 * keyed per trigger node, so sibling entries targeting the same trigger
 * lock together as soon as any of them submits.
 *
 * When `entry.formMode === "inline"` and the resolved template has input
 * fields, the parameter form is rendered inline via
 * {@link NodesPanelInlineRunForm} instead of the modal — the confirm
 * dialog is still mounted (hidden) to keep behavior identical when inline
 * mode is not applicable, e.g. schedule triggers or parameter-less
 * templates.
 */
function NodesPanelRunControl({
  entry,
  resolved,
  lock,
  testIds,
  buttonClassName,
}: {
  entry: NodesPanelNode;
  resolved: ReturnType<typeof resolveConsoleNode>;
  lock: ConsoleTriggerLock;
  testIds: RunControlTestIds;
  buttonClassName?: string;
}) {
  const { running, disabled, disabledReason, dialogOpen, setDialogOpen, handleClick, runTrigger } =
    useConsoleRunTrigger({
      resolved,
      triggerName: entry.triggerName,
      promptConfirmation: entry.promptConfirmation,
      lock,
    });

  const inlineTemplate = useInlineFormTemplate(entry, resolved);

  if (inlineTemplate) {
    return (
      <NodesPanelInlineRunForm
        template={inlineTemplate}
        onSubmit={runTrigger}
        running={running}
        disabled={disabled}
        disabledTitle={disabledTitleFor(disabledReason)}
        disabledMessage={disabledReason === "uncommitted-canvas-changes" ? disabledTitleFor(disabledReason) : undefined}
        submitLabel={entry.submitLabel}
        showFieldLabels={entry.showFieldLabels !== false}
        testIdPrefix={testIds.button}
        lock={lock}
        triggerNodeId={resolved?.node?.id}
      />
    );
  }

  return (
    <>
      <LoadingButton
        type="button"
        size="xs"
        variant="outline"
        loading={running || disabledReason === "run-in-flight"}
        loadingText="Running…"
        onClick={handleClick}
        disabled={disabled}
        title={disabledTitleFor(disabledReason)}
        data-testid={testIds.button}
        data-disabled-reason={disabledReason ?? undefined}
        className={buttonClassName}
      >
        <Play className="mr-1 h-3 w-3" />
        Run
      </LoadingButton>
      <NodeRunConfirmDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        resolved={resolved}
        templateName={entry.triggerName}
        onConfirm={runTrigger}
        confirmDisabled={disabled}
        confirmDisabledTitle={disabledTitleFor(disabledReason)}
        testId={testIds.dialog}
      />
    </>
  );
}

/**
 * True when the entry opts into inline mode AND the resolved trigger is a
 * manual-run Start with a parameterized template — the only case where the
 * inline widget can render anything useful. Returns the resolved template
 * so the caller can render it directly, or `undefined` to fall back to the
 * modal path.
 */
function useInlineFormTemplate(
  entry: NodesPanelNode,
  resolved: ReturnType<typeof resolveConsoleNode>,
): ReturnType<typeof resolveStartTemplate> {
  if (entry.formMode !== "inline") return undefined;
  if (!resolved?.node) return undefined;
  if (!isManualRunNode(resolved.node)) return undefined;
  const template = resolveStartTemplate(resolved.node, entry.triggerName);
  if (!template) return undefined;
  if ((template.parameters?.length ?? 0) === 0) return undefined;
  return template;
}

/**
 * Accept both the modern `type: "nodes"` shape and the legacy `type: "node"`
 * shape (still present in old canvases and YAML imports). Legacy content is
 * folded into a one-entry `nodes` list so the merged renderer treats both
 * uniformly.
 */
function normalizeContent(panel: ConsolePanel): NodesPanelContent {
  if (panel.type === "node") {
    return nodesPanelContentFromLegacyNode(panel.content);
  }
  return normalizeNodesContent(panel.content);
}

function normalizeNodesContent(raw: Record<string, unknown> | undefined): NodesPanelContent {
  const title = typeof raw?.title === "string" ? raw.title : "";
  const rawNodes = Array.isArray(raw?.nodes) ? raw.nodes : [];
  const nodes = rawNodes.map(normalizeEntry).filter((entry): entry is NodesPanelNode => entry != null);
  return { title, nodes };
}

function normalizeEntry(raw: unknown): NodesPanelNode | null {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;
  const obj = raw as Record<string, unknown>;
  if (typeof obj.node !== "string") return null;
  return {
    node: obj.node,
    label: typeof obj.label === "string" ? obj.label : undefined,
    description: typeof obj.description === "string" ? obj.description : undefined,
    showRun: typeof obj.showRun === "boolean" ? obj.showRun : false,
    triggerName: typeof obj.triggerName === "string" ? obj.triggerName : undefined,
    promptConfirmation: typeof obj.promptConfirmation === "boolean" ? obj.promptConfirmation : false,
    formMode: normalizeFormMode(obj.formMode),
    showNodeLabel: typeof obj.showNodeLabel === "boolean" ? obj.showNodeLabel : undefined,
    showFieldLabels: typeof obj.showFieldLabels === "boolean" ? obj.showFieldLabels : undefined,
    submitLabel: typeof obj.submitLabel === "string" ? obj.submitLabel : undefined,
  };
}

function normalizeFormMode(raw: unknown): NodesPanelFormMode | undefined {
  if (typeof raw !== "string") return undefined;
  return (NODES_PANEL_FORM_MODES as readonly string[]).includes(raw) ? (raw as NodesPanelFormMode) : undefined;
}
