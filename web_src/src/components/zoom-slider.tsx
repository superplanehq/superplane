"use client";

import React, { memo, useCallback, useEffect } from "react";
import {
  Camera,
  CircleDot,
  CircleDotDashed,
  Eye,
  LayoutDashboard,
  LayoutGrid,
  Locate,
  LocateOff,
  Minus,
  Plus,
} from "lucide-react";
import { toPng } from "html-to-image";

import {
  Panel,
  useViewport,
  useStore,
  useReactFlow,
  getNodesBounds,
  getViewportForBounds,
  type PanelProps,
} from "@xyflow/react";

import { Slider } from "@/components/ui/slider";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { LIVE_CANVAS_FIT_VIEW_OPTIONS } from "@/ui/CanvasPage/canvasFitOptions";

function hasPrimaryModifier(event: KeyboardEvent) {
  return event.ctrlKey || event.metaKey;
}

function isZoomInShortcut(event: KeyboardEvent) {
  return hasPrimaryModifier(event) && (event.key === "=" || event.key === "+");
}

function isZoomOutShortcut(event: KeyboardEvent) {
  return hasPrimaryModifier(event) && event.key === "-";
}

function isResetZoomShortcut(event: KeyboardEvent) {
  return hasPrimaryModifier(event) && event.key === "0";
}

function isFitViewShortcut(event: KeyboardEvent) {
  return hasPrimaryModifier(event) && !event.shiftKey && event.key === "1";
}

function isScreenshotShortcut(event: KeyboardEvent, screenshotName?: string) {
  return Boolean(screenshotName) && hasPrimaryModifier(event) && event.shiftKey && event.key === "s";
}

function AutoFocusToggleButton({ enabled, onToggle }: { enabled: boolean; onToggle: () => void }) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          variant="ghost"
          size="icon-sm"
          className="h-7 w-7"
          onClick={onToggle}
          aria-pressed={enabled}
          aria-label={enabled ? "Disable auto-focus on selection" : "Enable auto-focus on selection"}
          data-testid="canvas-auto-focus-toggle"
        >
          {enabled ? <Locate className="h-3 w-3" /> : <LocateOff className="h-3 w-3" />}
        </Button>
      </TooltipTrigger>
      <TooltipContent>
        {enabled
          ? "Auto-focus is on. Selecting a run or step centers the canvas on it."
          : "Auto-focus is off. Selecting a run or step keeps the current viewport."}
      </TooltipContent>
    </Tooltip>
  );
}

function ScreenshotToolbarButton({
  screenshotName,
  onScreenshot,
}: {
  screenshotName?: string;
  onScreenshot: () => void;
}) {
  if (!screenshotName) {
    return null;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button variant="ghost" size="icon-sm" className="h-7 w-7" onClick={onScreenshot}>
          <Camera className="h-3 w-3" />
        </Button>
      </TooltipTrigger>
      <TooltipContent>Download screenshot (Ctrl/Cmd + Shift + S)</TooltipContent>
    </Tooltip>
  );
}

function SnapToGridToggleButton({ enabled, onToggle }: { enabled?: boolean; onToggle?: () => void }) {
  if (!onToggle) {
    return null;
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button variant="ghost" size="icon-sm" className="h-7 w-7" onClick={onToggle}>
          {enabled ? <CircleDot className="h-3 w-3" /> : <CircleDotDashed className="h-3 w-3" />}
        </Button>
      </TooltipTrigger>
      <TooltipContent>{enabled ? "Disable snap to grid" : "Enable snap to grid"}</TooltipContent>
    </Tooltip>
  );
}

function AutoLayoutOnUpdateToggleButton({
  enabled,
  onToggle,
  disabled,
  disabledTooltip,
}: {
  enabled?: boolean;
  onToggle?: () => void;
  disabled?: boolean;
  disabledTooltip?: string;
}) {
  if (!onToggle) {
    return null;
  }

  const tooltipMessage =
    disabledTooltip ||
    (enabled
      ? "Auto-layout on add is enabled. New nodes reflow their connected graph."
      : "Auto-layout on add is disabled. Click to enable connected-graph layout for newly added nodes.");

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex">
          <Button
            variant="ghost"
            size="icon-sm"
            className="h-7 w-7"
            onClick={onToggle}
            disabled={disabled}
            aria-pressed={enabled}
          >
            {enabled ? <LayoutGrid className="h-3 w-3" /> : <LayoutDashboard className="h-3 w-3" />}
          </Button>
        </span>
      </TooltipTrigger>
      <TooltipContent>{tooltipMessage}</TooltipContent>
    </Tooltip>
  );
}

