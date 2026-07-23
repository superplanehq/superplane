import type { BoardPanelContent } from "./boardPanelContent";
import {
  BoardCardSection,
  BoardFieldDatalists,
  BoardFiltersSection,
  BoardHeaderFields,
  BoardLanesSection,
  BoardRowActionsSection,
  BoardSortSection,
} from "./boardPanelForm/BoardPanelFormSections";
import { useBoardPanelFormActions } from "./boardPanelForm/useBoardPanelFormActions";
import { useConsoleContext } from "./ConsoleContext";
import { DataSourceForm } from "./DataSourceForm";
import { isManualRunNode } from "./manualRunTriggers";
import { MemoryDiscoveryPanel } from "./MemoryDiscoveryPanel";
import { useRowActionPayloadDrafts } from "./tablePanelForm/useRowActionPayloadDrafts";
import { staticFieldsForDataSource } from "./widget/staticFieldCatalogs";
import { sampleRowFromFields, useMemoryCatalog, type MemoryFieldSummary } from "./widget/useMemoryCatalog";

interface BoardPanelFormProps {
  value: BoardPanelContent;
  onChange: (next: BoardPanelContent) => void;
}

/**
 * Editor for the `board` panel. Reuses the table panel's DataSourceForm,
 * memory discovery picker, filter row, action row, and row-action payload
 * drafts so authors get the same fluent authoring experience for both
 * table and board — with lane / card configuration layered on top of the
 * shared row-data plumbing.
 */
export function BoardPanelForm({ value, onChange }: BoardPanelFormProps) {
  const ctx = useConsoleContext();
  const canvasId = ctx?.canvasId;
  const triggerNodes = (ctx?.nodes ?? []).filter(isManualRunNode);
  const namespace = value.dataSource.kind === "memory" ? value.dataSource.namespace : "";
  const { fields: memoryFields } = useMemoryCatalog(canvasId, namespace);
  const fields = resolveFieldCatalog(value, memoryFields);
  const fieldOptions = fields.map((f) => f.field);
  const sampleRow = sampleRowFromFields(fields);
  const payloadDrafts = useRowActionPayloadDrafts(value.render.rowActions);
  const actions = useBoardPanelFormActions({ value, onChange, triggerNodes, payloadDrafts });

  return (
    <div className="space-y-4">
      <BoardHeaderFields value={value} fieldOptions={fieldOptions} onChange={onChange} />
      <DataSourceForm
        value={value.dataSource}
        onChange={(dataSource) => onChange({ ...value, dataSource })}
        loadAllWhenBlank
      />
      <BoardMemorySourcePicker value={value} canvasId={canvasId} onChange={onChange} />

      <BoardLanesSection value={value} actions={actions} />
      <BoardCardSection value={value} fieldOptions={fieldOptions} actions={actions} onChange={onChange} />
      <BoardFiltersSection value={value} fieldOptions={fieldOptions} actions={actions} />
      <BoardSortSection value={value} fieldOptions={fieldOptions} actions={actions} />
      <BoardRowActionsSection
        value={value}
        triggerNodes={triggerNodes}
        fieldOptions={fieldOptions}
        sampleRow={sampleRow}
        payloadDrafts={payloadDrafts}
        actions={actions}
      />
      <BoardFieldDatalists fieldOptions={fieldOptions} />
    </div>
  );
}

function resolveFieldCatalog(value: BoardPanelContent, memoryFields: MemoryFieldSummary[]): MemoryFieldSummary[] {
  if (value.dataSource.kind === "memory") return memoryFields;
  return staticFieldsForDataSource(value.dataSource.kind);
}

function BoardMemorySourcePicker({
  value,
  canvasId,
  onChange,
}: {
  value: BoardPanelContent;
  canvasId: string | undefined;
  onChange: (next: BoardPanelContent) => void;
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
