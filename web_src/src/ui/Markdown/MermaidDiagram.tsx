import { useCallback, useEffect, useId, useRef, useState } from "react";
import { AlertTriangle, Maximize2, Minus, Plus, RotateCcw } from "lucide-react";
import { Dialog, DialogContent } from "@/components/ui/dialog";

//
// Mermaid diagrams rendered from ```mermaid fenced code blocks.
//
// Mermaid is loaded lazily on first use — it pulls in a few hundred KB of
// diagram renderers we don't want on every page. Each diagram renders to an
// SVG string via `mermaid.render`, which we inject into a container div.
//
// Security: we run mermaid with `securityLevel: "strict"`, which routes
// user-authored labels through its HTML sanitizer and forbids raw HTML in
// node labels. The resulting SVG never contains `<script>` or `on*=`
// handlers, so inserting it via `dangerouslySetInnerHTML` is safe.
//
// Interactions: the rendered SVG is wrapped in a zoom/pan surface with a
// hover toolbar (zoom in/out, reset, expand). Expanding opens a full-screen
// modal that re-renders the diagram with the same controls plus bare wheel
// zoom — inline, wheel zoom requires Ctrl/Meta so page scroll still works.
//

let mermaidInitPromise: Promise<typeof import("mermaid").default> | null = null;

async function getMermaid() {
  if (!mermaidInitPromise) {
    mermaidInitPromise = import("mermaid").then((mod) => {
      const mermaid = mod.default;
      mermaid.initialize({
        startOnLoad: false,
        securityLevel: "strict",
        theme: "neutral",
        fontFamily: "inherit",
      });
      return mermaid;
    });
  }
  return mermaidInitPromise;
}

interface MermaidDiagramProps {
  code: string;
}

