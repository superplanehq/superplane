import { useId, useState } from "react";
import { CircleDot, Play } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";
import { Checkbox } from "@/ui/checkbox";
import type { DashboardPanel } from "@/hooks/useCanvasData";

import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { useDashboardContext, resolveDashboardNode, type DashboardNodeStatus } from "./DashboardContext";
import { DASHBOARD_TRIGGER_NODE_EVENT } from "./dashboardEvents";
import { NodeRunConfirmDialog } from "./NodeRunConfirmDialog";
import type { NodePanelContent } from "./panelTypes";

const STATUS_CLASS: Record<DashboardNodeStatus, string> = {
  passed: "bg-emerald-100 text-emerald-700 ring-emerald-300",
  failed: "bg-red-100 text-red-700 ring-red-300",
  cancelled: "bg-slate-200 text-slate-600 ring-slate-300",
  running: "bg-sky-100 text-sky-700 ring-sky-300",
  pending: "bg-amber-100 text-amber-700 ring-amber-300",
  skipped: "bg-slate-100 text-slate-500 ring-slate-300",
  unknown: "bg-slate-100 text-slate-500 ring-slate-300",
};

const STATUS_LABEL: Record<DashboardNodeStatus, string> = {
  passed: "Passed",
  failed: "Failed",
  cancelled: "Cancelled",
  running: "Running",
  pending: "Pending",
  skipped: "Skipped",
  unknown: "Unknown",
};

interface NodePanelCardProps {
  panel: DashboardPanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
}

/**
 * Single-node panel: status badge + node name + optional manual-run button.
 * Resolves the node reference (id or name) through {@link DashboardContext}.
 */
export function NodePanelCard({ panel, readOnly, onDelete, onChange }: NodePanelCardProps) {
  const [editing, setEditing] = useState(false);
  const content = normalizeContent(panel.content);

  return (
    <>
      <TypedPanelShell
        title={content.title}
        fallbackTitle={panel.id}
        readOnly={readOnly}
        onEdit={() => setEditing(true)}
        onDelete={onDelete}
      >
        <NodePanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<NodePanelContent>
        open={editing}
        onOpenChange={setEditing}
        panelId={panel.id}
        panelType="node"
        initialContent={content}
        onSave={(next) => onChange(next as unknown as Record<string, unknown>)}
        renderForm={({ value, onChange: setDraft }) => <NodePanelForm value={value} onChange={setDraft} />}
      />
    </>
  );
}

function NodePanelBody({ content }: { content: NodePanelContent }) {
  const ctx = useDashboardContext();
  if (!content.node) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-1.5 p-4 text-center text-slate-400">
        <CircleDot className="h-5 w-5" aria-hidden />
        <p className="text-xs">Pick a node from the editor to display its status here.</p>
      </div>
    );
  }
  const resolved = resolveDashboardNode(ctx, content.node);
  const status: DashboardNodeStatus = resolveStatus(ctx, resolved?.node.id);

  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 p-4">
      <span
        className={cn(
          "inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-[11px] font-medium ring-1 ring-inset",
          STATUS_CLASS[status],
        )}
        data-testid="node-panel-status"
      >
        <CircleDot className="h-3 w-3" aria-hidden />
        {STATUS_LABEL[status]}
      </span>
      <div className="text-sm font-semibold text-slate-800" data-testid="node-panel-name">
        {resolved?.label ?? content.node ?? "—"}
      </div>
      {content.showRun ? <NodePanelRunControl content={content} resolved={resolved} /> : null}
      {!resolved && content.node ? (
        <p className="text-xs text-amber-600">Node {JSON.stringify(content.node)} not found in this canvas.</p>
      ) : null}
    </div>
  );
}

/**
 * Run button + confirm dialog for the single-node panel. Mirrors the Key
 * Nodes panel: the click always opens {@link NodeRunConfirmDialog} (even
 * when the resolved Start template has no parameters) so the operator gets
 * a payload preview and a chance to cancel before the trigger fires.
 */
