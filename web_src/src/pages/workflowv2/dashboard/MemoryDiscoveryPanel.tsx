import { Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";

import { useMemoryCatalog } from "./widget/useMemoryCatalog";

interface MemoryDiscoveryPanelProps {
  canvasId: string | undefined;
  selectedNamespace: string;
  onSelectNamespace: (namespace: string) => void;
}

/**
 * Surfaces namespaces and entry counts discovered from live canvas memory so
 * authors don't have to guess namespace strings.
 */
export function MemoryDiscoveryPanel({ canvasId, selectedNamespace, onSelectNamespace }: MemoryDiscoveryPanelProps) {
  const { namespaces, isLoading, isEmpty } = useMemoryCatalog(canvasId, selectedNamespace);

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 rounded-md border border-dashed border-slate-200 bg-slate-50/80 px-3 py-2 text-xs text-slate-500">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        Scanning canvas memory…
      </div>
    );
  }

  if (isEmpty) {
    return (
      <div
        className="rounded-md border border-dashed border-amber-200 bg-amber-50/60 px-3 py-2 text-xs text-amber-800"
        data-testid="memory-discovery-empty"
      >
        No canvas memory entries yet. Components write to memory during runs; once data exists, namespaces and fields
        will appear here.
      </div>
    );
  }

  return (
    <div
      className="space-y-2 rounded-md border border-slate-200 bg-slate-50/80 px-3 py-2"
      data-testid="memory-discovery-panel"
    >
      <p className="text-xs font-medium text-slate-700">This data exists in your canvas memory:</p>
      <div className="flex flex-wrap gap-1.5">
        {namespaces.map((ns) => (
          <Button
            key={ns.namespace}
            type="button"
            size="sm"
            variant={selectedNamespace === ns.namespace ? "default" : "outline"}
            className="h-7 text-xs"
            onClick={() => onSelectNamespace(ns.namespace)}
            data-testid={`memory-namespace-${ns.namespace}`}
          >
            {ns.namespace}
            <span className="ml-1 opacity-70">({ns.count})</span>
          </Button>
        ))}
      </div>
      <p className="text-[11px] text-slate-500">
        Select a namespace, then add columns from the discovered fields below. Field names depend on what your workflow
        writes to memory.
      </p>
    </div>
  );
}