export function MermaidDiagram({ code }: MermaidDiagramProps) {
  const [isFullscreen, setIsFullscreen] = useState(false);

  return (
    <>
      <MermaidViewer code={code} onExpand={() => setIsFullscreen(true)} />
      <Dialog open={isFullscreen} onOpenChange={setIsFullscreen}>
        <DialogContent
          size="large"
          className="flex max-h-[95vh] h-[95vh] w-[95vw] flex-col gap-0 overflow-hidden p-0"
        >
          <div className="flex shrink-0 items-center justify-between border-b border-slate-200 bg-white px-4 py-2">
            <span className="font-mono text-xs text-slate-600">Mermaid diagram</span>
          </div>
          <div className="flex min-h-0 flex-1 flex-col bg-white">
            <MermaidViewer code={code} fullscreen />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

const MIN_SCALE = 0.2;
const MAX_SCALE = 8;
const ZOOM_STEP = 1.2;

interface MermaidViewerProps {
  code: string;
  onExpand?: () => void;
  fullscreen?: boolean;
}

function MermaidViewer({ code, onExpand, fullscreen = false }: MermaidViewerProps) {
  const reactId = useId();
  const diagramId = `mermaid-${reactId.replace(/[^a-zA-Z0-9]/g, "")}${fullscreen ? "-fs" : ""}`;
  const hostRef = useRef<HTMLDivElement | null>(null);
  const surfaceRef = useRef<HTMLDivElement | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isRendered, setIsRendered] = useState(false);
  const [transform, setTransform] = useState({ scale: 1, tx: 0, ty: 0 });
  const [isPanning, setIsPanning] = useState(false);
  const panStartRef = useRef({ x: 0, y: 0, tx: 0, ty: 0 });

  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        const mermaid = await getMermaid();
        const { svg, bindFunctions } = await mermaid.render(diagramId, code);

        if (cancelled || !hostRef.current) {
          return;
        }

        hostRef.current.innerHTML = svg;
        if (bindFunctions) {
          bindFunctions(hostRef.current);
        }

        // Mermaid bakes an inline `max-width: <natural-width>px` onto the SVG
        // which wins over any CSS rule and leaves small diagrams at their
        // tiny intrinsic size. Override it so the SVG fills the surface and
        // the transform (scale/translate) has something substantial to work
        // with.
        const svgEl = hostRef.current.querySelector("svg");
        if (svgEl) {
          svgEl.style.maxWidth = "100%";
          svgEl.style.width = "100%";
          svgEl.style.height = "auto";
          svgEl.style.display = "block";
        }

        setError(null);
        setIsRendered(true);
      } catch (err) {
        if (cancelled) return;
        const message = err instanceof Error ? err.message : String(err);
        setError(message);
        setIsRendered(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [code, diagramId]);

  const reset = useCallback(() => setTransform({ scale: 1, tx: 0, ty: 0 }), []);

  const zoomBy = useCallback((factor: number, centerX?: number, centerY?: number) => {
    setTransform((prev) => {
      const newScale = Math.max(MIN_SCALE, Math.min(MAX_SCALE, prev.scale * factor));
      if (newScale === prev.scale) return prev;
      if (centerX == null || centerY == null) {
        return { ...prev, scale: newScale };
      }
      const ratio = newScale / prev.scale;
      return {
        scale: newScale,
        tx: centerX - (centerX - prev.tx) * ratio,
        ty: centerY - (centerY - prev.ty) * ratio,
      };
    });
  }, []);

  // Toolbar +/- should zoom from the middle of the surface so the view stays
  // centered instead of drifting off into the bottom-right corner.
  const zoomFromCenter = useCallback(
    (factor: number) => {
      const rect = surfaceRef.current?.getBoundingClientRect();
      if (rect) {
        zoomBy(factor, rect.width / 2, rect.height / 2);
      } else {
        zoomBy(factor);
      }
    },
    [zoomBy],
  );

  const onWheel = (e: React.WheelEvent) => {
    // Inline: require modifier so readers can still scroll the page.
    // Fullscreen: bare wheel zooms.
    if (!fullscreen && !e.ctrlKey && !e.metaKey) return;
    e.preventDefault();
    const rect = e.currentTarget.getBoundingClientRect();
    const cx = e.clientX - rect.left;
    const cy = e.clientY - rect.top;
    const factor = e.deltaY > 0 ? 1 / ZOOM_STEP : ZOOM_STEP;
    zoomBy(factor, cx, cy);
  };

  const onMouseDown = (e: React.MouseEvent) => {
    if (e.button !== 0) return;
    setIsPanning(true);
    panStartRef.current = {
      x: e.clientX,
      y: e.clientY,
      tx: transform.tx,
      ty: transform.ty,
    };
  };

  useEffect(() => {
    if (!isPanning) return;

    const onMove = (e: MouseEvent) => {
      const dx = e.clientX - panStartRef.current.x;
      const dy = e.clientY - panStartRef.current.y;
      setTransform((prev) => ({
        ...prev,
        tx: panStartRef.current.tx + dx,
        ty: panStartRef.current.ty + dy,
      }));
    };
    const onUp = () => setIsPanning(false);

    window.addEventListener("mousemove", onMove);
    window.addEventListener("mouseup", onUp);
    return () => {
      window.removeEventListener("mousemove", onMove);
      window.removeEventListener("mouseup", onUp);
    };
  }, [isPanning]);

  if (error) {
    return (
      <div className="my-2 rounded-md border border-red-200 bg-red-50 p-3 text-xs text-red-700">
        <div className="mb-1 flex items-center gap-1.5 font-semibold">
          <AlertTriangle className="h-3.5 w-3.5" />
          Mermaid diagram failed to render
        </div>
        <pre className="whitespace-pre-wrap break-words text-[11px] text-red-600">{error}</pre>
        <details className="mt-2">
          <summary className="cursor-pointer text-[11px] text-red-500">Show source</summary>
          <pre className="mt-1 overflow-x-auto rounded bg-white p-2 text-[11px] text-gray-700">{code}</pre>
        </details>
      </div>
    );
  }

  const wrapperClass = fullscreen
    ? "group relative flex min-h-0 flex-1 flex-col bg-white"
    : "group relative my-2 overflow-hidden rounded-md border border-slate-200 bg-white";

  // Surface gives the host a concrete width to fill, so Mermaid's SVG
  // (which ships with `width="100%"`) stretches out to the available space
  // instead of sitting at its tiny intrinsic size.
  const surfaceClass = fullscreen
    ? "relative flex h-full min-h-0 flex-1 items-center justify-center overflow-hidden"
    : "relative flex max-h-[70vh] items-center justify-center overflow-hidden";

  const cursor = isPanning ? "cursor-grabbing" : "cursor-grab";

  return (
    <div className={wrapperClass}>
      <div
        ref={surfaceRef}
        className={`${surfaceClass} ${cursor}`}
        onWheel={onWheel}
        onMouseDown={onMouseDown}
        onDoubleClick={reset}
        style={{ userSelect: "none", touchAction: "none" }}
      >
        <div
          ref={hostRef}
          data-testid="mermaid-diagram"
          className="w-full px-6 py-5 [&>svg]:block [&>svg]:w-full [&>svg]:max-w-full [&>svg]:h-auto [&>svg]:mx-auto"
          style={{
            transform: `translate(${transform.tx}px, ${transform.ty}px) scale(${transform.scale})`,
            transformOrigin: "0 0",
            transition: isPanning ? "none" : "transform 120ms ease-out",
            willChange: "transform",
          }}
          aria-busy={!isRendered}
        />
      </div>

      <div
        className={`absolute right-2 top-2 flex items-center gap-0.5 rounded-md border border-slate-200 bg-white/95 p-0.5 shadow-sm backdrop-blur transition-opacity ${
          fullscreen ? "opacity-100" : "opacity-0 group-hover:opacity-100 focus-within:opacity-100"
        }`}
        onMouseDown={(e) => e.stopPropagation()}
        onDoubleClick={(e) => e.stopPropagation()}
      >
        <ToolbarButton onClick={() => zoomFromCenter(ZOOM_STEP)} label="Zoom in">
          <Plus className="h-3.5 w-3.5" />
        </ToolbarButton>
        <ToolbarButton onClick={() => zoomFromCenter(1 / ZOOM_STEP)} label="Zoom out">
          <Minus className="h-3.5 w-3.5" />
        </ToolbarButton>
        <span className="px-1 text-[10px] tabular-nums text-slate-500" aria-live="polite">
          {Math.round(transform.scale * 100)}%
        </span>
        <ToolbarButton onClick={reset} label="Reset view">
          <RotateCcw className="h-3.5 w-3.5" />
        </ToolbarButton>
        {onExpand && (
          <ToolbarButton onClick={onExpand} label="Expand to full screen">
            <Maximize2 className="h-3.5 w-3.5" />
          </ToolbarButton>
        )}
      </div>

      <div
        className={`pointer-events-none absolute bottom-1.5 left-2 text-[10px] text-slate-400 transition-opacity ${
          fullscreen ? "opacity-70" : "opacity-0 group-hover:opacity-80"
        }`}
      >
        {fullscreen
          ? "scroll to zoom · drag to pan · double-click to reset"
          : "⌘/Ctrl + scroll to zoom · drag to pan · double-click to reset"}
      </div>
    </div>
  );
}

function ToolbarButton({
  children,
  onClick,
  label,
}: {
  children: React.ReactNode;
  onClick: () => void;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      title={label}
      className="inline-flex h-6 w-6 items-center justify-center rounded text-slate-600 hover:bg-slate-100 focus:outline-none focus:ring-1 focus:ring-slate-300"
    >
      {children}
    </button>
  );
}
