"use client";

import React from "react";
import { Maximize, Minus, Plus } from "lucide-react";

import {
  Panel,
  useViewport,
  useStore,
  useReactFlow,
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
  ...props
}: Omit<PanelProps, "children"> & {
  orientation?: "horizontal" | "vertical";
  children?: React.ReactNode;
}) {
  const { zoom } = useViewport();
  const { zoomTo, zoomIn, zoomOut, fitView } = useReactFlow();
  const minZoom = useStore((state) => state.minZoom);
  const maxZoom = useStore((state) => state.maxZoom);

  return (
    <TooltipProvider delayDuration={300}>
      <Panel
        className={cn(
          "bg-primary-foreground text-foreground flex gap-0.5 rounded-md p-0.5",
          orientation === "horizontal" ? "flex-row" : "flex-col",
          className,
        )}
        {...props}
      >
        <div
          className={cn(
            "flex gap-0.5",
            orientation === "horizontal" ? "flex-row" : "flex-col-reverse",
          )}
        >
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => zoomOut({ duration: 300 })}
              >
                <Minus className="h-3 w-3" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Zoom out</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <div className={cn(orientation === "horizontal" ? "w-[100px]" : "h-[100px]")}>
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
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => zoomIn({ duration: 300 })}
              >
                <Plus className="h-3 w-3" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Zoom in</TooltipContent>
          </Tooltip>
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              className={cn(
                "tabular-nums text-xs",
                orientation === "horizontal"
                  ? "w-[50px] min-w-[50px] h-8"
                  : "h-[40px] w-[40px]",
              )}
              variant="ghost"
              onClick={() => zoomTo(1, { duration: 300 })}
            >
              {(100 * zoom).toFixed(0)}%
            </Button>
          </TooltipTrigger>
          <TooltipContent>Reset zoom to 100%</TooltipContent>
        </Tooltip>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => fitView({ duration: 300 })}
            >
              <Maximize className="h-3 w-3" />
            </Button>
          </TooltipTrigger>
          <TooltipContent>Fit all nodes in view</TooltipContent>
        </Tooltip>
        {children}
      </Panel>
    </TooltipProvider>
  );
}
