import { Trash2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Checkbox } from "@/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { SuperplaneComponentsNode } from "@/api-client";
import type { PayloadDraftEntry } from "@/lib/tablePanelPayloadDraft";

import type { BoardPanelContent } from "../boardPanelContent";
import { ActionRow } from "../TablePanelFormActionRow";
import { ColumnRow, FilterRow } from "../TablePanelFormRows";
import type { RowActionPayloadDrafts } from "../tablePanelForm/useRowActionPayloadDrafts";
import {
  WIDGET_BOARD_LANE_COLORS,
  WIDGET_SORT_ORDERS,
  type WidgetBoardLane,
  type WidgetBoardLaneColor,
  type WidgetRowAction,
  type WidgetSortOrder,
  type WidgetTableColumn,
  type WidgetTableFilter,
} from "../widget/types";
import type { BoardPanelFormActions } from "./useBoardPanelFormActions";

const NEUTRAL_LANE_COLOR: WidgetBoardLaneColor = "neutral";

/** Title + groupBy + otherLane toggle. */
export function BoardHeaderFields({
  value,
  fieldOptions,
  onChange,
}: {
  value: BoardPanelContent;
  fieldOptions: string[];
  onChange: (next: BoardPanelContent) => void;
}) {
  return (
    <>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Title (optional)</Label>
        <Input
          value={value.title ?? ""}
          onChange={(e) => onChange({ ...value, title: e.target.value })}
          placeholder="Defaults to panel id"
        />
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Group rows by</Label>
        <Input
          value={value.render.groupBy}
          onChange={(e) => onChange({ ...value, render: { ...value.render, groupBy: e.target.value } })}
          placeholder="e.g. status or {{ payload.state }}"
          list={fieldOptions.length > 0 ? "board-field-options" : undefined}
          data-testid="board-groupby-field"
        />
        <p className="text-[11px] text-slate-500 dark:text-gray-400">
          Row field whose value places the row into a lane. Values are matched case-insensitively and trimmed.
        </p>
      </div>
      <div className="flex items-center gap-2">
        <Checkbox
          id="board-other-lane"
          checked={Boolean(value.render.otherLane)}
          onCheckedChange={(v) =>
            onChange({ ...value, render: { ...value.render, otherLane: v === true ? true : undefined } })
          }
          data-testid="board-other-lane-toggle"
        />
        <Label
          htmlFor="board-other-lane"
          className="cursor-pointer text-xs font-medium text-slate-600 dark:text-gray-400"
        >
          Show unmatched rows in a trailing &ldquo;Other&rdquo; lane
        </Label>
      </div>
    </>
  );
}

export function BoardLanesSection({ value, actions }: { value: BoardPanelContent; actions: BoardPanelFormActions }) {
  const lanes = value.render.lanes;
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Lanes</Label>
        <Button type="button" size="sm" variant="outline" onClick={actions.addLane} data-testid="board-add-lane">
          Add lane
        </Button>
      </div>
      <div className="space-y-2">
        {lanes.map((lane, idx) => (
          <BoardLaneRow
            key={idx}
            lane={lane}
            onChange={(patch) => actions.updateLane(idx, patch)}
            onRemove={() => actions.removeLane(idx)}
          />
        ))}
        {lanes.length === 0 ? (
          <p className="text-xs text-slate-500 dark:text-gray-400">
            Add at least one lane. Each lane matches rows whose <code className="text-[11px]">groupBy</code> value
            equals the lane value.
          </p>
        ) : null}
      </div>
    </div>
  );
}