function NodePanelRunControl({
  content,
  resolved,
}: {
  content: NodePanelContent;
  resolved: ReturnType<typeof resolveDashboardNode>;
}) {
  const ctx = useDashboardContext();
  const [open, setOpen] = useState(false);
  const canRun = (ctx?.canRunNodes ?? false) && Boolean(resolved);
  return (
    <>
      <Button
        type="button"
        size="sm"
        variant="outline"
        onClick={() => setOpen(true)}
        disabled={!canRun}
        title={canRun ? undefined : "You do not have permission to run this node"}
        data-testid="node-panel-run"
      >
        <Play className="mr-1 h-3.5 w-3.5" />
        Run
      </Button>
      <NodeRunConfirmDialog
        open={open}
        onOpenChange={setOpen}
        resolved={resolved}
        templateName={content.triggerName}
        onConfirm={async (parameters) => {
          if (!resolved?.node?.id) return;
          confirmTriggerNode(ctx, resolved.node.id, content.triggerName, parameters);
        }}
        testId="node-panel-run-dialog"
      />
    </>
  );
}

function resolveStatus(ctx: ReturnType<typeof useDashboardContext>, nodeId: string | undefined): DashboardNodeStatus {
  if (!nodeId) return "unknown";
  return ctx?.nodeStatuses?.[nodeId] ?? "unknown";
}

/**
 * Fire the configured trigger after the user confirms in the run dialog.
 * Forwards the pre-built parameters object (already in the
 * `{ template, ...values }` shape that the backend expects) so the host
 * skips the default `mergeTriggerParameters` step it would normally apply
 * when no parameters are provided. Falls back to the legacy window event
 * when there is no live dashboard context.
 */
function confirmTriggerNode(
  ctx: ReturnType<typeof useDashboardContext>,
  nodeId: string,
  triggerName: string | undefined,
  parameters: Record<string, unknown>,
) {
  if (ctx?.onTriggerNode) {
    ctx.onTriggerNode(nodeId, {
      hookName: "run",
      templateName: triggerName,
      parameters,
    });
    return;
  }
  window.dispatchEvent(
    new CustomEvent(DASHBOARD_TRIGGER_NODE_EVENT, {
      detail: { nodeId, triggerName, parameters },
    }),
  );
}

function NodePanelForm({ value, onChange }: { value: NodePanelContent; onChange: (next: NodePanelContent) => void }) {
  const ctx = useDashboardContext();
  const nodes = ctx?.nodes ?? [];
  const showRunId = useId();
  return (
    <div className="space-y-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Title (optional)</Label>
        <Input
          value={value.title ?? ""}
          onChange={(e) => onChange({ ...value, title: e.target.value })}
          placeholder="Defaults to panel id"
        />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Node</Label>
        <Select value={value.node} onValueChange={(v) => onChange({ ...value, node: v })}>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Select a node" />
          </SelectTrigger>
          <SelectContent>
            {nodes.length === 0 ? (
              <SelectItem value="__none__" disabled>
                No nodes in this canvas
              </SelectItem>
            ) : (
              nodes.map((n) => (
                <SelectItem key={n.id} value={n.name || n.id || ""}>
                  {n.name || n.id}
                </SelectItem>
              ))
            )}
          </SelectContent>
        </Select>
      </div>
      <div className="flex items-center gap-2">
        <Checkbox
          id={showRunId}
          checked={Boolean(value.showRun)}
          onCheckedChange={(checked) => onChange({ ...value, showRun: checked === true })}
          className="border-slate-300 data-[state=checked]:border-sky-600 data-[state=checked]:bg-sky-600"
        />
        <Label htmlFor={showRunId} className="text-xs text-slate-700">
          Show a manual "Run" button (requires run permission).
        </Label>
      </div>
      {value.showRun ? (
        <div className="space-y-1.5">
          <Label className="text-xs font-medium text-slate-600">Trigger template (optional)</Label>
          <Input
            value={value.triggerName ?? ""}
            onChange={(e) => onChange({ ...value, triggerName: e.target.value || undefined })}
            placeholder="e.g. manual"
          />
        </div>
      ) : null}
    </div>
  );
}

function normalizeContent(raw: Record<string, unknown> | undefined): NodePanelContent {
  return {
    title: typeof raw?.title === "string" ? raw.title : "",
    node: typeof raw?.node === "string" ? raw.node : "",
    showRun: typeof raw?.showRun === "boolean" ? raw.showRun : false,
    triggerName: typeof raw?.triggerName === "string" ? raw.triggerName : undefined,
  };
}
