import { useRef } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { SuperplaneComponentsNode } from "@/api-client";

import { useDashboardContext } from "./DashboardContext";
import { DataSourceForm } from "./DataSourceForm";
import { MemoryDiscoveryPanel } from "./MemoryDiscoveryPanel";
import type { TablePanelContent } from "./panelTypes";
import { ActionRow, ColumnRow, FilterRow, type PayloadDraftEntry } from "./TablePanelFormRows";
import type { WidgetRowAction, WidgetTableColumn, WidgetTableFilter } from "./widget/types";
import { sampleRowFromFields, suggestColumnFormat, useMemoryCatalog } from "./widget/useMemoryCatalog";

interface TablePanelFormProps {
  value: TablePanelContent;
  onChange: (next: TablePanelContent) => void;
}

export function TablePanelForm({ value, onChange }: TablePanelFormProps) {
  const ctx = useDashboardContext();
  const canvasId = ctx?.canvasId;
  const triggerNodes = (ctx?.nodes ?? []).filter((n) => n.type === "TYPE_TRIGGER");
  const namespace = value.dataSource.kind === "memory" ? value.dataSource.namespace : "";
  const { fields } = useMemoryCatalog(canvasId, namespace);
  const fieldOptions = fields.map((f) => f.field);
  const sampleRow = sampleRowFromFields(fields);
  const payloadDrafts = usePayloadDrafts(value);
  const actions = useTablePanelFormActions({ value, onChange, fields, triggerNodes, payloadDrafts });

  return (
    <div className="space-y-4">
      <TitleField value={value} onChange={onChange} />
      <DataSourceForm value={value.dataSource} onChange={(dataSource) => onChange({ ...value, dataSource })} />
      <MemorySourcePicker value={value} canvasId={canvasId} onChange={onChange} />
      <ColumnsSection value={value} fields={fields} fieldOptions={fieldOptions} actions={actions} />
      <FiltersSection value={value} fieldOptions={fieldOptions} actions={actions} />
      <RowActionsSection
        value={value}
        triggerNodes={triggerNodes}
        fieldOptions={fieldOptions}
        sampleRow={sampleRow}
        payloadDrafts={payloadDrafts}
        actions={actions}
      />
    </div>
  );
}

/**
 * Track stable per-action payload row ids so React reconciles inputs by id
 * rather than by `path`. Re-keying by path caused focus loss + duplicate rows
 * whenever the user edited a path (tab/blur).
 *
 * The persisted shape is still `Record<string, string>`. Drafts are the
 * authoritative ordered row list during editing — blank rows and rows with an
 * empty path are kept in the UI but stripped before save. We only re-seed
 * drafts from the persisted payload when the value changes externally (e.g.
 * YAML tab edit or undo), detected by comparing what the draft would persist
 * against what is actually persisted.
 */
function usePayloadDrafts(value: TablePanelContent) {
  const drafts = useRef<Map<number, PayloadDraftEntry[]>>(new Map());
  const counter = useRef(0);
  const newRowId = () => `row-${counter.current++}`;

  const seedFromPayload = (payload: Record<string, string> | undefined): PayloadDraftEntry[] => {
    const seeded: PayloadDraftEntry[] = [];
    for (const [path, template] of Object.entries(payload ?? {})) {
      seeded.push({ rowId: newRowId(), path, template });
    }
    return seeded;
  };

  const ensureTrailingBlank = (entries: PayloadDraftEntry[]): PayloadDraftEntry[] => {
    const last = entries[entries.length - 1];
    if (last && !last.path && !last.template) return entries;
    return [...entries, { rowId: newRowId(), path: "", template: "" }];
  };

  const snapshots: PayloadDraftEntry[][] = (value.render.rowActions ?? []).map((action, idx) => {
    let entries = drafts.current.get(idx);
    if (!entries) {
      entries = seedFromPayload(action.payload);
    } else {
      const wouldPersist = draftToPayloadShape(entries);
      const persisted = action.payload ?? {};
      if (!payloadShallowEqual(wouldPersist, persisted)) {
        entries = seedFromPayload(action.payload);
      }
    }
    entries = ensureTrailingBlank(entries);
    drafts.current.set(idx, entries);
    return entries;
  });

  return {
    snapshots,
    newRowId,
    setDraft: (idx: number, entries: PayloadDraftEntry[]) => {
      drafts.current.set(idx, entries);
    },
    dropDraft: (idx: number) => {
      drafts.current.delete(idx);
    },
  };
}

