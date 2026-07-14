import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type MutableRefObject,
  type RefObject,
} from "react";
import { computeFitScale, normalizeSvgForViewport, readSvgSize } from "./mermaidSvgSize";

const FIT_PADDING = 48;

export function useMermaidFitToViewport({
  viewportRef,
  contentRef,
  svg,
  onScaleChange,
  onFitScaleChange,
  fitToViewportRef,
  scaleRef,
  fitScaleRef,
}: {
  viewportRef: RefObject<HTMLDivElement | null>;
  contentRef: RefObject<HTMLDivElement | null>;
  svg: string;
  onScaleChange: (scale: number) => void;
  onFitScaleChange: (scale: number) => void;
  fitToViewportRef: MutableRefObject<(() => void) | null>;
  scaleRef: MutableRefObject<number>;
  fitScaleRef: MutableRefObject<number>;
}) {
  // Keep callbacks in refs so fitToViewport stays stable across parent re-renders
  // (e.g. zoom/pan). Otherwise useLayoutEffect would re-fit and wipe user scale.
  const onScaleChangeRef = useRef(onScaleChange);
  const onFitScaleChangeRef = useRef(onFitScaleChange);
  onScaleChangeRef.current = onScaleChange;
  onFitScaleChangeRef.current = onFitScaleChange;

  const fitToViewport = useCallback(() => {
    const viewport = viewportRef.current;
    const svgEl = contentRef.current?.querySelector("svg");
    if (!viewport || !svgEl) {
      return;
    }

    // Mermaid emits `max-width: 100%` which shrinks the SVG to its parent
    // before we apply transform scale — clear that so fit math uses the
    // diagram's intrinsic size and can fill the dialog.
    normalizeSvgForViewport(svgEl);

    const viewportWidth = Math.max(viewport.clientWidth - FIT_PADDING, 1);
    const viewportHeight = Math.max(viewport.clientHeight - FIT_PADDING, 1);
    const bounds = readSvgSize(svgEl);
    const nextFit = computeFitScale(viewportWidth, viewportHeight, bounds.width, bounds.height);
    fitScaleRef.current = nextFit;
    onFitScaleChangeRef.current(nextFit);
    onScaleChangeRef.current(nextFit);
  }, [contentRef, fitScaleRef, viewportRef]);

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
  }, [fitScaleRef, fitToViewport, scaleRef, viewportRef]);

  return fitToViewport;
}

export function useMermaidPan() {
  const [translate, setTranslate] = useState({ x: 0, y: 0 });
  const dragRef = useRef<{ startX: number; startY: number; startTx: number; startTy: number } | null>(null);

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

  const resetPan = useCallback(() => setTranslate({ x: 0, y: 0 }), []);

  return { translate, resetPan, handleMouseDown, handleMouseMove, handleMouseUp };
}
