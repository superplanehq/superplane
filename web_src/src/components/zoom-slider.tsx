"use client";

import React, { useEffect } from "react";
import { Maximize, Minus, MousePointer2, Plus } from "lucide-react";

import { Panel, useViewport, useStore, useReactFlow, type PanelProps } from "@xyflow/react";

import { Slider } from "@/components/ui/slider";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

export function ZoomSlider({
  className,
  orientation = "horizontal",
  children,
  isSelectionModeEnabled,
  onSelectionModeToggle,
  ...props
}: Omit<PanelProps, "children"> & {
  orientation?: "horizontal" | "vertical";
  children?: React.ReactNode;
  isSelectionModeEnabled?: boolean;
  onSelectionModeToggle?: () => void;
}) {
  const { zoom } = useViewport();
  const { zoomTo, zoomIn, zoomOut, fitView } = useReactFlow();
  const minZoom = useStore((state) => state.minZoom);
  const maxZoom = useStore((state) => state.maxZoom);

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
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [zoomIn, zoomOut, zoomTo, fitView]);

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
          <TooltipContent>Fit all nodes in view (Ctrl/Cmd + 1)</TooltipContent>
        </Tooltip>
        {onSelectionModeToggle && (
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant={isSelectionModeEnabled ? "default" : "ghost"}
                size="icon-sm"
                className="h-8 w-8"
                onClick={onSelectionModeToggle}
              >
                <MousePointer2 className="h-3 w-3" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              {isSelectionModeEnabled
                ? "Disable rectangle selection mode"
                : "Select and move multiple components at once (Ctrl/Cmd + drag)"}
            </TooltipContent>
          </Tooltip>
        )}
        {children}
      </Panel>
    </TooltipProvider>
  );
}