type PayloadDrafts = ReturnType<typeof usePayloadDrafts>;

function draftToPayloadShape(entries: PayloadDraftEntry[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const entry of entries) {
    if (!entry.path) continue;
    out[entry.path] = entry.template;
  }
  return out;
}

/**
 * Serialize a draft list back into the persisted `Record<string, string>`.
 * Trailing blank rows are dropped, and entries with an empty path are skipped
 * so the on-disk shape stays clean.
 */
function draftToPayload(entries: PayloadDraftEntry[]): Record<string, string> | undefined {
  const shape = draftToPayloadShape(entries);
  return Object.keys(shape).length > 0 ? shape : undefined;
}

function payloadShallowEqual(a: Record<string, string>, b: Record<string, string>): boolean {
  const ak = Object.keys(a);
  const bk = Object.keys(b);
  if (ak.length !== bk.length) return false;
  for (const k of ak) {
    if (!(k in b) || a[k] !== b[k]) return false;
  }
  return true;
}

function useTablePanelFormActions({
  value,
  onChange,
  fields,
  triggerNodes,
  payloadDrafts,
}: {
  value: TablePanelContent;
  onChange: (next: TablePanelContent) => void;
  fields: Array<{ field: string }>;
  triggerNodes: SuperplaneComponentsNode[];
  payloadDrafts: PayloadDrafts;
}) {
  const updateColumn = (idx: number, patch: Partial<WidgetTableColumn>) => {
    const columns = value.render.columns.map((col, i) => (i === idx ? { ...col, ...patch } : col));
    onChange({ ...value, render: { ...value.render, columns } });
  };

  const addColumnFromField = (field: string) => {
    if (!field || value.render.columns.some((c) => c.field === field)) return;
    onChange({
      ...value,
      render: {
        ...value.render,
        columns: [...value.render.columns, { field, label: field, format: suggestColumnFormat(field) }],
      },
    });
  };

  const addAllFields = () => {
    const existing = new Set(value.render.columns.map((c) => c.field));
    const next = [...value.render.columns];
    for (const { field } of fields) {
      if (existing.has(field)) continue;
      next.push({ field, label: field, format: suggestColumnFormat(field) });
    }
    onChange({ ...value, render: { ...value.render, columns: next } });
  };

  const addColumn = () => {
    onChange({
      ...value,
      render: { ...value.render, columns: [...value.render.columns, { field: "", label: "" }] },
    });
  };

  const removeColumn = (idx: number) => {
    onChange({
      ...value,
      render: { ...value.render, columns: value.render.columns.filter((_, i) => i !== idx) },
    });
  };

  const updateFilter = (idx: number, patch: Partial<WidgetTableFilter>) => {
    const where = (value.render.where ?? []).map((f, i) => (i === idx ? { ...f, ...patch } : f));
    onChange({ ...value, render: { ...value.render, where } });
  };

  const addFilter = () => {
    const where = [...(value.render.where ?? []), { field: "", op: "eq" as const, value: "" }];
    onChange({ ...value, render: { ...value.render, where } });
  };

  const removeFilter = (idx: number) => {
    onChange({
      ...value,
      render: { ...value.render, where: (value.render.where ?? []).filter((_, i) => i !== idx) },
    });
  };

  const updateAction = (idx: number, patch: Partial<WidgetRowAction>) => {
    const rowActions = (value.render.rowActions ?? []).map((a, i) =>
      i === idx ? { ...a, ...patch } : a,
    ) as WidgetRowAction[];
    onChange({ ...value, render: { ...value.render, rowActions } });
  };

  const addAction = () => {
    const rowActions: WidgetRowAction[] = [
      ...(value.render.rowActions ?? []),
      { kind: "trigger", label: "Run", node: triggerNodes[0]?.name || triggerNodes[0]?.id || "", hook: "run" },
    ];
    onChange({ ...value, render: { ...value.render, rowActions } });
  };

  const removeAction = (idx: number) => {
    payloadDrafts.dropDraft(idx);
    onChange({
      ...value,
      render: { ...value.render, rowActions: (value.render.rowActions ?? []).filter((_, i) => i !== idx) },
    });
  };

  const payloadActions = makePayloadActions({ payloadDrafts, updateAction });

  return {
    updateColumn,
    addColumnFromField,
    addAllFields,
    addColumn,
    removeColumn,
    updateFilter,
    addFilter,
    removeFilter,
    updateAction,
    addAction,
    removeAction,
    ...payloadActions,
  };
}

