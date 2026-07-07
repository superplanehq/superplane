import { useState } from "react";
import { CircleDot, Network, Play } from "lucide-react";

import { LoadingButton } from "@/components/ui/loading-button";
import type { ConsolePanel } from "@/hooks/useCanvasData";

import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { WidgetEmptyState } from "./WidgetEmptyState";
import { isManualRunNode, useConsoleContext, resolveConsoleNode } from "./ConsoleContext";
import { NodeRunConfirmDialog } from "./NodeRunConfirmDialog";
import { useConsoleRunTrigger } from "./useConsoleRunTrigger";
import type { NodesPanelContent, NodesPanelNode } from "./nodesPanelContent";
import { nodesPanelContentFromLegacyNode } from "./nodesPanelContent";
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
  if (content.nodes.length === 0) {
    return (
      <WidgetEmptyState icon={Network} className="min-h-0" message="Add nodes from the editor to surface them here." />
    );
  }
  if (content.nodes.length === 1) {
    return <SingleNodeBody entry={content.nodes[0]} entryIndex={0} />;
  }
  return (
    <ul className="flex h-full flex-col divide-y divide-slate-100 dark:divide-gray-800" data-testid="nodes-panel-list">
      {content.nodes.map((entry, idx) => (
        <NodesPanelRow key={`${entry.node}-${idx}`} entry={entry} entryIndex={idx} />
      ))}
    </ul>
  );
}

/**
 * Compact single-node presentation used when the panel holds exactly one
 * entry — matches the pre-merge single-node card so existing dashboards
 * keep their look after we consolidate the widget.
 */
function SingleNodeBody({ entry, entryIndex }: { entry: NodesPanelNode; entryIndex: number }) {
  const ctx = useConsoleContext();
  if (!entry.node) {
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
  const canManualRun = isManualRunNode(ctx, resolved?.node);

  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 p-4">
      <div className="text-[13px] font-semibold text-slate-800 dark:text-gray-100" data-testid="node-panel-name">
        {displayName}
      </div>
      {entry.showRun && canManualRun ? (
        <NodesPanelRunControl
          entry={entry}
          entryIndex={entryIndex}
          resolved={resolved}
          testIds={{ button: "node-panel-run", dialog: "node-panel-run-dialog" }}
        />
      ) : null}
      {!resolved && entry.node ? (
        <p className="text-[13px] text-amber-600 dark:text-amber-400">
          Node {JSON.stringify(entry.node)} not found in this canvas.
        </p>
      ) : null}
    </div>
  );
}

function NodesPanelRow({ entry, entryIndex }: { entry: NodesPanelNode; entryIndex: number }) {
  const ctx = useConsoleContext();
  const resolved = resolveConsoleNode(ctx, entry.node);
  const displayName = entry.label?.trim() || resolved?.label || entry.node;
  const canManualRun = isManualRunNode(ctx, resolved?.node);

  return (
    <li className="flex items-center gap-3 px-3 py-2" data-testid="nodes-panel-row">
      <div className="min-w-0 flex-1">
        <div
          className="truncate text-[13px] font-medium text-slate-800 dark:text-gray-100"
          data-testid="nodes-panel-row-name"
        >
          {displayName}
        </div>
        {entry.description ? (
          <p className="truncate text-[13px] text-slate-500 dark:text-gray-400" title={entry.description}>
            {entry.description}
          </p>
        ) : null}
        {!resolved ? (
          <p className="truncate text-[13px] text-amber-600 dark:text-amber-400">
            Node {JSON.stringify(entry.node)} not found in this canvas.
          </p>
        ) : null}
      </div>
      {entry.showRun && canManualRun ? (
        <NodesPanelRunControl
          entry={entry}
          entryIndex={entryIndex}
          resolved={resolved}
          testIds={{ button: "nodes-panel-row-run", dialog: "nodes-panel-row-run-dialog" }}
          buttonClassName="shrink-0"
        />
      ) : null}
    </li>
  );
}

interface RunControlTestIds {
  button: string;
  dialog: string;
}

function disabledTitleFor(reason: string | null): string | undefined {
  switch (reason) {
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
 * the pipeline is still executing.
 */
function NodesPanelRunControl({
  entry,
  entryIndex,
  resolved,
  testIds,
  buttonClassName,
}: {
  entry: NodesPanelNode;
  entryIndex: number;
  resolved: ReturnType<typeof resolveConsoleNode>;
  testIds: RunControlTestIds;
  buttonClassName?: string;
}) {
  const { running, disabled, disabledReason, dialogOpen, setDialogOpen, handleClick, runTrigger } =
    useConsoleRunTrigger({
      resolved,
      triggerName: entry.triggerName,
      promptConfirmation: entry.promptConfirmation,
      lockKey: `nodes-panel-entry:${entryIndex}:${resolved?.node?.id ?? entry.node}`,
    });

  return (
    <>
      <LoadingButton
        type="button"
        size="xs"
        variant="outline"
        loading={running || disabledReason === "run-in-flight"}
        loadingText={disabledReason === "run-in-flight" ? "Running…" : "Running…"}
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
        testId={testIds.dialog}
      />
    </>
  );
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
  };
}
