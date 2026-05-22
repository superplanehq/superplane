import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { SuperplaneComponentsNode } from "@/api-client";

import { MemoryDiscoveryPanel } from "../MemoryDiscoveryPanel";
import type { TablePanelContent } from "../panelTypes";
import { ActionRow, ColumnRow, FilterRow } from "../TablePanelFormRows";
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
      <Label className="text-xs font-medium text-slate-600">Title (optional)</Label>
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
      <TablePanelMemoryFieldButtons value={value} fields={fields} onSelect={actions.addColumnFromField} />
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

function TablePanelMemoryFieldButtons({
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
