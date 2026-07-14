import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { SuperplaneComponentsNode } from "@/api-client";

import { MemoryDiscoveryPanel } from "../MemoryDiscoveryPanel";
import type { TablePanelContent } from "../panelTypes";
import { ActionRow } from "../TablePanelFormActionRow";
import { ColumnRow, FilterRow, RowStyleRow } from "../TablePanelFormRows";
import { WIDGET_SORT_ORDERS, type WidgetSortOrder } from "../widget/types";
import type { TablePanelFormActions } from "./useTablePanelFormActions";
import type { TablePanelPayloadDrafts } from "./useTablePanelPayloadDrafts";

export function TablePanelTitleField({
  value,
  onChange,
}: {
  value: TablePanelContent;
  onChange: (next: TablePanelContent) => void;
}) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Title (optional)</Label>
      <Input
        value={value.title ?? ""}
        onChange={(e) => onChange({ ...value, title: e.target.value })}
        placeholder="Defaults to panel id"
      />
    </div>
  );
}

export function TablePanelMemorySourcePicker({
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

export function TablePanelColumnsSection({
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
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Columns</Label>
        <div className="flex gap-1">
          {fields.length > 0 ? (
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
      <TablePanelFieldQuickAddButtons fields={fields} onSelect={actions.addColumnFromField} />
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
          <p className="text-xs text-slate-500 dark:text-gray-400">{emptyColumnsHint(value.dataSource.kind)}</p>
        ) : null}
      </div>
    </div>
  );
}

function emptyColumnsHint(kind: string): string {
  if (kind === "executions")
    return "Add columns to display execution rows. Use the suggested fields or custom paths / CEL.";
  if (kind === "runs") return "Add columns to display run rows. Use the suggested fields or custom paths / CEL.";
  return "Add columns to display memory rows. Use discovered fields or custom paths / CEL.";
}

function TablePanelFieldQuickAddButtons({
  fields,
  onSelect,
}: {
  fields: Array<{ field: string; sample?: string }>;
  onSelect: (field: string) => void;
}) {
  if (fields.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-1" data-testid="table-field-quick-add">
      {fields.map((f) => (
        <Button
          key={f.field}
          type="button"
          size="sm"
          variant="ghost"
          className="h-6 bg-slate-100 text-[10px] text-gray-800 hover:bg-slate-200 dark:bg-gray-800 dark:text-gray-200 dark:hover:bg-gray-700"
          onClick={() => onSelect(f.field)}
          title={f.sample ? `e.g. ${f.sample}` : undefined}
        >
          {f.field}
        </Button>
      ))}
    </div>
  );
}

export function TablePanelFiltersSection({
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
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Filters</Label>
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

export function TablePanelRowStylesSection({
  value,
  fieldOptions,
  actions,
}: {
  value: TablePanelContent;
  fieldOptions: string[];
  actions: TablePanelFormActions;
}) {
  const rules = value.render.rowStyles ?? [];
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Row background</Label>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={actions.addRowStyle}
          data-testid="table-add-row-style"
        >
          Add rule
        </Button>
      </div>
      <p className="text-[11px] text-slate-500 dark:text-gray-400">
        Tint a row when its data matches a condition (e.g.{" "}
        <code className="text-[10px]">status == &quot;error&quot;</code>). First matching rule wins.
      </p>
      <div className="space-y-2">
        {rules.map((rule, idx) => (
          <RowStyleRow
            key={idx}
            rule={rule}
            fieldOptions={fieldOptions}
            onChange={(patch) => actions.updateRowStyle(idx, patch)}
            onRemove={() => actions.removeRowStyle(idx)}
          />
        ))}
      </div>
    </div>
  );
}

export function TablePanelSortSection({
  value,
  fieldOptions,
  actions,
}: {
  value: TablePanelContent;
  fieldOptions: string[];
  actions: TablePanelFormActions;
}) {
  const sort = value.render.sort;
  const sortField = sort?.field ?? "";
  const sortOrder: WidgetSortOrder = sort?.order ?? "asc";
  const hasSortField = sortField.trim() !== "";
  const datalistId = fieldOptions.length > 0 ? "table-sort-field-options" : undefined;

  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Sort by (optional)</Label>
      <div className="grid grid-cols-3 gap-2">
        <Input
          className="col-span-2 h-8"
          list={datalistId}
          value={sortField}
          onChange={(e) =>
            actions.setSort(e.target.value.trim() ? { field: e.target.value, order: sort?.order } : undefined)
          }
          placeholder="e.g. createdAt or {{ expr }} (blank = unsorted)"
          data-testid="table-sort-field"
        />
        <Select
          value={sortOrder}
          onValueChange={(v) => actions.setSort({ field: sortField, order: v as WidgetSortOrder })}
          disabled={!hasSortField}
        >
          <SelectTrigger className="h-8 w-full" data-testid="table-sort-order">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {WIDGET_SORT_ORDERS.map((o) => (
              <SelectItem key={o} value={o}>
                {o === "asc" ? "Ascending" : "Descending"}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
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

export function TablePanelRowActionsSection({
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
  payloadDrafts: TablePanelPayloadDrafts;
  actions: TablePanelFormActions;
}) {
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Row actions</Label>
        <Button type="button" size="sm" variant="outline" onClick={actions.addAction} data-testid="table-add-action">
          Add action
        </Button>
      </div>
      <p className="max-w-xl text-xs text-slate-500 dark:text-gray-400">
        Pick the <strong className="font-semibold">trigger</strong> that starts your flow (e.g. Start). HTTP Request and
        other steps run when that trigger fires. Payload values support{" "}
        <code className="text-[11px]">{`{{ field }}`}</code> CEL — wrap numeric strings with{" "}
        <code className="text-[11px]">int()</code> or <code className="text-[11px]">float()</code> for arithmetic (e.g.{" "}
        <code className="text-[11px]">{`{{ int(value) / 2 }}`}</code>).
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
