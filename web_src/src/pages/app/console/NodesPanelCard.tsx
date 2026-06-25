import { useId, useState } from "react";
import { Network, Play, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/ui/checkbox";
import type { ConsolePanel } from "@/hooks/useCanvasData";

import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { WidgetEmptyState } from "./WidgetEmptyState";
import { useConsoleContext, resolveConsoleNode } from "./ConsoleContext";
import { confirmConsoleTriggerNode } from "./confirmConsoleTriggerNode";
import { NodeRunConfirmDialog } from "./NodeRunConfirmDialog";
import type { NodesPanelContent, NodesPanelNode } from "./nodesPanelContent";

interface NodesPanelCardProps {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onEditingChange?: (editing: boolean) => void;
}

/**
 * Multi-node panel: renders a compact list of canvas nodes with an optional
 * purpose line. Useful for "Key Nodes" style cards that want to surface
 * several pinned nodes in a single console card instead of one panel per
 * node.
 *
 * Resolution and trigger plumbing reuse {@link useConsoleContext} and
 * mirror the single-node panel so authorization and runtime behavior stay
 * consistent across both panel types.
 */
export function NodesPanelCard({ panel, readOnly, onDelete, onChange, onEditingChange }: NodesPanelCardProps) {
  const [editing, setEditing] = useState(false);
  const content = normalizeContent(panel.content);
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
  return (
    <ul className="flex h-full flex-col divide-y divide-slate-100" data-testid="nodes-panel-list">
      {content.nodes.map((entry, idx) => (
        <NodesPanelRow key={`${entry.node}-${idx}`} entry={entry} />
      ))}
    </ul>
  );
}

function NodesPanelRow({ entry }: { entry: NodesPanelNode }) {
  const ctx = useConsoleContext();
  const resolved = resolveConsoleNode(ctx, entry.node);
  const displayName = entry.label?.trim() || resolved?.label || entry.node;
  const isTrigger = resolved?.node.type === "TYPE_TRIGGER";

  return (
    <li className="flex items-center gap-3 px-3 py-2" data-testid="nodes-panel-row">
      <div className="min-w-0 flex-1">
        <div className="truncate text-sm font-medium text-slate-800" data-testid="nodes-panel-row-name">
          {displayName}
        </div>
        {entry.description ? (
          <p className="truncate text-xs text-slate-500" title={entry.description}>
            {entry.description}
          </p>
        ) : null}
        {!resolved ? (
          <p className="truncate text-[11px] text-amber-600">
            Node {JSON.stringify(entry.node)} not found in this canvas.
          </p>
        ) : null}
      </div>
      {entry.showRun && isTrigger ? <NodesPanelRunControl entry={entry} resolved={resolved} /> : null}
    </li>
  );
}

/**
 * Run button + confirm dialog for a single Key Nodes row. Opens
 * {@link NodeRunConfirmDialog} so the operator can preview the merged
 * trigger payload (and fill in any declared template parameters) before the
 * trigger is fired. We never trigger directly from the row click anymore —
 * the dialog confirm is the single submission path.
 */
function NodesPanelRunControl({
  entry,
  resolved,
}: {
  entry: NodesPanelNode;
  resolved: ReturnType<typeof resolveConsoleNode>;
}) {
  const ctx = useConsoleContext();
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
        data-testid="nodes-panel-row-run"
        className="shrink-0"
      >
        <Play className="mr-1 h-3 w-3" />
        Run
      </Button>
      <NodeRunConfirmDialog
        open={open}
        onOpenChange={setOpen}
        resolved={resolved}
        templateName={entry.triggerName}
        onConfirm={async (parameters) => {
          if (!resolved?.node?.id) return;
          await confirmConsoleTriggerNode(ctx, resolved.node.id, entry.triggerName, parameters);
        }}
        testId="nodes-panel-row-run-dialog"
      />
    </>
  );
}

