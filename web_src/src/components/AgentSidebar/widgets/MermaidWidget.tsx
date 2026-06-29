import { useEffect, useId, useRef, useState, useCallback } from "react";
import mermaid from "mermaid";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";

mermaid.initialize({
  startOnLoad: false,
  theme: "base",
  securityLevel: "strict",
  fontFamily: "ui-sans-serif, system-ui, sans-serif",
  themeVariables: {
    primaryColor: "#ede9fe",
    primaryTextColor: "#4c1d95",
    primaryBorderColor: "#8b5cf6",
    secondaryColor: "#ecfeff",
    secondaryTextColor: "#164e63",
    secondaryBorderColor: "#06b6d4",
    tertiaryColor: "#fffbeb",
    tertiaryTextColor: "#78350f",
    tertiaryBorderColor: "#f59e0b",
    lineColor: "#94a3b8",
    textColor: "#334155",
    nodeBorder: "#8b5cf6",
    nodeTextColor: "#1e293b",
    clusterBkg: "#f8fafc",
    clusterBorder: "#e2e8f0",
    defaultLinkColor: "#8b5cf6",
    fontSize: "13px",
  },
});

interface MermaidWidgetProps {
  content: string;
}

export function MermaidWidget({ content }: MermaidWidgetProps) {
  const id = useId().replace(/:/g, "m");
  const containerRef = useRef<HTMLDivElement>(null);
  const [svg, setSvg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState(false);

  useEffect(() => {
    let cancelled = false;

    async function render() {
      try {
        const { svg: rendered } = await mermaid.render(`mermaid-${id}`, content.trim());
        if (!cancelled) {
          setSvg(rendered);
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to render diagram");
          setSvg(null);
        }
        document.getElementById(`dmermaid-${id}`)?.remove();
      }
    }

    render();
    return () => {
      cancelled = true;
    };
  }, [content, id]);

  if (error) {
    return (
      <div className="my-4 border border-red-200 bg-red-50 rounded-lg p-2">
        <p className="text-xs text-red-600 font-medium">Diagram error</p>
        <pre className="text-xs text-red-500 mt-1 whitespace-pre-wrap">{content.trim()}</pre>
      </div>
    );
  }

  if (!svg) {
    return (
      <div className="my-4 flex items-center justify-center py-4 text-xs text-slate-400">Rendering diagram...</div>
    );
  }

  return (
    <>
      <div
        ref={containerRef}
        onClick={() => setExpanded(true)}
        className="my-4 w-full min-w-0 rounded-lg border border-slate-200 bg-white p-3 overflow-x-auto cursor-pointer hover:border-slate-300 transition-colors [&_svg]:max-w-full [&_svg]:h-auto [&_svg]:mx-auto"
      >
        <div className="pointer-events-none" dangerouslySetInnerHTML={{ __html: svg }} />
      </div>

      <Dialog open={expanded} onOpenChange={setExpanded}>
        <DialogContent size="large" className="w-[90vw] max-h-[85vh] flex flex-col">
          <DialogHeader>
            <DialogTitle className="text-sm font-medium">Diagram</DialogTitle>
          </DialogHeader>
          <MermaidPanZoom svg={svg} />
        </DialogContent>
      </Dialog>
    </>
  );
}

function MermaidPanZoom({ svg }: { svg: string }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState(2.5);
  const [translate, setTranslate] = useState({ x: 0, y: 0 });
  const dragRef = useRef<{ startX: number; startY: number; startTx: number; startTy: number } | null>(null);

  const handleWheel = useCallback((e: React.WheelEvent) => {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setScale((prev) => Math.min(Math.max(prev * delta, 0.2), 5));
  }, []);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      dragRef.current = {
        startX: e.clientX,
        startY: e.clientY,
        startTx: translate.x,
        startTy: translate.y,
      };
    },
    [translate],
  );

  const handleMouseMove = useCallback((e: React.MouseEvent) => {
    if (!dragRef.current) return;
    setTranslate({
      x: dragRef.current.startTx + (e.clientX - dragRef.current.startX),
      y: dragRef.current.startTy + (e.clientY - dragRef.current.startY),
    });
  }, []);

  const handleMouseUp = useCallback(() => {
    dragRef.current = null;
  }, []);

  const resetView = useCallback(() => {
    setScale(2.5);
    setTranslate({ x: 0, y: 0 });
  }, []);

  return (
    <div className="flex-1 min-h-0 flex flex-col">
      <div className="flex items-center gap-2 mb-2 text-xs text-slate-500">
        <button
          type="button"
          onClick={() => setScale((s) => Math.min(s * 1.2, 5))}
          className="px-2 py-1 rounded border border-slate-200 hover:bg-slate-50 cursor-pointer"
        >
          Zoom +
        </button>
        <button
          type="button"
          onClick={() => setScale((s) => Math.max(s * 0.8, 0.2))}
          className="px-2 py-1 rounded border border-slate-200 hover:bg-slate-50 cursor-pointer"
        >
          Zoom −
        </button>
        <button
          type="button"
          onClick={resetView}
          className="px-2 py-1 rounded border border-slate-200 hover:bg-slate-50 cursor-pointer"
        >
          Reset
        </button>
        <span>{Math.round(scale * 100)}%</span>
      </div>
      <div
        ref={containerRef}
        className="flex-1 min-h-0 overflow-hidden rounded-lg border border-slate-200 bg-slate-50/50 cursor-grab active:cursor-grabbing"
        onWheel={handleWheel}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
      >
        <div
          className="w-full h-full flex items-center justify-center"
          style={{
            transform: `translate(${translate.x}px, ${translate.y}px) scale(${scale})`,
            transformOrigin: "center center",
            minHeight: "40vh",
          }}
        >
          <div className="[&_svg]:max-w-none [&_svg]:h-auto" dangerouslySetInnerHTML={{ __html: svg }} />
        </div>
      </div>
    </div>
  );
}
