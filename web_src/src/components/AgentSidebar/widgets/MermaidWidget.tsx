import { useCallback, useEffect, useId, useRef, useState } from "react";
import { Minus, Plus, RotateCcw } from "lucide-react";
import mermaid from "mermaid";

import { useTheme } from "@/contexts/useTheme";
import { FullscreenContentDialog } from "@/ui/FullscreenContentDialog";
import { HeaderIconButton } from "@/ui/HeaderIconButton";
import { mermaidInitializeConfig } from "./mermaidTheme";
import { useMermaidFitToViewport, useMermaidPan } from "./useMermaidViewport";

interface MermaidWidgetProps {
  content: string;
}

export function MermaidWidget({ content }: MermaidWidgetProps) {
  const id = useId().replace(/:/g, "m");
  const { resolvedTheme } = useTheme();
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
        mermaid.initialize(mermaidInitializeConfig(resolvedTheme));
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
  }, [content, id, resolvedTheme]);

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
  const scaleRef = useRef(scale);
  const fitScaleRef = useRef(1);
  scaleRef.current = scale;

  const { translate, resetPan, handleMouseDown, handleMouseMove, handleMouseUp } = useMermaidPan();

  const applyFitScale = useCallback(
    (next: number) => {
      resetPan();
      onScaleChange(next);
    },
    [onScaleChange, resetPan],
  );

  useMermaidFitToViewport({
    viewportRef,
    contentRef,
    svg,
    onScaleChange: applyFitScale,
    onFitScaleChange,
    fitToViewportRef,
    scaleRef,
    fitScaleRef,
  });

  const handleWheel = useCallback(
    (e: React.WheelEvent) => {
      e.preventDefault();
      const delta = e.deltaY > 0 ? 0.9 : 1.1;
      onScaleChange(Math.min(Math.max(scaleRef.current * delta, 0.2), 5));
    },
    [onScaleChange],
  );

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
