import { useId, useState } from "react";
import { CircleDot, Play } from "lucide-react";

import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Checkbox } from "@/ui/checkbox";
import type { ConsolePanel } from "@/hooks/useCanvasData";

import { PanelEditorDialog } from "./PanelEditorDialog";
import { TypedPanelShell } from "./TypedPanelShell";
import { WidgetEmptyState } from "./WidgetEmptyState";
import { useConsoleContext, resolveConsoleNode } from "./ConsoleContext";
import { NodeRunConfirmDialog } from "./NodeRunConfirmDialog";
import { useConsoleRunTrigger } from "./useConsoleRunTrigger";
import type { NodePanelContent } from "./panelTypes";

interface NodePanelCardProps {
  panel: ConsolePanel;
  readOnly: boolean;
  onDelete: () => void;
  onChange: (content: Record<string, unknown>) => void;
  onEditingChange?: (editing: boolean) => void;
}

/**
 * Single-node panel: node name + optional manual-run button. Resolves the
 * node reference (id or name) through {@link ConsoleContext}.
 */
export function NodePanelCard({ panel, readOnly, onDelete, onChange, onEditingChange }: NodePanelCardProps) {
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
        <NodePanelBody content={content} />
      </TypedPanelShell>
      <PanelEditorDialog<NodePanelContent>
        open={editing}
        onOpenChange={setEditingState}
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
  const ctx = useConsoleContext();
  if (!content.node) {
    return (
      <WidgetEmptyState
        icon={CircleDot}
        className="min-h-0"
        message="Pick a node from the editor to display it here."
      />
    );
  }
  const resolved = resolveConsoleNode(ctx, content.node);
  const displayName = content.label?.trim() || resolved?.label || content.node || "—";
  const isTrigger = resolved?.node.type === "TYPE_TRIGGER";

  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 p-4">
      <div className="text-[13px] font-semibold text-slate-800 dark:text-gray-100" data-testid="node-panel-name">
        {displayName}
      </div>
      {content.showRun && isTrigger ? <NodePanelRunControl content={content} resolved={resolved} /> : null}
      {!resolved && content.node ? (
        <p className="text-[13px] text-amber-600 dark:text-amber-400">
          Node {JSON.stringify(content.node)} not found in this canvas.
        </p>
      ) : null}
    </div>
  );
}

/**
 * Run button + confirm dialog for the single-node panel. A template with
 * input fields always opens {@link NodeRunConfirmDialog} so the operator can
 * fill them in. A parameter-less template only prompts when the panel opts in
 * via `promptConfirmation`; otherwise the click fires the trigger directly.
 */
function NodePanelRunControl({
  content,
  resolved,
}: {
  content: NodePanelContent;
  resolved: ReturnType<typeof resolveConsoleNode>;
}) {
  const { canRun, running, dialogOpen, setDialogOpen, handleClick, runTrigger } = useConsoleRunTrigger({
    resolved,
    triggerName: content.triggerName,
    promptConfirmation: content.promptConfirmation,
  });

  return (
    <>
      <LoadingButton
        type="button"
        size="xs"
        variant="outline"
        loading={running}
        loadingText="Running…"
        onClick={handleClick}
        disabled={!canRun || running}
        title={canRun ? undefined : "You do not have permission to run this node"}
        data-testid="node-panel-run"
      >
        <Play className="mr-1 h-3.5 w-3.5" />
        Run
      </LoadingButton>
      <NodeRunConfirmDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        resolved={resolved}
        templateName={content.triggerName}
        onConfirm={runTrigger}
        testId="node-panel-run-dialog"
      />
    </>
  );
}

function NodePanelForm({ value, onChange }: { value: NodePanelContent; onChange: (next: NodePanelContent) => void }) {
  const ctx = useConsoleContext();
  const nodes = ctx?.nodes ?? [];
  const showRunId = useId();
  const promptConfirmationId = useId();
  const resolved = resolveConsoleNode(ctx, value.node);
  const isTrigger = resolved?.node.type === "TYPE_TRIGGER";
  return (
    <div className="space-y-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Title (optional)</Label>
        <Input
          value={value.title ?? ""}
          onChange={(e) => onChange({ ...value, title: e.target.value })}
          placeholder="Defaults to panel id"
        />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Node</Label>
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
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Label (optional)</Label>
        <Input
          value={value.label ?? ""}
          onChange={(e) => onChange({ ...value, label: e.target.value || undefined })}
          placeholder="Display name override"
        />
      </div>
      {isTrigger ? (
        <>
          <div className="flex items-center gap-2">
            <Checkbox
              id={showRunId}
              checked={Boolean(value.showRun)}
              onCheckedChange={(checked) => onChange({ ...value, showRun: checked === true })}
              className="border-slate-300 data-[state=checked]:border-sky-600 data-[state=checked]:bg-sky-600 dark:border-gray-600"
            />
            <Label htmlFor={showRunId} className="text-xs text-slate-700 dark:text-gray-300">
              Show a manual "Run" button (requires run permission).
            </Label>
          </div>
          {value.showRun ? (
            <>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">
                  Trigger template (optional)
                </Label>
                <Input
                  value={value.triggerName ?? ""}
                  onChange={(e) => onChange({ ...value, triggerName: e.target.value || undefined })}
                  placeholder="e.g. manual"
                />
              </div>
              <div className="flex items-center gap-2">
                <Checkbox
                  id={promptConfirmationId}
                  checked={Boolean(value.promptConfirmation)}
                  onCheckedChange={(checked) => onChange({ ...value, promptConfirmation: checked === true })}
                  className="border-slate-300 data-[state=checked]:border-sky-600 data-[state=checked]:bg-sky-600"
                />
                <Label htmlFor={promptConfirmationId} className="text-xs text-slate-700 dark:text-gray-300">
                  Prompt confirmation before running (templates with input fields always prompt).
                </Label>
              </div>
            </>
          ) : null}
        </>
      ) : value.node && resolved ? (
        <p className="text-xs text-slate-500 dark:text-gray-400">
          Only trigger nodes can be run from the console. Pick the trigger that starts your flow.
        </p>
      ) : null}
    </div>
  );
}

function normalizeContent(raw: Record<string, unknown> | undefined): NodePanelContent {
  return {
    title: typeof raw?.title === "string" ? raw.title : "",
    node: typeof raw?.node === "string" ? raw.node : "",
    label: typeof raw?.label === "string" ? raw.label : undefined,
    showRun: typeof raw?.showRun === "boolean" ? raw.showRun : false,
    triggerName: typeof raw?.triggerName === "string" ? raw.triggerName : undefined,
    promptConfirmation: typeof raw?.promptConfirmation === "boolean" ? raw.promptConfirmation : false,
  };
}
