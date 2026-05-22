import { CanvasMemoryView, type CanvasMemoryViewProps } from "./CanvasMemoryView";

export type MemoryOverlayProps = CanvasMemoryViewProps;

export function MemoryOverlay(props: MemoryOverlayProps) {
  return (
    <div
      className="absolute inset-x-0 bottom-0 z-10 flex flex-col bg-slate-100 top-[5rem]"
      data-testid="memory-overlay"
    >
      <div className="flex shrink-0 items-center justify-between border-b border-gray-200 bg-white px-4 py-3">
        <span className="font-mono text-sm text-gray-600">Canvas Memory</span>
      </div>
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden bg-slate-50">
        <CanvasMemoryView {...props} />
      </div>
    </div>
  );
}
