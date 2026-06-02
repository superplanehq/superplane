import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

import {
  isCompositeMemoryDataSource,
  isMultiNumberContent,
  type MemoryNumberSource,
  type NumberMetric,
  type NumberPanelContent,
  type TablePanelDataSource,
} from "./panelTypes";

export type NumberSourceMode = "single" | "composite" | "multi";

export function detectMode(value: NumberPanelContent): NumberSourceMode {
  if (isMultiNumberContent(value)) return "multi";
  if (isCompositeMemoryDataSource(value.dataSource)) return "composite";
  return "single";
}

/**
 * Convert number-panel content between source modes. Exposed (alongside the
 * toggle component) so the conversion behavior can be unit-tested without
 * rendering the React component.
 */
export function convertNumberPanelMode(target: NumberSourceMode, value: NumberPanelContent): NumberPanelContent {
  const current = detectMode(value);
  if (target === current) return value;
  if (target === "single") return toSingleMode(value, current);
  if (target === "composite") return toCompositeMode(value, current);
  return toMultiMode(value, current);
}

export function NumberPanelSourceModeToggle({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const mode = detectMode(value);

  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600">Source mode</Label>
      <div className="flex flex-wrap gap-1">
        <Button
          type="button"
          size="sm"
          variant={mode === "single" ? "secondary" : "outline"}
          onClick={() => switchTo("single", mode, value, onChange)}
          data-testid="number-mode-simple"
        >
          Single source
        </Button>
        <Button
          type="button"
          size="sm"
          variant={mode === "composite" ? "secondary" : "outline"}
          onClick={() => switchTo("composite", mode, value, onChange)}
          data-testid="number-mode-composite"
        >
          Multiple memory sources
        </Button>
        <Button
          type="button"
          size="sm"
          variant={mode === "multi" ? "secondary" : "outline"}
          onClick={() => switchTo("multi", mode, value, onChange)}
          data-testid="number-mode-multi"
        >
          Multiple numbers
        </Button>
      </div>
    </div>
  );
}

function switchTo(
  target: NumberSourceMode,
  current: NumberSourceMode,
  value: NumberPanelContent,
  onChange: (next: NumberPanelContent) => void,
): void {
  if (target === current) return;
  onChange(convertNumberPanelMode(target, value));
}

function toSingleMode(value: NumberPanelContent, current: NumberSourceMode): NumberPanelContent {
  if (current === "multi") return singleFromMulti(value);
  return singleFromComposite(value);
}

function singleFromMulti(value: NumberPanelContent): NumberPanelContent {
  const first = value.metrics?.[0];
  return {
    title: value.title,
    dataSource: first?.dataSource ?? { kind: "runs", limit: 100 },
    render: first?.render ?? { kind: "number", aggregation: "count" },
  };
}

function singleFromComposite(value: NumberPanelContent): NumberPanelContent {
  const composite = value.dataSource;
  if (!composite || !isCompositeMemoryDataSource(composite)) return value;
  const first = composite.sources[0];
  return {
    title: value.title,
    dataSource: { kind: "memory", namespace: first?.namespace ?? "", fieldPath: first?.fieldPath },
    render: {
      ...(value.render ?? { kind: "number" }),
      kind: "number",
      aggregation: first?.aggregation ?? "count",
      field: first?.field,
    },
  };
}

function toCompositeMode(value: NumberPanelContent, current: NumberSourceMode): NumberPanelContent {
  const sources =
    current === "multi" ? compositeSourcesFromMetrics(value.metrics ?? []) : [compositeSeedFromSingle(value)];
  // Preserve presentation options (format, prefix, suffix, label, sparklineField)
  // configured before the switch. Aggregation and field move into the composite
  // sources, so clear them on the top-level render.
  const baseRender = current === "multi" ? (value.metrics?.[0]?.render ?? { kind: "number" }) : value.render;
  return {
    title: value.title,
    dataSource: { kind: "memory", sources, combine: "sum" },
    render: { ...(baseRender ?? { kind: "number" }), kind: "number", aggregation: undefined, field: undefined },
  };
}

function compositeSourcesFromMetrics(metrics: NumberMetric[]): MemoryNumberSource[] {
  const memoryMetrics = metrics.filter((m) => m.dataSource.kind === "memory");
  if (memoryMetrics.length === 0) return [{ namespace: "", aggregation: "count" }];
  return memoryMetrics.map((m) => ({
    namespace: m.dataSource.kind === "memory" ? m.dataSource.namespace : "",
    aggregation: m.render.aggregation ?? "count",
    field: m.render.field,
    fieldPath: m.dataSource.kind === "memory" ? m.dataSource.fieldPath : undefined,
  }));
}

function compositeSeedFromSingle(value: NumberPanelContent): MemoryNumberSource {
  const ds = value.dataSource;
  if (ds && ds.kind === "memory" && !isCompositeMemoryDataSource(ds)) {
    return {
      namespace: ds.namespace || "",
      aggregation: value.render?.aggregation ?? "count",
      field: value.render?.field,
      fieldPath: ds.fieldPath,
    };
  }
  return { namespace: "", aggregation: value.render?.aggregation ?? "count", field: value.render?.field };
}

function toMultiMode(value: NumberPanelContent, current: NumberSourceMode): NumberPanelContent {
  const metrics = current === "composite" ? multiMetricsFromComposite(value) : [multiSeedFromSingle(value)];
  return { title: value.title, metrics };
}

function multiMetricsFromComposite(value: NumberPanelContent): NumberMetric[] {
  const ds = value.dataSource;
  const sources = ds && isCompositeMemoryDataSource(ds) ? ds.sources : [];
  if (sources.length === 0) return [multiSeedFromSingle(value)];
  // Map every composite source to its own metric (mirroring
  // compositeSourcesFromMetrics) instead of keeping only the first. Per-source
  // aggregation/field move onto each metric's render while the top-level
  // presentation options (label, format, prefix, suffix, sparklineField) are
  // preserved so styling survives the switch.
  const baseRender = value.render ?? { kind: "number" };
  return sources.map((source) => ({
    dataSource: {
      kind: "memory",
      namespace: source.namespace,
      fieldPath: source.fieldPath,
    },
    render: {
      ...baseRender,
      kind: "number",
      aggregation: source.aggregation,
      field: source.field,
    },
  }));
}

function multiSeedFromSingle(value: NumberPanelContent): NumberMetric {
  const ds = (value.dataSource as TablePanelDataSource | undefined) ?? { kind: "runs", limit: 100 };
  return {
    dataSource: isCompositeMemoryDataSource(ds) ? { kind: "runs", limit: 100 } : ds,
    render: value.render ?? { kind: "number", aggregation: "count" },
  };
}