type ZoomSliderKeyboardActions = {
  zoomIn: (options?: { duration?: number }) => unknown;
  zoomOut: (options?: { duration?: number }) => unknown;
  zoomTo: (zoomLevel: number, options?: { duration?: number }) => unknown;
  fitView: (options?: object) => unknown;
  handleScreenshot: () => void;
};

function handleZoomSliderKeyDown(event: KeyboardEvent, actions: ZoomSliderKeyboardActions, screenshotName?: string) {
  if (isZoomInShortcut(event)) {
    event.preventDefault();
    actions.zoomIn({ duration: 300 });
    return;
  }

  if (isZoomOutShortcut(event)) {
    event.preventDefault();
    actions.zoomOut({ duration: 300 });
    return;
  }

  if (isResetZoomShortcut(event)) {
    event.preventDefault();
    actions.zoomTo(1, { duration: 300 });
    return;
  }

  if (isFitViewShortcut(event)) {
    event.preventDefault();
    actions.fitView({ duration: 300, ...LIVE_CANVAS_FIT_VIEW_OPTIONS });
    return;
  }

  if (isScreenshotShortcut(event, screenshotName)) {
    event.preventDefault();
    actions.handleScreenshot();
  }
}

export const ZoomSlider = memo(function ZoomSlider({
  className,
  orientation = "horizontal",
  children,
  leadingContent,
  screenshotName,
  isSnapToGridEnabled,
  onSnapToGridToggle,
  isAutoLayoutOnUpdateEnabled,
  onAutoLayoutOnUpdateToggle,
  autoLayoutOnUpdateDisabled,
  autoLayoutOnUpdateDisabledTooltip,
  isAutoFocusEnabled,
  onAutoFocusToggle,
  usePanel = true,
  ...props
}: Omit<PanelProps, "children"> & {
  orientation?: "horizontal" | "vertical";
  children?: React.ReactNode;
  leadingContent?: React.ReactNode;
  screenshotName?: string;
  isSnapToGridEnabled?: boolean;
  onSnapToGridToggle?: () => void;
  isAutoLayoutOnUpdateEnabled?: boolean;
  onAutoLayoutOnUpdateToggle?: () => void;
  autoLayoutOnUpdateDisabled?: boolean;
  autoLayoutOnUpdateDisabledTooltip?: string;
  isAutoFocusEnabled?: boolean;
  onAutoFocusToggle?: () => void;
  usePanel?: boolean;
}) {
  const { zoom } = useViewport();
  const { zoomTo, zoomIn, zoomOut, fitView, getNodes } = useReactFlow();
  const minZoom = useStore((state) => state.minZoom);
  const maxZoom = useStore((state) => state.maxZoom);

  const handleScreenshot = useCallback(() => {
    const nodes = getNodes();
    if (nodes.length === 0) return;

    const nodesBounds = getNodesBounds(nodes);
    const padding = 0.25;

    // Calculate dimensions based on content with padding
    const contentWidth = nodesBounds.width * (1 + padding * 2);
    const contentHeight = nodesBounds.height * (1 + padding * 2);

    // Target ~1.5x scale for good resolution, cap at 4096px per side
    const maxDimension = 4096;
    const targetScale = 1.5;

    let imageWidth = Math.round(contentWidth * targetScale);
    let imageHeight = Math.round(contentHeight * targetScale);

    // Scale down if exceeding max dimension while maintaining aspect ratio
    if (imageWidth > maxDimension || imageHeight > maxDimension) {
      const scale = maxDimension / Math.max(imageWidth, imageHeight);
      imageWidth = Math.round(imageWidth * scale);
      imageHeight = Math.round(imageHeight * scale);
    }

    // Ensure minimum dimensions for small canvases
    imageWidth = Math.max(imageWidth, 800);
    imageHeight = Math.max(imageHeight, 600);

    const viewport = getViewportForBounds(nodesBounds, imageWidth, imageHeight, 0.5, 2, padding);
    const viewportElement = document.querySelector(".react-flow__viewport") as HTMLElement;

    if (!viewportElement) return;

    toPng(viewportElement, {
      backgroundColor: "#F1F5F9",
      width: imageWidth,
      height: imageHeight,
      skipFonts: true,
      style: {
        width: String(imageWidth),
        height: String(imageHeight),
        transform: `translate(${viewport.x}px, ${viewport.y}px) scale(${viewport.zoom})`,
      },
    }).then((dataUrl) => {
      const link = document.createElement("a");
      const date = new Date().toISOString().split("T")[0];
      const name = screenshotName || "Workflow";
      link.download = `${name} screenshot ${date}.png`;
      link.href = dataUrl;
      link.click();
    });
  }, [getNodes, screenshotName]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      handleZoomSliderKeyDown(event, { zoomIn, zoomOut, zoomTo, fitView, handleScreenshot }, screenshotName);
    };

    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [zoomIn, zoomOut, zoomTo, fitView, handleScreenshot, screenshotName]);

  const baseClassName = cn(
    "bg-white text-gray-800 outline-1 outline-slate-950/15 flex items-center gap-0.5 rounded-md p-0.5 h-7",
    "dark:bg-gray-800 dark:text-gray-100 dark:outline-gray-600/70 [&_[data-slot=button]]:dark:hover:bg-gray-700",
    orientation === "horizontal" ? "flex-row" : "flex-col",
    className,
  );

  const content = (
    <>
      <div className={cn("flex items-center gap-1", orientation === "horizontal" ? "flex-row" : "flex-col-reverse")}>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon-sm" className="h-7 w-7" onClick={() => zoomOut({ duration: 300 })}>
              <Minus className="h-3 w-3" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Zoom out (Ctrl/Cmd + -)</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className={cn("hidden", orientation === "horizontal" ? "w-[100px]" : "h-[100px]")}>
              <Slider
                className="w-full h-full"
                orientation={orientation}
                value={[zoom]}
                min={minZoom}
                max={maxZoom}
                step={0.01}
                onValueChange={(values) => zoomTo(values[0])}
              />
            </div>
          </TooltipTrigger>
          <TooltipContent>Zoom level</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon-sm" className="h-7 w-7" onClick={() => zoomIn({ duration: 300 })}>
              <Plus className="h-3 w-3" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Zoom in (Ctrl/Cmd + +)</TooltipContent>
        </Tooltip>
      </div>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            className={cn(
              "tabular-nums text-xs",
              orientation === "horizontal" ? "w-[50px] min-w-[50px] h-7" : "h-[40px] w-[40px]",
            )}
            variant="ghost"
            onClick={() => zoomTo(1, { duration: 300 })}
          >
            {(100 * zoom).toFixed(0)}%
          </Button>
        </TooltipTrigger>
        <TooltipContent>Reset zoom to 100% (Ctrl/Cmd + 0)</TooltipContent>
      </Tooltip>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon-sm"
            className="h-7 w-7"
            onClick={() => fitView({ duration: 300, ...LIVE_CANVAS_FIT_VIEW_OPTIONS })}
          >
            <Eye className="h-3 w-3" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Fit all components in view (Ctrl/Cmd + 1)</TooltipContent>
      </Tooltip>
      {leadingContent}
      <ScreenshotToolbarButton screenshotName={screenshotName} onScreenshot={handleScreenshot} />
      <SnapToGridToggleButton enabled={isSnapToGridEnabled} onToggle={onSnapToGridToggle} />
      <AutoLayoutOnUpdateToggleButton
        enabled={isAutoLayoutOnUpdateEnabled}
        onToggle={onAutoLayoutOnUpdateToggle}
        disabled={autoLayoutOnUpdateDisabled}
        disabledTooltip={autoLayoutOnUpdateDisabledTooltip}
      />
      {onAutoFocusToggle && (
        <AutoFocusToggleButton enabled={Boolean(isAutoFocusEnabled)} onToggle={onAutoFocusToggle} />
      )}
      {children}
    </>
  );

  return (
    <TooltipProvider delayDuration={300}>
      {usePanel ? (
        <Panel className={baseClassName} {...props}>
          {content}
        </Panel>
      ) : (
        <div className={baseClassName}>{content}</div>
      )}
    </TooltipProvider>
  );
});
