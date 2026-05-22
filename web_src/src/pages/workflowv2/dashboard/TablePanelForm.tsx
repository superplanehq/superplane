import { useDashboardContext } from "./DashboardContext";
import { DataSourceForm } from "./DataSourceForm";
import type { TablePanelContent } from "./panelTypes";
import {
  TablePanelColumnsSection,
  TablePanelFiltersSection,
  TablePanelMemorySourcePicker,
  TablePanelRowActionsSection,
  TablePanelTitleField,
} from "./tablePanelForm/TablePanelFormSections";
import { useTablePanelFormActions } from "./tablePanelForm/useTablePanelFormActions";
import { useTablePanelPayloadDrafts } from "./tablePanelForm/useTablePanelPayloadDrafts";
import { sampleRowFromFields, useMemoryCatalog } from "./widget/useMemoryCatalog";

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
  const payloadDrafts = useTablePanelPayloadDrafts(value);
  const actions = useTablePanelFormActions({ value, onChange, fields, triggerNodes, payloadDrafts });

  return (
    <div className="space-y-4">
      <TablePanelTitleField value={value} onChange={onChange} />
      <DataSourceForm value={value.dataSource} onChange={(dataSource) => onChange({ ...value, dataSource })} />
      <TablePanelMemorySourcePicker value={value} canvasId={canvasId} onChange={onChange} />
      <TablePanelColumnsSection value={value} fields={fields} fieldOptions={fieldOptions} actions={actions} />
      <TablePanelFiltersSection value={value} fieldOptions={fieldOptions} actions={actions} />
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
