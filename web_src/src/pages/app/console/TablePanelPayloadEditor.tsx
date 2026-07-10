import { useId, useMemo } from "react";
import { AlertTriangle, Plus, Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

import type { PayloadDraftEntry } from "@/lib/tablePanelPayloadDraft";

import { CONSOLE_CODE_BADGE_CLASSES } from "./consoleCodeStyles";
import { buildEnv, compileTemplate, evalTemplateDetailed } from "./widget/celExpr";

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
    <div className="space-y-1">
      <div className="grid grid-cols-12 items-center gap-1">
        <Input
          value={entry.path}
          onChange={(e) => onChange({ path: e.target.value })}
          placeholder="data.issue.number"
          className="col-span-5 h-7 text-xs"
          list={datalistId}
        />
        <Input
          value={entry.template}
          onChange={(e) => onChange({ template: e.target.value })}
          placeholder="{{ pr_number }} or int(value) / 2"
          className="col-span-6 h-7 text-xs"
        />
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
      <PayloadPreview entry={entry} sampleRow={sampleRow} />
    </div>
  );
}

/**
 * Inline preview of a payload value template evaluated against the first
 * memory sample row. Surfaces CEL compile/eval errors that `evalExpr` would
 * otherwise silently swallow so authors get fast feedback while typing.
 */
function PayloadPreview({ entry, sampleRow }: { entry: PayloadDraftEntry; sampleRow: Record<string, unknown> }) {
  const preview = useMemo(() => {
    if (!entry.template) return null;
    if (!entry.template.includes("{{")) return null;
    const env = buildEnv();
    const stringify = (v: unknown) => (v == null ? "" : typeof v === "string" ? v : JSON.stringify(v));
    return evalTemplateDetailed(compileTemplate(entry.template), sampleRow, env, stringify);
  }, [entry.template, sampleRow]);

  if (!preview) return null;
  if (!preview.ok) {
    return (
      <p className="col-span-12 flex items-start gap-1 text-[10px] text-red-600">
        <AlertTriangle className="mt-0.5 h-3 w-3 shrink-0" />
        <span>
          <span className="font-medium">CEL error:</span> {preview.error}
        </span>
      </p>
    );
  }

  const hasSample = Object.keys(sampleRow).length > 0;
  const text = preview.value;
  if (!text && !hasSample) {
    return (
      <p className="text-[10px] text-slate-400">
        Preview unavailable — no memory data yet. Run your workflow once, then revisit.
      </p>
    );
  }
  return (
    <p className="text-[10px] text-slate-500" data-testid="payload-preview">
      <span className="font-medium text-slate-600">Preview:</span>{" "}
      <code className={CONSOLE_CODE_BADGE_CLASSES}>{text || "(empty)"}</code>
    </p>
  );
}
