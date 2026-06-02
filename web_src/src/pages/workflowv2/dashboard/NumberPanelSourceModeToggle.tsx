import { Label } from "@/components/ui/label";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { isCompositeMemoryDataSource, type MemoryNumberSource, type NumberPanelContent } from "./panelTypes";

export function NumberPanelSourceModeToggle({
  value,
  onChange,
}: {
  value: NumberPanelContent;
  onChange: (next: NumberPanelContent) => void;
}) {
  const dataSource = value.dataSource;
  const composite = isCompositeMemoryDataSource(dataSource);
  const mode = composite ? "composite" : "simple";

  const switchToComposite = () => {
    if (isCompositeMemoryDataSource(dataSource)) return;
    const seed: MemoryNumberSource =
      dataSource.kind === "memory"
        ? {
            namespace: dataSource.namespace || "",
            aggregation: value.render.aggregation ?? "count",
            field: value.render.field,
            fieldPath: dataSource.fieldPath,
          }
        : { namespace: "", aggregation: value.render.aggregation ?? "count", field: value.render.field };
    onChange({
      ...value,
      dataSource: { kind: "memory", sources: [seed], combine: "sum" },
      render: { ...value.render, aggregation: undefined, field: undefined },
    });
  };

  const switchToSimple = () => {
    if (!isCompositeMemoryDataSource(dataSource)) return;
    const first = dataSource.sources[0];
    onChange({
      ...value,
      dataSource: { kind: "memory", namespace: first?.namespace ?? "", fieldPath: first?.fieldPath },
      render: { ...value.render, aggregation: first?.aggregation ?? "count", field: first?.field },
    });
  };

  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-slate-600">Source mode</Label>
      <Tabs
        value={mode}
        onValueChange={(next) => {
          if (next === "composite") switchToComposite();
          else switchToSimple();
        }}
      >
        <TabsList>
          <TabsTrigger value="simple" data-testid="number-mode-simple">
            Single source
          </TabsTrigger>
          <TabsTrigger value="composite" data-testid="number-mode-composite">
            Multiple memory sources
          </TabsTrigger>
        </TabsList>
      </Tabs>
    </div>
  );
}
