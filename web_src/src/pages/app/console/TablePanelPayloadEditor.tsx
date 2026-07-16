import { useId } from "react";
import { Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ExpressionEditor } from "@/components/ExpressionEditor";

import type { PayloadDraftEntry } from "@/lib/tablePanelPayloadDraft";

export type { PayloadDraftEntry } from "@/lib/tablePanelPayloadDraft";

export function PayloadEditor({
  entries,
  fieldOptions,
  sampleRow,
  onEntryChange,
  onEntryRemove,
  onQuickInsert,
}: {
  entries: PayloadDraftEntry[];
  fieldOptions: string[];
  sampleRow: Record<string, unknown>;
  onEntryChange: (rowId: string, patch: Partial<Omit<PayloadDraftEntry, "rowId">>) => void;
  onEntryRemove: (rowId: string) => void;
  onQuickInsert: (field: string) => void;
}) {
  const hasFieldChips = fieldOptions.length > 0;

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <span className="text-[11px] font-medium text-slate-600">Payload fields</span>
        <span className="text-[10px] text-slate-500">
          {entries.length === 0 ? "Type below to add" : "Empty row appears automatically"}
        </span>
      </div>
      {hasFieldChips ? (
        <div className="flex flex-wrap gap-1">
          {fieldOptions.map((field) => (
            <Button
              key={field}
              type="button"
              size="sm"
              variant="secondary"
              className="h-6 text-[10px]"
              onClick={() => onQuickInsert(field)}
              title={`Add ${field}: {{ ${field} }}`}
            >
              <Plus className="mr-0.5 h-2.5 w-2.5" />
              {field}
            </Button>
          ))}
        </div>
      ) : null}
      <div className="space-y-1">
        {entries.map((entry) => (
          <PayloadEntry
            key={entry.rowId}
            entry={entry}
            fieldOptions={fieldOptions}
            sampleRow={sampleRow}
            onChange={(patch) => onEntryChange(entry.rowId, patch)}
            onRemove={() => onEntryRemove(entry.rowId)}
          />
        ))}
      </div>
    </div>
  );
}

function PayloadEntry({
  entry,
  fieldOptions,
  sampleRow,
  onChange,
  onRemove,
}: {
  entry: PayloadDraftEntry;
  fieldOptions: string[];
  sampleRow: Record<string, unknown>;
  onChange: (patch: Partial<Omit<PayloadDraftEntry, "rowId">>) => void;
  onRemove: () => void;
}) {
  const isBlankTrailingRow = !entry.path && !entry.template;
  // Each row owns its own datalist id so multiple actions on the same panel do
  // not collide and the chrome can render the field autocomplete inline.
  const reactId = useId();
  const datalistId = fieldOptions.length > 0 ? `payload-fields-${reactId}` : undefined;
  return (
    <div className="grid grid-cols-12 items-start gap-1">
      <Input
        value={entry.path}
        onChange={(e) => onChange({ path: e.target.value })}
        placeholder="data.issue.number"
        className="col-span-5 h-7 text-xs"
        list={datalistId}
      />
      <div className="col-span-6">
        <ExpressionEditor
          dialect="cel"
          exampleObj={sampleRow}
          value={entry.template}
          onChange={(next) => onChange({ template: next })}
          placeholder="{{ pr_number }} or int(value) / 2"
          inputSize="xs"
          showValuePreview
          valuePreviewLabel="Preview"
          data-testid="payload-template-input"
        />
      </div>
      <Button
        type="button"
        size="icon"
        variant="ghost"
        className="col-span-1 h-7 w-7"
        onClick={onRemove}
        disabled={isBlankTrailingRow}
        title={isBlankTrailingRow ? "Empty row — type to add" : "Remove field"}
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
      {datalistId ? (
        <datalist id={datalistId}>
          {fieldOptions.map((f) => (
            <option key={f} value={f} />
          ))}
        </datalist>
      ) : null}
    </div>
  );
}
