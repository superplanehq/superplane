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
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={() => zoomOut({ duration: 300 })}
          title="Zoom out"
        >
          <Minus className="h-3 w-3" />
        </Button>
        <Slider
          className={cn(
            orientation === "horizontal" ? "w-[100px]" : "h-[100px]",
          )}
          orientation={orientation}
          value={[zoom]}
          min={minZoom}
          max={maxZoom}
          step={0.01}
          onValueChange={(values) => zoomTo(values[0])}
          title="Zoom level"
        />
        <Button
          variant="ghost"
          size="icon-sm"
          onClick={() => zoomIn({ duration: 300 })}
          title="Zoom in"
        >
          <Plus className="h-3 w-3" />
        </Button>
      </div>
      <Button
        className={cn(
          "tabular-nums text-xs",
          orientation === "horizontal"
            ? "w-[50px] min-w-[50px] h-8"
            : "h-[40px] w-[40px]",
        )}
        variant="ghost"
        onClick={() => zoomTo(1, { duration: 300 })}
        title="Reset zoom to 100%"
      >
        {(100 * zoom).toFixed(0)}%
      </Button>
      <Button
        variant="ghost"
        size="icon-sm"
        onClick={() => fitView({ duration: 300 })}
        title="Fit all nodes in view"
      >
        <Maximize className="h-3 w-3" />
      </Button>
      {children}
    </Panel>
  );
}
