import { useCallback, useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { Minus, Plus, RotateCcw } from "lucide-react";
import mermaid from "mermaid";

import { FullscreenContentDialog } from "@/ui/FullscreenContentDialog";
import { HeaderIconButton } from "@/ui/HeaderIconButton";

mermaid.initialize({
  startOnLoad: false,
  theme: "base",
  securityLevel: "strict",
  fontFamily: "ui-sans-serif, system-ui, sans-serif",
  themeVariables: {
    // Keep the purple primary; give secondary/tertiary real mid-tone fills
    // instead of near-white pastels that read as washed out.
    primaryColor: "#ddd6fe",
    primaryTextColor: "#4c1d95",
    primaryBorderColor: "#7c3aed",
    secondaryColor: "#67e8f9",
    secondaryTextColor: "#164e63",
    secondaryBorderColor: "#0891b2",
    tertiaryColor: "#fcd34d",
    tertiaryTextColor: "#78350f",
    tertiaryBorderColor: "#d97706",
    lineColor: "#64748b",
    textColor: "#1e293b",
    nodeBorder: "#7c3aed",
    nodeTextColor: "#1e293b",
    clusterBkg: "#f8fafc",
    clusterBorder: "#cbd5e1",
    defaultLinkColor: "#7c3aed",
    fontSize: "13px",
    // Pie slices: purple first, then saturated companions (opacity 1 so they
    // don't look faded on white).
    pie1: "#8b5cf6",
    pie2: "#06b6d4",
    pie3: "#f59e0b",
    pie4: "#10b981",
    pie5: "#f43f5e",
    pie6: "#3b82f6",
    pie7: "#eab308",
    pie8: "#14b8a6",
    pie9: "#ec4899",
    pie10: "#6366f1",
    pie11: "#84cc16",
    pie12: "#f97316",
    pieOpacity: "1",
    pieStrokeColor: "#ffffff",
    pieStrokeWidth: "1px",
    pieOuterStrokeColor: "#e2e8f0",
    pieTitleTextColor: "#1e293b",
    pieSectionTextColor: "#0f172a",
    pieLegendTextColor: "#334155",
  },
});

interface MermaidWidgetProps {
  content: string;
}

export function MermaidWidget({ content }: MermaidWidgetProps) {
  const id = useId().replace(/:/g, "m");
  const [svg, setSvg] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [scale, setScale] = useState(1);
  const [fitScale, setFitScale] = useState(1);
  const fitToViewportRef = useRef<(() => void) | null>(null);

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
      <div className="my-4 rounded-lg border border-red-200 bg-red-50 p-2 dark:border-red-900/60 dark:bg-red-950/40">
        <p className="text-xs font-medium text-red-600 dark:text-red-300">Diagram error</p>
        <pre className="mt-1 whitespace-pre-wrap text-xs text-red-500 dark:text-red-300">{content.trim()}</pre>
      </div>
    );
  }

  if (!svg) {
    return (
      <div className="my-4 flex items-center justify-center py-4 text-xs text-slate-400 dark:text-gray-500">
        Rendering diagram...
      </div>
    );
  }

  return (
    <>
      <button
        type="button"
        onClick={() => setExpanded(true)}
        aria-label="Expand diagram"
        className="my-4 w-full min-w-0 cursor-pointer overflow-x-auto rounded-lg border border-slate-200 bg-white p-3 text-left transition-colors hover:border-slate-300 dark:border-gray-700 dark:bg-gray-800 dark:hover:border-gray-600 [&_svg]:mx-auto [&_svg]:h-auto [&_svg]:max-w-full"
      >
        <div className="pointer-events-none" dangerouslySetInnerHTML={{ __html: svg }} />
      </button>

      <FullscreenContentDialog
        open={expanded}
        onOpenChange={setExpanded}
        title="Diagram"
        size="wide"
        bodyClassName="overflow-hidden p-0"
        headerActions={
          <>
            <HeaderIconButton
              label="Zoom in"
              icon={<Plus className="h-3.5 w-3.5" />}
              onClick={() => setScale((current) => Math.min(current * 1.2, 5))}
            />
            <HeaderIconButton
              label="Zoom out"
              icon={<Minus className="h-3.5 w-3.5" />}
              onClick={() => setScale((current) => Math.max(current * 0.8, 0.2))}
            />
            <HeaderIconButton
              label="Fit to view"
              icon={<RotateCcw className="h-3.5 w-3.5" />}
              onClick={() => fitToViewportRef.current?.()}
            />
            <span className="px-1 text-[11px] tabular-nums text-slate-500 dark:text-gray-400">
              {Math.round(scale * 100)}%{Math.abs(scale - fitScale) < 0.01 ? " · fitted" : ""}
            </span>
          </>
        }
      >
        <MermaidPanZoom
          svg={svg}
          scale={scale}
          onScaleChange={setScale}
          onFitScaleChange={setFitScale}
          fitToViewportRef={fitToViewportRef}
        />
      </FullscreenContentDialog>
    </>
  );
}

