"use client";

import React, { useCallback, useEffect } from "react";
import { Camera, Grid3X3, Maximize, Minus, Plus, SquareDot } from "lucide-react";
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

export function ZoomSlider({
  className,
  orientation = "horizontal",
  children,
  screenshotName,
  isSnapToGridEnabled,
  onSnapToGridToggle,
  ...props
}: Omit<PanelProps, "children"> & {
  orientation?: "horizontal" | "vertical";
  children?: React.ReactNode;
  screenshotName?: string;
  isSnapToGridEnabled?: boolean;
  onSnapToGridToggle?: () => void;
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

  // Add keyboard shortcuts for zoom controls
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Zoom in: Ctrl/Cmd + = or Ctrl/Cmd + Plus
      if ((e.ctrlKey || e.metaKey) && (e.key === "=" || e.key === "+")) {
        e.preventDefault();
        zoomIn({ duration: 300 });
      }
      // Zoom out: Ctrl/Cmd + - or Ctrl/Cmd + Minus
      else if ((e.ctrlKey || e.metaKey) && e.key === "-") {
        e.preventDefault();
        zoomOut({ duration: 300 });
      }
      // Reset zoom: Ctrl/Cmd + 0
      else if ((e.ctrlKey || e.metaKey) && e.key === "0") {
        e.preventDefault();
        zoomTo(1, { duration: 300 });
      }
      // Fit view: Ctrl/Cmd + 1
      else if ((e.ctrlKey || e.metaKey) && !e.shiftKey && e.key === "1") {
        e.preventDefault();
        fitView({ duration: 300 });
      }
      // Screenshot: Ctrl/Cmd + Shift + S
      else if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "s") {
        e.preventDefault();
        handleScreenshot();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [zoomIn, zoomOut, zoomTo, fitView, handleScreenshot]);

  return (
    <TooltipProvider delayDuration={300}>
      <Panel
        className={cn(
          "bg-white text-gray-800 outline-1 outline-slate-950/20 flex items-center gap-1 rounded-md p-0.5 h-8",
          orientation === "horizontal" ? "flex-row" : "flex-col",
          className,
        )}
        {...props}
      >
        <div className={cn("flex items-center gap-1", orientation === "horizontal" ? "flex-row" : "flex-col-reverse")}>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-sm" className="h-8 w-8" onClick={() => zoomOut({ duration: 300 })}>
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
              <Button variant="ghost" size="icon-sm" className="h-8 w-8" onClick={() => zoomIn({ duration: 300 })}>
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
                orientation === "horizontal" ? "w-[50px] min-w-[50px] h-8" : "h-[40px] w-[40px]",
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
            <Button variant="ghost" size="icon-sm" className="h-8 w-8" onClick={() => fitView({ duration: 300 })}>
              <Maximize className="h-3 w-3" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Fit all components in view (Ctrl/Cmd + 1)</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button variant="ghost" size="icon-sm" className="h-8 w-8" onClick={handleScreenshot}>
              <Camera className="h-3 w-3" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Download screenshot (Ctrl/Cmd + Shift + S)</TooltipContent>
        </Tooltip>
        {onSnapToGridToggle && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon-sm" className="h-8 w-8" onClick={onSnapToGridToggle}>
                {isSnapToGridEnabled ? <SquareDot className="h-3 w-3" /> : <Grid3X3 className="h-3 w-3" />}
              </Button>
            </TooltipTrigger>
            <TooltipContent>{isSnapToGridEnabled ? "Disable snap to grid" : "Enable snap to grid"}</TooltipContent>
          </Tooltip>
        )}
        {children}
      </Panel>
    </TooltipProvider>
  );
}
