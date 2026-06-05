import { useConsoleContext } from "./ConsoleContext";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import type { ChartPanelDataSource } from "./panelTypes";
import { DataSourceExecutionsFields, DataSourceMemoryFields, DataSourceRunsFields } from "./dataSourceFormFields";
import { useMemoryCatalog } from "./widget/useMemoryCatalog";

interface DataSourceFormProps {
  value: ChartPanelDataSource;
  onChange: (next: ChartPanelDataSource) => void;
  /** Hide the limit input (e.g. number panels that aggregate everything). */
  hideLimit?: boolean;
}

/**
 * Shared editor for `panel.content.dataSource`. Used by the table, chart, and
 * number form editors. Switching `kind` resets the kind-specific fields to
 * sensible defaults so the resulting object always matches the validator.
 */
export function DataSourceForm({ value, onChange, hideLimit }: DataSourceFormProps) {
  const ctx = useConsoleContext();
  const nodes = ctx?.nodes ?? [];
  const canvasId = ctx?.canvasId;
  const memoryNamespace = value.kind === "memory" ? value.namespace : undefined;
  const { namespaces, fields } = useMemoryCatalog(canvasId, memoryNamespace);
  const namespaceListId = canvasId ? `data-source-namespaces-${canvasId}` : undefined;
  const fieldPathListId = memoryNamespace ? `data-source-field-paths-${memoryNamespace}` : undefined;

  const setKind = (kind: "memory" | "executions" | "runs") => {
    if (kind === "memory") {
      onChange({ kind: "memory", namespace: "" });
    } else if (kind === "runs") {
      onChange({ kind: "runs", limit: 100 });
    } else {
      onChange({ kind: "executions", limit: 50 });
    }
  };

  return (
    <div className="space-y-3 rounded-lg bg-slate-100 p-3">
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Source</Label>
        <Select value={value.kind} onValueChange={(v) => setKind(v as "memory" | "executions" | "runs")}>
          <SelectTrigger className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="runs">Runs</SelectItem>
            <SelectItem value="executions">Executions</SelectItem>
            <SelectItem value="memory">Memory</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {value.kind === "runs" ? (
        <DataSourceRunsFields value={value} hideLimit={hideLimit} onChange={onChange} />
      ) : value.kind === "executions" ? (
        <DataSourceExecutionsFields value={value} hideLimit={hideLimit} nodes={nodes} onChange={onChange} />
      ) : (
        <DataSourceMemoryFields
          value={value}
          namespaces={namespaces}
          fields={fields}
          namespaceListId={namespaceListId}
          fieldPathListId={fieldPathListId}
          onChange={onChange}
        />
      )}
    </div>
  );
}