function MermaidPanZoom({
  svg,
  scale,
  onScaleChange,
  onFitScaleChange,
  fitToViewportRef,
}: {
  svg: string;
  scale: number;
  onScaleChange: (scale: number) => void;
  onFitScaleChange: (scale: number) => void;
  fitToViewportRef: React.MutableRefObject<(() => void) | null>;
}) {
  const viewportRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const [translate, setTranslate] = useState({ x: 0, y: 0 });
  const dragRef = useRef<{ startX: number; startY: number; startTx: number; startTy: number } | null>(null);
  const scaleRef = useRef(scale);
  const fitScaleRef = useRef(1);
  scaleRef.current = scale;

  const fitToViewport = useCallback(() => {
    const viewport = viewportRef.current;
    const content = contentRef.current;
    const svgEl = content?.querySelector("svg");
    if (!viewport || !svgEl) {
      return;
    }

    // Mermaid emits `max-width: 100%` which shrinks the SVG to its parent
    // before we apply transform scale — clear that so fit math uses the
    // diagram's intrinsic size and can fill the dialog.
    normalizeSvgForViewport(svgEl);

    const padding = 48;
    const viewportWidth = Math.max(viewport.clientWidth - padding, 1);
    const viewportHeight = Math.max(viewport.clientHeight - padding, 1);
    const bounds = readSvgSize(svgEl);
    if (bounds.width <= 0 || bounds.height <= 0) {
      return;
    }

    // Fill the viewport (scale up or down). Cap only to stay within the
    // interactive zoom range — do not keep diagrams artificially small.
    const nextFit = Math.min(viewportWidth / bounds.width, viewportHeight / bounds.height, 5);
    fitScaleRef.current = nextFit;
    onFitScaleChange(nextFit);
    onScaleChange(nextFit);
    setTranslate({ x: 0, y: 0 });
  }, [onFitScaleChange, onScaleChange]);

  useEffect(() => {
    fitToViewportRef.current = fitToViewport;
    return () => {
      fitToViewportRef.current = null;
    };
  }, [fitToViewport, fitToViewportRef]);

  useLayoutEffect(() => {
    fitToViewport();
  }, [fitToViewport, svg]);

  useEffect(() => {
    const viewport = viewportRef.current;
    if (!viewport || typeof ResizeObserver === "undefined") {
      return;
    }

    // Re-fit when the dialog viewport size changes, but only while the user is
    // still at the fitted scale so intentional zoom/pan is preserved.
    let lastWidth = viewport.clientWidth;
    let lastHeight = viewport.clientHeight;
    const observer = new ResizeObserver(() => {
      const width = viewport.clientWidth;
      const height = viewport.clientHeight;
      if (width === lastWidth && height === lastHeight) {
        return;
      }
      lastWidth = width;
      lastHeight = height;
      if (Math.abs(scaleRef.current - fitScaleRef.current) < 0.01) {
        fitToViewport();
      }
    });
    observer.observe(viewport);
    return () => observer.disconnect();
  }, [fitToViewport]);

  const handleWheel = useCallback(
    (e: React.WheelEvent) => {
      e.preventDefault();
      const delta = e.deltaY > 0 ? 0.9 : 1.1;
      onScaleChange(Math.min(Math.max(scaleRef.current * delta, 0.2), 5));
    },
    [onScaleChange],
  );

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

  return (
    <div
      ref={viewportRef}
      className="h-full min-h-0 cursor-grab overflow-hidden bg-slate-50/50 active:cursor-grabbing dark:bg-gray-900"
      onWheel={handleWheel}
      onMouseDown={handleMouseDown}
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
      onMouseLeave={handleMouseUp}
    >
      <div
        className="flex h-full w-full items-center justify-center"
        style={{
          transform: `translate(${translate.x}px, ${translate.y}px) scale(${scale})`,
          transformOrigin: "center center",
        }}
      >
        <div ref={contentRef} className="[&_svg]:h-auto [&_svg]:max-w-none" dangerouslySetInnerHTML={{ __html: svg }} />
      </div>
    </div>
  );
}

function normalizeSvgForViewport(svgEl: SVGSVGElement) {
  svgEl.style.maxWidth = "none";
  svgEl.style.height = "auto";

  const bounds = readSvgSize(svgEl);
  if (bounds.width > 0) {
    svgEl.setAttribute("width", String(bounds.width));
  }
  if (bounds.height > 0) {
    svgEl.setAttribute("height", String(bounds.height));
  }
}

function readSvgSize(svgEl: SVGSVGElement): { width: number; height: number } {
  const viewBox = svgEl.viewBox?.baseVal;
  if (viewBox && viewBox.width > 0 && viewBox.height > 0) {
    return { width: viewBox.width, height: viewBox.height };
  }

  try {
    const bbox = svgEl.getBBox();
    if (bbox.width > 0 && bbox.height > 0) {
      return { width: bbox.width, height: bbox.height };
    }
  } catch {
    // getBBox can throw if the SVG is not rendered yet.
  }

  const widthAttr = svgEl.getAttribute("width");
  const heightAttr = svgEl.getAttribute("height");
  const width = widthAttr && !widthAttr.endsWith("%") ? Number.parseFloat(widthAttr) : NaN;
  const height = heightAttr && !heightAttr.endsWith("%") ? Number.parseFloat(heightAttr) : NaN;
  return {
    width: Number.isFinite(width) && width > 0 ? width : svgEl.clientWidth,
    height: Number.isFinite(height) && height > 0 ? height : svgEl.clientHeight,
  };
}
