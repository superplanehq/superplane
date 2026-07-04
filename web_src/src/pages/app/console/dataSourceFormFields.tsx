import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { SuperplaneComponentsNode } from "@/api-client";

import type { ChartPanelDataSource } from "./panelTypes";

export function DataSourceRunsFields({
  value,
  hideLimit,
  loadAllWhenBlank,
  onChange,
}: {
  value: Extract<ChartPanelDataSource, { kind: "runs" }>;
  hideLimit?: boolean;
  loadAllWhenBlank?: boolean;
  onChange: (next: ChartPanelDataSource) => void;
}) {
  if (hideLimit) return null;

  return (
    <LimitField
      value={value.limit}
      loadAllWhenBlank={loadAllWhenBlank}
      defaultPlaceholder="100"
      onChange={(limit) => onChange({ ...value, limit })}
    />
  );
}

export function DataSourceExecutionsFields({
  value,
  hideLimit,
  loadAllWhenBlank,
  nodes,
  onChange,
}: {
  value: Extract<ChartPanelDataSource, { kind: "executions" }>;
  hideLimit?: boolean;
  loadAllWhenBlank?: boolean;
  nodes: SuperplaneComponentsNode[];
  onChange: (next: ChartPanelDataSource) => void;
}) {
  return (
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
        <LimitField
          value={value.limit}
          loadAllWhenBlank={loadAllWhenBlank}
          defaultPlaceholder="50"
          onChange={(limit) => onChange({ ...value, limit })}
        />
      )}
    </>
  );
}

function LimitField({
  value,
  loadAllWhenBlank,
  defaultPlaceholder,
  onChange,
}: {
  value: number | undefined;
  loadAllWhenBlank: boolean | undefined;
  defaultPlaceholder: string;
  onChange: (limit: number | undefined) => void;
}) {
  const placeholder = loadAllWhenBlank ? "On demand" : defaultPlaceholder;
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600">Limit</Label>
      <Input
        type="number"
        min={1}
        value={value ?? ""}
        onChange={(e) => onChange(e.target.value === "" ? undefined : Number(e.target.value))}
        placeholder={placeholder}
        data-testid="data-source-limit"
      />
      {loadAllWhenBlank ? (
        <p className="text-[11px] text-slate-500">
          Leave blank to load rows on demand — scroll or use the "Load more" button to reveal more. Very long histories
          are capped for performance; set a number to fix how many rows are fetched.
        </p>
      ) : null}
    </div>
  );
}

export function DataSourceMemoryFields({
  value,
  namespaces,
  fields,
  namespaceListId,
  fieldPathListId,
  onChange,
}: {
  value: Extract<ChartPanelDataSource, { kind: "memory" }>;
  namespaces: { namespace: string }[];
  fields: { field: string }[];
  namespaceListId: string | undefined;
  fieldPathListId: string | undefined;
  onChange: (next: ChartPanelDataSource) => void;
}) {
  return (
    <>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Namespace</Label>
        <Input
          list={namespaces.length > 0 && namespaceListId ? namespaceListId : undefined}
          value={value.namespace}
          onChange={(e) => onChange({ ...value, namespace: e.target.value })}
          placeholder="e.g. deployments"
          data-testid="data-source-namespace"
        />
        {namespaces.length > 0 && namespaceListId ? (
          <datalist id={namespaceListId}>
            {namespaces.map((n) => (
              <option key={n.namespace} value={n.namespace} />
            ))}
          </datalist>
        ) : null}
      </div>
      <div className="space-y-1.5">
        <Label className="text-xs font-medium text-slate-600">Field path (optional)</Label>
        <Input
          list={fields.length > 0 && fieldPathListId ? fieldPathListId : undefined}
          value={value.fieldPath ?? ""}
          onChange={(e) => onChange({ ...value, fieldPath: e.target.value || undefined })}
          placeholder="e.g. items"
          data-testid="data-source-field-path"
        />
        {fields.length > 0 && fieldPathListId ? (
          <datalist id={fieldPathListId}>
            {fields.map((f) => (
              <option key={f.field} value={f.field} />
            ))}
          </datalist>
        ) : null}
      </div>
    </>
  );
}
