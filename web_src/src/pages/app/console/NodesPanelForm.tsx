import { useId } from "react";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Checkbox } from "@/ui/checkbox";

import { useConsoleContext, resolveConsoleNode } from "./ConsoleContext";
import { isManualRunNode } from "./manualRunTriggers";
import { resolveStartTemplate } from "./consoleTriggerParameters";
import type { NodesPanelContent, NodesPanelNode, NodesPanelFormMode } from "./nodesPanelContent";

interface NodesPanelFormProps {
  value: NodesPanelContent;
  onChange: (next: NodesPanelContent) => void;
}

/**
 * Editor form for the merged node/nodes panel. Reused by `NodesPanelCard`;
 * lives in a separate file so `NodesPanelCard.tsx` stays under the shared
 * `max-lines` lint budget.
 */
export function NodesPanelForm({ value, onChange }: NodesPanelFormProps) {
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
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Title (optional)</Label>
        <Input
          value={value.title ?? ""}
          onChange={(e) => onChange({ ...value, title: e.target.value })}
          placeholder="Defaults to panel id"
        />
      </div>
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Nodes</Label>
          <Button type="button" size="sm" variant="outline" onClick={addEntry} data-testid="nodes-panel-add-entry">
            <Plus className="mr-1 h-3.5 w-3.5" />
            Add node
          </Button>
        </div>
        {value.nodes.length === 0 ? (
          <p className="rounded border border-dashed border-slate-200 px-3 py-4 text-center text-xs text-slate-500 dark:border-gray-600 dark:text-gray-400">
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
  const canManualRun = isManualRunNode(resolved?.node);

  return (
    <div className="space-y-2 rounded border border-slate-200 p-2.5 dark:border-gray-600">
      <div className="grid grid-cols-12 gap-2">
        <div className="col-span-6 space-y-1.5">
          <Label className="text-[11px] font-medium text-slate-600 dark:text-gray-400">Node</Label>
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
          <Label className="text-[11px] font-medium text-slate-600 dark:text-gray-400">Label (optional)</Label>
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
        <Label className="text-[11px] font-medium text-slate-600 dark:text-gray-400">Description (optional)</Label>
        <Textarea
          value={entry.description ?? ""}
          onChange={(e) => onChange({ description: e.target.value || undefined })}
          placeholder="Short purpose line shown under the node name"
          className="min-h-[2.25rem] text-xs"
          rows={1}
        />
      </div>
      {canManualRun ? (
        <>
          <div className="flex items-center gap-2">
            <Checkbox
              id={showRunId}
              checked={Boolean(entry.showRun)}
              onCheckedChange={(checked) => onChange({ showRun: checked === true })}
              className="border-slate-300 data-[state=checked]:border-sky-600 data-[state=checked]:bg-sky-600 dark:border-gray-600"
            />
            <Label htmlFor={showRunId} className="text-xs text-slate-700 dark:text-gray-300">
              Show a manual "Run" button (requires run permission).
            </Label>
          </div>
          {entry.showRun ? <NodesPanelRunFormOptions entry={entry} resolved={resolved} onChange={onChange} /> : null}
        </>
      ) : entry.node && resolved ? (
        <p className="text-[11px] text-slate-500 dark:text-gray-400">
          Only trigger nodes with a manual run can be fired from the console. Pick the trigger that starts your flow.
        </p>
      ) : null}
    </div>
  );
}

/**
 * Run-controls section of {@link NodesPanelEntryRow} — trigger template
 * override, form-mode selector, and (in modal mode) the prompt-
 * confirmation checkbox. Extracted so the parent function stays inside
 * the shared lint budget.
 */
function NodesPanelRunFormOptions({
  entry,
  resolved,
  onChange,
}: {
  entry: NodesPanelNode;
  resolved: ReturnType<typeof resolveConsoleNode>;
  onChange: (patch: Partial<NodesPanelNode>) => void;
}) {
  const promptConfirmationId = useId();
  const formModeId = useId();
  const templateForInline = resolved?.node ? resolveStartTemplate(resolved.node, entry.triggerName) : undefined;
  const canOfferInlineForm = (templateForInline?.parameters?.length ?? 0) > 0;
  const currentFormMode: NodesPanelFormMode = entry.formMode ?? "modal";

  return (
    <>
      <div className="space-y-1.5">
        <Label className="text-[11px] font-medium text-slate-600 dark:text-gray-400">Trigger template (optional)</Label>
        <Input
          value={entry.triggerName ?? ""}
          onChange={(e) => onChange({ triggerName: e.target.value || undefined })}
          placeholder="e.g. manual"
          className="h-8"
        />
      </div>
      {canOfferInlineForm ? (
        <div className="space-y-1.5">
          <Label htmlFor={formModeId} className="text-[11px] font-medium text-slate-600 dark:text-gray-400">
            Run form
          </Label>
          <Select
            value={currentFormMode}
            onValueChange={(v) => onChange({ formMode: v === "inline" ? "inline" : undefined })}
          >
            <SelectTrigger id={formModeId} className="h-8">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="modal">Modal (open Run dialog)</SelectItem>
              <SelectItem value="inline">Inline (render form in widget)</SelectItem>
            </SelectContent>
          </Select>
          <p className="text-[11px] text-slate-500 dark:text-gray-400">
            Inline renders the template's parameter form directly in the widget body — best for prompt-submission style
            widgets.
          </p>
        </div>
      ) : null}
      {currentFormMode === "modal" ? (
        <div className="flex items-center gap-2">
          <Checkbox
            id={promptConfirmationId}
            checked={Boolean(entry.promptConfirmation)}
            onCheckedChange={(checked) => onChange({ promptConfirmation: checked === true })}
            className="border-slate-300 data-[state=checked]:border-sky-600 data-[state=checked]:bg-sky-600"
          />
          <Label htmlFor={promptConfirmationId} className="text-xs text-slate-700 dark:text-gray-300">
            Prompt confirmation before running (templates with input fields always prompt).
          </Label>
        </div>
      ) : null}
    </>
  );
}
