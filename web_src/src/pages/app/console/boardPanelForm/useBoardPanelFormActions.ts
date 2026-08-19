import type { SuperplaneComponentsNode } from "@/api-client";

import { WIDGET_BOARD_CARD_FORMATS, type BoardPanelContent } from "../boardPanelContent";
import { makeRowPanelFormActions, type RowActionPayloadDrafts } from "../tablePanelForm/useRowActionPayloadDrafts";
import type { WidgetBoardLane, WidgetTableColumn } from "../widget/types";
import { suggestColumnFormat } from "../widget/useMemoryCatalog";

/**
 * Actions surface for {@link BoardPanelForm}. Mirrors
 * `useTablePanelFormActions` in shape but scoped to `BoardPanelContent`
 * (lanes, card, row actions, filters, sort). Keeping this a hook means
 * the row-action payload drafts stay stable across renders even as the
 * user edits list items.
 */
export function useBoardPanelFormActions({
  value,
  onChange,
  triggerNodes,
  payloadDrafts,
}: {
  value: BoardPanelContent;
  onChange: (next: BoardPanelContent) => void;
  triggerNodes: SuperplaneComponentsNode[];
  payloadDrafts: RowActionPayloadDrafts;
}) {
  const laneActions = makeLaneActions(value, onChange);
  const cardActions = makeCardActions(value, onChange);
  const rowPanelActions = makeRowPanelFormActions({
    render: value.render,
    onRenderChange: (render) => onChange({ ...value, render }),
    triggerNodes,
    payloadDrafts,
  });

  return {
    ...laneActions,
    ...cardActions,
    ...rowPanelActions,
  };
}

export type BoardPanelFormActions = ReturnType<typeof useBoardPanelFormActions>;

function makeLaneActions(value: BoardPanelContent, onChange: (next: BoardPanelContent) => void) {
  const setLanes = (lanes: WidgetBoardLane[]) => onChange({ ...value, render: { ...value.render, lanes } });
  return {
    addLane: () => setLanes([...value.render.lanes, { value: "" }]),
    updateLane: (idx: number, patch: Partial<WidgetBoardLane>) =>
      setLanes(value.render.lanes.map((lane, i) => (i === idx ? { ...lane, ...patch } : lane))),
    removeLane: (idx: number) => setLanes(value.render.lanes.filter((_, i) => i !== idx)),
  };
}

function makeCardActions(value: BoardPanelContent, onChange: (next: BoardPanelContent) => void) {
  const setCardFields = (fields: WidgetTableColumn[] | undefined) => {
    const card = { ...value.render.card };
    if (fields && fields.length > 0) {
      card.fields = fields;
    } else {
      delete card.fields;
    }
    onChange({ ...value, render: { ...value.render, card } });
  };
  const currentFields = () => value.render.card.fields ?? [];
  return {
    addCardField: () => setCardFields([...currentFields(), { field: "", label: "" }]),
    updateCardField: (idx: number, patch: Partial<WidgetTableColumn>) => {
      const next = currentFields().map((col, i) => (i === idx ? applyCardFieldPatch(col, patch) : col));
      setCardFields(next);
    },
    removeCardField: (idx: number) => setCardFields(currentFields().filter((_, i) => i !== idx)),
  };
}

function applyCardFieldPatch(col: WidgetTableColumn, patch: Partial<WidgetTableColumn>): WidgetTableColumn {
  const next = { ...col, ...patch };
  // Suggest a sensible format when the user picks a new field and hasn't
  // customized the format yet. Mirrors the ColumnRow onChange behavior.
  if (patch.field && !next.format) {
    const suggestedFormat = suggestColumnFormat(patch.field);
    if (WIDGET_BOARD_CARD_FORMATS.includes(suggestedFormat)) next.format = suggestedFormat;
  }
  return next;
}