function makePayloadActions({
  payloadDrafts,
  updateAction,
}: {
  payloadDrafts: PayloadDrafts;
  updateAction: (idx: number, patch: Partial<WidgetRowAction>) => void;
}) {
  const commitDraft = (idx: number, entries: PayloadDraftEntry[]) => {
    payloadDrafts.setDraft(idx, entries);
    updateAction(idx, { payload: draftToPayload(entries) });
  };

  const updatePayloadEntry = (actionIdx: number, rowId: string, patch: Partial<Omit<PayloadDraftEntry, "rowId">>) => {
    const current = payloadDrafts.snapshots[actionIdx];
    if (!current) return;
    const next = current.map((entry) => (entry.rowId === rowId ? { ...entry, ...patch } : entry));
    commitDraft(actionIdx, next);
  };

  const removePayloadEntry = (actionIdx: number, rowId: string) => {
    const current = payloadDrafts.snapshots[actionIdx];
    if (!current) return;
    const next = current.filter((entry) => entry.rowId !== rowId);
    commitDraft(actionIdx, next);
  };

  const quickInsertPayloadField = (actionIdx: number, field: string) => {
    const current = payloadDrafts.snapshots[actionIdx];
    if (!current) return;
    const path = field;
    const template = `{{ ${field} }}`;
    // If the field already exists as a path, leave the existing row alone —
    // the chip click acts as "ensure this field is mapped" and shouldn't
    // overwrite an author-customized value template.
    if (current.some((entry) => entry.path === path)) return;
    const trailing = current[current.length - 1];
    const next = [...current];
    if (trailing && !trailing.path && !trailing.template) {
      next[next.length - 1] = { ...trailing, path, template };
    } else {
      next.push({ rowId: payloadDrafts.newRowId(), path, template });
    }
    commitDraft(actionIdx, next);
  };

  return { updatePayloadEntry, removePayloadEntry, quickInsertPayloadField };
}

type TablePanelFormActions = ReturnType<typeof useTablePanelFormActions>;

function TitleField({ value, onChange }: { value: TablePanelContent; onChange: (next: TablePanelContent) => void }) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600">Title (optional)</Label>
      <Input
        value={value.title ?? ""}
        onChange={(e) => onChange({ ...value, title: e.target.value })}
        placeholder="Defaults to panel id"
      />
    </div>
  );
}

function MemorySourcePicker({
  value,
  canvasId,
  onChange,
}: {
  value: TablePanelContent;
  canvasId: string | undefined;
  onChange: (next: TablePanelContent) => void;
}) {
  if (value.dataSource.kind !== "memory") return null;
  const dataSource = value.dataSource;

  return (
    <MemoryDiscoveryPanel
      canvasId={canvasId}
      selectedNamespace={dataSource.namespace}
      onSelectNamespace={(namespace) => onChange({ ...value, dataSource: { ...dataSource, namespace } })}
    />
  );
}

