import { useConsoleContext } from "./ConsoleContext";
import { DataSourceForm } from "./DataSourceForm";
import { isManualRunNode } from "./manualRunTriggers";
import type { TablePanelContent } from "./panelTypes";
import {
  TablePanelColumnsSection,
  TablePanelFiltersSection,
  TablePanelMemorySourcePicker,
  TablePanelRowActionsSection,
  TablePanelRowStylesSection,
  TablePanelSortSection,
  TablePanelTitleField,
} from "./tablePanelForm/TablePanelFormSections";
import { useTablePanelFormActions } from "./tablePanelForm/useTablePanelFormActions";
import { useRowActionPayloadDrafts } from "./tablePanelForm/useRowActionPayloadDrafts";
import { staticFieldsForDataSource } from "./widget/staticFieldCatalogs";
import { sampleRowFromFields, useMemoryCatalog, type MemoryFieldSummary } from "./widget/useMemoryCatalog";

interface TablePanelFormProps {
  value: TablePanelContent;
  onChange: (next: TablePanelContent) => void;
}

export function TablePanelForm({ value, onChange }: TablePanelFormProps) {
  const ctx = useConsoleContext();
  const canvasId = ctx?.canvasId;
  // Row actions fire the user-invokable `run` hook, so the editor's node
  // dropdown only exposes triggers the backend will actually accept —
  // `TYPE_TRIGGER` nodes whose component is in the manual-run allowlist.
  const triggerNodes = (ctx?.nodes ?? []).filter(isManualRunNode);
  const namespace = value.dataSource.kind === "memory" ? value.dataSource.namespace : "";
  const { fields: memoryFields } = useMemoryCatalog(canvasId, namespace);
  const fields = resolveFieldCatalog(value, memoryFields);
  const fieldOptions = fields.map((f) => f.field);
  const sampleRow = sampleRowFromFields(fields);
  const payloadDrafts = useRowActionPayloadDrafts(value.render.rowActions);
  const actions = useTablePanelFormActions({ value, onChange, fields, triggerNodes, payloadDrafts });

  return (
    <div className="space-y-4">
      <TablePanelTitleField value={value} onChange={onChange} />
      <DataSourceForm
        value={value.dataSource}
        onChange={(dataSource) => onChange({ ...value, dataSource })}
        loadAllWhenBlank
      />
      <TablePanelMemorySourcePicker value={value} canvasId={canvasId} onChange={onChange} />
      <TablePanelColumnsSection value={value} fields={fields} fieldOptions={fieldOptions} actions={actions} />
      <TablePanelFiltersSection value={value} fieldOptions={fieldOptions} actions={actions} />
      <TablePanelRowStylesSection value={value} fieldOptions={fieldOptions} actions={actions} />
      <TablePanelSortSection value={value} fieldOptions={fieldOptions} actions={actions} />
      <TablePanelRowActionsSection
        value={value}
        triggerNodes={triggerNodes}
        fieldOptions={fieldOptions}
        sampleRow={sampleRow}
        payloadDrafts={payloadDrafts}
        actions={actions}
      />
      {fieldOptions.length > 0 ? (
        <datalist id="table-field-options">
          {fieldOptions.map((f) => (
            <option key={f} value={f} />
          ))}
        </datalist>
      ) : null}
      {fieldOptions.length > 0 ? (
        <datalist id="table-href-field-options">
          {fieldOptions.map((f) => (
            <option key={f} value={`{{ ${f} }}`} />
          ))}
        </datalist>
      ) : null}
    </div>
  );
}

/**
 * Pick the right field catalog for the configured data source. Memory rows
 * are dynamic (discovered from the live canvas memory), while executions
 * and runs have fixed shapes — see `staticFieldsForDataSource` for the
 * hard-coded catalogs. Returns an empty list when no suggestions are
 * available, so the form falls back to free-text input cleanly.
 */
function resolveFieldCatalog(value: TablePanelContent, memoryFields: MemoryFieldSummary[]): MemoryFieldSummary[] {
  if (value.dataSource.kind === "memory") return memoryFields;
  return staticFieldsForDataSource(value.dataSource.kind);
}
