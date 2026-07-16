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
import { useTablePanelPayloadDrafts } from "./tablePanelForm/useTablePanelPayloadDrafts";
import { useWidgetExpressionContext } from "./widget/useWidgetExpressionContext";

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
  const { row: sampleRow, fields } = useWidgetExpressionContext({
    canvasId: canvasId ?? "",
    dataSource: value.dataSource,
    render: value.render,
  });
  const fieldOptions = fields.map((f) => f.field);
  const payloadDrafts = useTablePanelPayloadDrafts(value);
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
      <TablePanelColumnsSection
        value={value}
        fields={fields}
        fieldOptions={fieldOptions}
        sampleRow={sampleRow}
        actions={actions}
      />
      <TablePanelFiltersSection value={value} fieldOptions={fieldOptions} sampleRow={sampleRow} actions={actions} />
      <TablePanelRowStylesSection value={value} fieldOptions={fieldOptions} sampleRow={sampleRow} actions={actions} />
      <TablePanelSortSection value={value} fieldOptions={fieldOptions} sampleRow={sampleRow} actions={actions} />
      <TablePanelRowActionsSection
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