function NodesPanelForm({
  value,
  onChange,
}: {
  value: NodesPanelContent;
  onChange: (next: NodesPanelContent) => void;
}) {
  const updateEntry = (index: number, patch: Partial<NodesPanelNode>) => {
    const nodes = value.nodes.map((entry, i) => (i === index ? { ...entry, ...patch } : entry));
    onChange({ ...value, nodes });
  };
  const removeEntry = (index: number) => {
    onChange({ ...value, nodes: value.nodes.filter((_, i) => i !== index) });
  };
  const addEntry = () => {
    onChange({ ...value, nodes: [...value.nodes, { node: "", description: "", showRun: false }] });
  };

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
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-medium text-slate-600">Nodes</Label>
          <Button type="button" size="sm" variant="outline" onClick={addEntry} data-testid="nodes-panel-add-entry">
            <Plus className="mr-1 h-3.5 w-3.5" />
            Add node
          </Button>
        </div>
        {value.nodes.length === 0 ? (
          <p className="rounded border border-dashed border-slate-200 px-3 py-4 text-center text-xs text-slate-500">
            No nodes yet. Add one to display it in this panel.
          </p>
        ) : (
          <div className="space-y-3">
            {value.nodes.map((entry, index) => (
              <NodesPanelEntryRow
                key={index}
                entry={entry}
                onChange={(patch) => updateEntry(index, patch)}
                onRemove={() => removeEntry(index)}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function NodesPanelEntryRow({
  entry,
  onChange,
  onRemove,
}: {
  entry: NodesPanelNode;
  onChange: (patch: Partial<NodesPanelNode>) => void;
  onRemove: () => void;
}) {
  const ctx = useConsoleContext();
  const nodes = ctx?.nodes ?? [];
  const showRunId = useId();
  const resolved = resolveConsoleNode(ctx, entry.node);
  const isTrigger = resolved?.node.type === "TYPE_TRIGGER";

  return (
    <div className="space-y-2 rounded border border-slate-200 p-2.5">
      <div className="grid grid-cols-12 gap-2">
        <div className="col-span-6 space-y-1.5">
          <Label className="text-[11px] font-medium text-slate-600">Node</Label>
          <Select value={entry.node || "__none__"} onValueChange={(v) => onChange({ node: v === "__none__" ? "" : v })}>
            <SelectTrigger className="h-8">
              <SelectValue placeholder="Select a node" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__none__">Select a node…</SelectItem>
              {nodes.map((n) => (
                <SelectItem key={n.id} value={n.name || n.id || ""}>
                  {n.name || n.id}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="col-span-5 space-y-1.5">
          <Label className="text-[11px] font-medium text-slate-600">Label (optional)</Label>
          <Input
            value={entry.label ?? ""}
            onChange={(e) => onChange({ label: e.target.value || undefined })}
            placeholder="Display name override"
            className="h-8"
          />
        </div>
        <div className="col-span-1 flex items-end justify-end">
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="h-8 w-8"
            onClick={onRemove}
            aria-label="Remove node entry"
            data-testid="nodes-panel-remove-entry"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>
      <div className="space-y-1.5">
        <Label className="text-[11px] font-medium text-slate-600">Description (optional)</Label>
        <Textarea
          value={entry.description ?? ""}
          onChange={(e) => onChange({ description: e.target.value || undefined })}
          placeholder="Short purpose line shown under the node name"
          className="min-h-[2.25rem] text-xs"
          rows={1}
        />
      </div>
      {isTrigger ? (
        <>
          <div className="flex items-center gap-2">
            <Checkbox
              id={showRunId}
              checked={Boolean(entry.showRun)}
              onCheckedChange={(checked) => onChange({ showRun: checked === true })}
              className="border-slate-300 data-[state=checked]:border-sky-600 data-[state=checked]:bg-sky-600"
            />
            <Label htmlFor={showRunId} className="text-xs text-slate-700">
              Show a manual "Run" button (requires run permission).
            </Label>
          </div>
          {entry.showRun ? (
            <div className="space-y-1.5">
              <Label className="text-[11px] font-medium text-slate-600">Trigger template (optional)</Label>
              <Input
                value={entry.triggerName ?? ""}
                onChange={(e) => onChange({ triggerName: e.target.value || undefined })}
                placeholder="e.g. manual"
                className="h-8"
              />
            </div>
          ) : null}
        </>
      ) : entry.node && resolved ? (
        <p className="text-[11px] text-slate-500">
          Only trigger nodes can be run from the console. Pick the trigger that starts your flow.
        </p>
      ) : null}
    </div>
  );
}

function normalizeContent(raw: Record<string, unknown> | undefined): NodesPanelContent {
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
  };
}
