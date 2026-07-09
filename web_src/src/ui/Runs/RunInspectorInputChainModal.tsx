import { Check, Copy } from "lucide-react";
import { useEffect, useMemo, useState, type CSSProperties } from "react";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { cn } from "@/lib/utils";
import type { RunInspectorUpstreamSection } from "./runNodeDetailModel";
import { HeaderIconButton, JsonPayload } from "./RunInspectorTimelineCard";
import { NodeMarker } from "./RunInspectorTimelineMarkers";

export function InputChainMoreChip({
  count,
  sections,
  initialSelectedNodeId,
  componentIconMap,
  jsonViewStyle,
}: {
  count: number;
  sections: RunInspectorUpstreamSection[];
  initialSelectedNodeId?: string;
  componentIconMap: Record<string, string>;
  jsonViewStyle: CSSProperties;
}) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        type="button"
        title="Open input chain"
        onClick={(event) => {
          event.stopPropagation();
          setOpen(true);
        }}
        className="flex shrink-0 items-center rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-600 transition-colors hover:bg-slate-200 hover:text-slate-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"
      >
        +{count} more
      </button>
      <InputChainModal
        open={open}
        onOpenChange={setOpen}
        sections={sections}
        initialSelectedNodeId={initialSelectedNodeId}
        componentIconMap={componentIconMap}
        jsonViewStyle={jsonViewStyle}
      />
    </>
  );
}

function InputChainModal({
  open,
  onOpenChange,
  sections,
  initialSelectedNodeId,
  componentIconMap,
  jsonViewStyle,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sections: RunInspectorUpstreamSection[];
  initialSelectedNodeId?: string;
  componentIconMap: Record<string, string>;
  jsonViewStyle: CSSProperties;
}) {
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const selected =
    sections.find((section) => section.nodeId === selectedNodeId) ??
    sections.find((section) => section.nodeId === initialSelectedNodeId) ??
    sections.at(-1);
  const payloadString = useMemo(() => JSON.stringify(selected?.output ?? {}, null, 2), [selected?.output]);

  useEffect(() => {
    if (!open) return;

    setSelectedNodeId(initialSelectedNodeId ?? sections.at(-1)?.nodeId ?? null);
  }, [initialSelectedNodeId, open, sections]);

  const copyPayload = () => {
    void navigator.clipboard?.writeText(payloadString).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className="flex h-[80vh] w-[70vw] max-w-[70vw] flex-col gap-0 overflow-hidden p-0"
        onClick={(event) => event.stopPropagation()}
      >
        <DialogTitle className="sr-only">Input chain</DialogTitle>
        <div className="flex min-h-0 flex-1">
          <div className="flex w-56 shrink-0 flex-col gap-0.5 overflow-y-auto border-r border-slate-200 bg-slate-50 p-2 dark:border-gray-800 dark:bg-gray-900">
            <div className="px-2 py-1 text-[11px] font-semibold uppercase tracking-wide text-slate-400">
              Input chain
            </div>
            {sections.map((section) => (
              <button
                key={section.nodeId}
                type="button"
                onClick={() => setSelectedNodeId(section.nodeId)}
                className={cn(
                  "flex items-center gap-2 rounded px-2 py-1.5 text-left text-[12px] transition-colors",
                  selected?.nodeId === section.nodeId
                    ? "bg-white font-medium text-slate-900 shadow-sm ring-1 ring-slate-200 dark:bg-gray-950 dark:text-gray-100 dark:ring-gray-800"
                    : "text-slate-600 hover:bg-slate-100 dark:text-gray-300 dark:hover:bg-gray-800",
                )}
              >
                <NodeMarker section={section} fallbackLabel={section.nodeName} componentIconMap={componentIconMap} />
                <span className="min-w-0 truncate">{section.nodeName}</span>
              </button>
            ))}
          </div>
          <div className="flex min-w-0 flex-1 flex-col">
            <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-1.5 pr-10 dark:border-gray-800 dark:bg-gray-900">
              <div className="flex min-w-0 items-center gap-1.5">
                {selected ? (
                  <NodeMarker
                    section={selected}
                    fallbackLabel={selected.nodeName}
                    componentIconMap={componentIconMap}
                  />
                ) : null}
                <span className="truncate text-[12px] font-medium text-slate-700 dark:text-gray-200">
                  {selected?.nodeName}
                </span>
                <span className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-500">
                  Output
                </span>
              </div>
              <div className="flex items-center gap-0.5">
                <HeaderIconButton
                  label={copied ? "Copied" : "Copy"}
                  icon={copied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />}
                  onClick={copyPayload}
                />
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-auto p-3">
              <JsonPayload value={selected?.output} jsonViewStyle={jsonViewStyle} collapsed={false} />
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