function ColumnsSection({
  value,
  fields,
  fieldOptions,
  actions,
}: {
  value: TablePanelContent;
  fields: Array<{ field: string; sample?: string }>;
  fieldOptions: string[];
  actions: TablePanelFormActions;
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600">Columns</Label>
        <div className="flex gap-1">
          {value.dataSource.kind === "memory" && fields.length > 0 ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              onClick={actions.addAllFields}
              data-testid="table-add-all-columns"
            >
              Add all fields
            </Button>
          ) : null}
          <Button type="button" size="sm" variant="outline" onClick={actions.addColumn} data-testid="table-add-column">
            Add column
          </Button>
        </div>
      </div>
      <MemoryFieldButtons value={value} fields={fields} onSelect={actions.addColumnFromField} />
      <div className="space-y-2">
        {value.render.columns.map((col, idx) => (
          <ColumnRow
            key={idx}
            col={col}
            fieldOptions={fieldOptions}
            onChange={(patch) => actions.updateColumn(idx, patch)}
            onRemove={() => actions.removeColumn(idx)}
          />
        ))}
        {value.render.columns.length === 0 ? (
          <p className="text-xs text-slate-500">
            Add columns to display memory rows. Use discovered fields or custom paths / CEL.
          </p>
        ) : null}
      </div>
    </div>
  );
}

function MemoryFieldButtons({
  value,
  fields,
  onSelect,
}: {
  value: TablePanelContent;
  fields: Array<{ field: string; sample?: string }>;
  onSelect: (field: string) => void;
}) {
  if (value.dataSource.kind !== "memory" || fields.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-1">
      {fields.map((f) => (
        <Button
          key={f.field}
          type="button"
          size="sm"
          variant="secondary"
          className="h-6 text-[10px]"
          onClick={() => onSelect(f.field)}
          title={f.sample ? `e.g. ${f.sample}` : undefined}
        >
          {f.field}
        </Button>
      ))}
    </div>
  );
}

function FiltersSection({
  value,
  fieldOptions,
  actions,
}: {
  value: TablePanelContent;
  fieldOptions: string[];
  actions: TablePanelFormActions;
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600">Filters</Label>
        <Button type="button" size="sm" variant="outline" onClick={actions.addFilter}>
          Add filter
        </Button>
      </div>
      <div className="space-y-2">
        {(value.render.where ?? []).map((filter, idx) => (
          <FilterRow
            key={idx}
            filter={filter}
            fieldOptions={fieldOptions}
            onChange={(patch) => actions.updateFilter(idx, patch)}
            onRemove={() => actions.removeFilter(idx)}
          />
        ))}
      </div>
    </div>
  );
}

function RowActionsSection({
  value,
  triggerNodes,
  fieldOptions,
  sampleRow,
  payloadDrafts,
  actions,
}: {
  value: TablePanelContent;
  triggerNodes: SuperplaneComponentsNode[];
  fieldOptions: string[];
  sampleRow: Record<string, unknown>;
  payloadDrafts: PayloadDrafts;
  actions: TablePanelFormActions;
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600">Row actions</Label>
        <Button type="button" size="sm" variant="outline" onClick={actions.addAction} data-testid="table-add-action">
          Add action
        </Button>
      </div>
      <p className="text-[11px] text-slate-500">
        Pick the <strong>trigger</strong> that starts your flow (e.g. Start). HTTP Request and other steps run when that
        trigger fires. Payload values support <code className="text-[10px]">{`{{ field }}`}</code> CEL — wrap numeric
        strings with <code className="text-[10px]">int()</code> or <code className="text-[10px]">float()</code> for
        arithmetic (e.g. <code className="text-[10px]">{`{{ int(value) / 2 }}`}</code>).
      </p>
      <div className="space-y-3">
        {(value.render.rowActions ?? []).map((action, idx) => (
          <ActionRow
            key={idx}
            action={action}
            triggerNodes={triggerNodes}
            fieldOptions={fieldOptions}
            sampleRow={sampleRow}
            payloadEntries={payloadDrafts.snapshots[idx] ?? []}
            onChange={(patch) => actions.updateAction(idx, patch)}
            onRemove={() => actions.removeAction(idx)}
            onPayloadEntryChange={(rowId, patch) => actions.updatePayloadEntry(idx, rowId, patch)}
            onPayloadEntryRemove={(rowId) => actions.removePayloadEntry(idx, rowId)}
            onPayloadEntryQuickInsert={(field) => actions.quickInsertPayloadField(idx, field)}
          />
        ))}
      </div>
    </div>
  );
}