function BoardLaneRow({
  lane,
  onChange,
  onRemove,
}: {
  lane: WidgetBoardLane;
  onChange: (patch: Partial<WidgetBoardLane>) => void;
  onRemove: () => void;
}) {
  return (
    <div className="flex gap-2 rounded-lg bg-slate-100 p-2 dark:bg-gray-800" data-testid="board-lane-row">
      <div className="grid min-w-0 flex-1 grid-cols-12 items-center gap-2">
        <Input
          className="col-span-5 h-8"
          value={lane.value}
          onChange={(e) => onChange({ value: e.target.value })}
          placeholder="Value (e.g. Done)"
          data-testid="board-lane-value"
        />
        <Input
          className="col-span-4 h-8"
          value={lane.label ?? ""}
          onChange={(e) => onChange({ label: e.target.value || undefined })}
          placeholder="Header label (optional)"
        />
        <Select
          value={lane.color ?? NEUTRAL_LANE_COLOR}
          onValueChange={(v) => onChange({ color: v as WidgetBoardLaneColor })}
        >
          <SelectTrigger className="col-span-3 h-8" data-testid="board-lane-color">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {WIDGET_BOARD_LANE_COLORS.map((c) => (
              <SelectItem key={c} value={c}>
                {c}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="flex shrink-0 items-start justify-end">
        <Button
          type="button"
          size="icon"
          variant="ghost"
          className="h-6 w-6 cursor-pointer text-slate-500 hover:bg-red-50 hover:text-red-600 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-red-400"
          onClick={onRemove}
          aria-label="Remove lane"
        >
          <Trash2 className="size-3.5" />
        </Button>
      </div>
    </div>
  );
}

export function BoardCardSection({
  value,
  fieldOptions,
  actions,
  onChange,
}: {
  value: BoardPanelContent;
  fieldOptions: string[];
  actions: BoardPanelFormActions;
  onChange: (next: BoardPanelContent) => void;
}) {
  const cardFields = value.render.card.fields ?? [];
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Card</Label>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={actions.addCardField}
          data-testid="board-add-card-field"
        >
          Add field
        </Button>
      </div>
      <div className="space-y-1.5">
        <Label className="text-[11px] font-medium text-slate-500 dark:text-gray-400">Title field</Label>
        <Input
          value={value.render.card.titleField}
          onChange={(e) =>
            onChange({
              ...value,
              render: { ...value.render, card: { ...value.render.card, titleField: e.target.value } },
            })
          }
          placeholder="e.g. title or {{ payload.name }}"
          list={fieldOptions.length > 0 ? "board-field-options" : undefined}
          data-testid="board-card-title-field"
        />
      </div>
      <div className="space-y-2">
        {cardFields.map((col, idx) => (
          <ColumnRow
            key={idx}
            col={col}
            fieldOptions={fieldOptions}
            onChange={(patch) => actions.updateCardField(idx, patch)}
            onRemove={() => actions.removeCardField(idx)}
          />
        ))}
        {cardFields.length === 0 ? (
          <p className="text-xs text-slate-500 dark:text-gray-400">
            Optional extra fields shown under the card title. Each field reuses the table column formatting vocabulary.
          </p>
        ) : null}
      </div>
    </div>
  );
}

export function BoardFiltersSection({
  value,
  fieldOptions,
  actions,
}: {
  value: BoardPanelContent;
  fieldOptions: string[];
  actions: BoardPanelFormActions;
}) {
  const filters = value.render.where ?? [];
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Filters</Label>
        <Button type="button" size="sm" variant="outline" onClick={actions.addFilter} data-testid="board-add-filter">
          Add filter
        </Button>
      </div>
      <div className="space-y-2">
        {filters.map((filter: WidgetTableFilter, idx) => (
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

export function BoardSortSection({
  value,
  fieldOptions,
  actions,
}: {
  value: BoardPanelContent;
  fieldOptions: string[];
  actions: BoardPanelFormActions;
}) {
  const sort = value.render.sort;
  const sortField = sort?.field ?? "";
  const sortOrder: WidgetSortOrder = sort?.order ?? "asc";
  const hasSortField = sortField.trim() !== "";
  const datalistId = fieldOptions.length > 0 ? "board-sort-field-options" : undefined;
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Sort within lane (optional)</Label>
      <div className="grid grid-cols-3 gap-2">
        <Input
          className="col-span-2 h-8"
          list={datalistId}
          value={sortField}
          onChange={(e) =>
            actions.setSort(e.target.value.trim() ? { field: e.target.value, order: sort?.order } : undefined)
          }
          placeholder="e.g. updatedAt (blank = unsorted)"
          data-testid="board-sort-field"
        />
        <Select
          value={sortOrder}
          onValueChange={(v) => actions.setSort({ field: sortField, order: v as WidgetSortOrder })}
          disabled={!hasSortField}
        >
          <SelectTrigger className="h-8 w-full" data-testid="board-sort-order">
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

export function BoardRowActionsSection({
  value,
  triggerNodes,
  fieldOptions,
  sampleRow,
  payloadDrafts,
  actions,
}: {
  value: BoardPanelContent;
  triggerNodes: SuperplaneComponentsNode[];
  fieldOptions: string[];
  sampleRow: Record<string, unknown>;
  payloadDrafts: RowActionPayloadDrafts;
  actions: BoardPanelFormActions;
}) {
  const rowActions = (value.render.rowActions ?? []) as WidgetRowAction[];
  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between">
        <Label className="text-xs font-medium text-slate-600 dark:text-gray-400">Row actions</Label>
        <Button type="button" size="sm" variant="outline" onClick={actions.addAction} data-testid="board-add-action">
          Add action
        </Button>
      </div>
      <p className="max-w-xl text-xs text-slate-500 dark:text-gray-400">
        Trigger-only, same rules as the table panel. Buttons appear at the bottom of each card.
      </p>
      <div className="space-y-3">
        {rowActions.map((action, idx) => (
          <ActionRow
            key={idx}
            action={action}
            triggerNodes={triggerNodes}
            fieldOptions={fieldOptions}
            sampleRow={sampleRow}
            payloadEntries={payloadDrafts.snapshots[idx] ?? []}
            onChange={(patch: Partial<WidgetRowAction>) => actions.updateAction(idx, patch)}
            onRemove={() => actions.removeAction(idx)}
            onPayloadEntryChange={(rowId: string, patch: Partial<Omit<PayloadDraftEntry, "rowId">>) =>
              actions.updatePayloadEntry(idx, rowId, patch)
            }
            onPayloadEntryRemove={(rowId: string) => actions.removePayloadEntry(idx, rowId)}
            onPayloadEntryQuickInsert={(field: string) => actions.quickInsertPayloadField(idx, field)}
          />
        ))}
      </div>
    </div>
  );
}

export function BoardFieldDatalists({ fieldOptions }: { fieldOptions: string[] }) {
  if (fieldOptions.length === 0) return null;
  return (
    <>
      <datalist id="board-field-options">
        {fieldOptions.map((f) => (
          <option key={f} value={f} />
        ))}
      </datalist>
      <datalist id="board-href-field-options">
        {fieldOptions.map((f) => (
          <option key={f} value={`{{ ${f} }}`} />
        ))}
      </datalist>
    </>
  );
}

// Re-exported so `WidgetTableColumn` stays a local import when consumers
// only need the type; avoids extra imports in the parent form.
export type { WidgetTableColumn };
