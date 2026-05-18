import { useDashboardContext } from "./DashboardContext";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

import type { ChartPanelDataSource } from "./panelTypes";

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
  const ctx = useDashboardContext();
  const nodes = ctx?.nodes ?? [];

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
    <div className="space-y-3 rounded-md border border-slate-200 bg-slate-50/40 p-3">
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
        hideLimit ? null : (
          <div className="space-y-1.5">
            <Label className="text-xs font-medium text-slate-600">Limit</Label>
            <Input
              type="number"
              min={1}
              value={value.limit ?? ""}
              onChange={(e) =>
                onChange({
                  ...value,
                  limit: e.target.value === "" ? undefined : Number(e.target.value),
                })
              }
              placeholder="100"
            />
          </div>
        )
      ) : value.kind === "executions" ? (
        <>
          <div className="space-y-1.5">
            <Label className="text-xs font-medium text-slate-600">Node (optional)</Label>
            <Select
              value={value.node ?? "__all__"}
              onValueChange={(v) =>
                onChange({
                  ...value,
                  node: v === "__all__" ? undefined : v,
                })
              }
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="All nodes" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">All nodes</SelectItem>
                {nodes.map((n) => (
                  <SelectItem key={n.id} value={n.name || n.id || ""}>
                    {n.name || n.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {hideLimit ? null : (
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-slate-600">Limit</Label>
              <Input
                type="number"
                min={1}
                value={value.limit ?? ""}
                onChange={(e) =>
                  onChange({
                    ...value,
                    limit: e.target.value === "" ? undefined : Number(e.target.value),
                  })
                }
                placeholder="50"
              />
            </div>
          )}
        </>
      ) : (
        <>
          <div className="space-y-1.5">
            <Label className="text-xs font-medium text-slate-600">Namespace</Label>
            <Input
              value={value.namespace}
              onChange={(e) => onChange({ ...value, namespace: e.target.value })}
              placeholder="e.g. deployments"
            />
          </div>
          <div className="space-y-1.5">
            <Label className="text-xs font-medium text-slate-600">Field path (optional)</Label>
            <Input
              value={value.fieldPath ?? ""}
              onChange={(e) => onChange({ ...value, fieldPath: e.target.value || undefined })}
              placeholder="e.g. items"
            />
          </div>
        </>
      )}
    </div>
  );
}
