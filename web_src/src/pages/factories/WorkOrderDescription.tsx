import { useLayoutEffect, useRef, useState } from "react";

import type { FilesFile } from "@/api-client";
import { Button } from "@/components/ui/button";
import { rewriteWorkOrderFileRefs } from "@/lib/workOrderFiles";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { ChevronDown } from "lucide-react";

import {
  FALLBACK_COLLAPSED_MAX_HEIGHT_PX,
  collapsedDescriptionMaxHeight,
  descriptionLeftoverCapacity,
  descriptionNeedsCollapse,
  descriptionPaneCapacity,
  nearestScrollParent,
  readScrollPaneMetrics,
  reservedPaneSiblingHeight,
} from "./workOrderDescriptionOverflow";

interface WorkOrderDescriptionProps {
  description: string;
  className?: string;
  /** When false, always show the full markdown. Default is true. */
  collapsible?: boolean;
  files?: FilesFile[];
  /** When set, collapse at this height instead of filling the leftover scroll pane. */
  previewHeight?: number;
  fadeClassName?: string;
}

export function WorkOrderDescription({
  description,
  className,
  collapsible = true,
  files,
  previewHeight,
  fadeClassName = "from-background via-background/90",
}: WorkOrderDescriptionProps) {
  const contentRef = useRef<HTMLDivElement | null>(null);
  const [isExpanded, setIsExpanded] = useState(false);
  const [needsToggle, setNeedsToggle] = useState(false);
  const [collapsedMaxHeight, setCollapsedMaxHeight] = useState(FALLBACK_COLLAPSED_MAX_HEIGHT_PX);

  useLayoutEffect(() => {
    if (!collapsible) {
      return;
    }
    const content = contentRef.current;
    if (!content) {
      return;
    }

    const updateOverflow = () => {
      if (previewHeight != null) {
        setNeedsToggle(descriptionNeedsCollapse(content.scrollHeight, previewHeight));
        setCollapsedMaxHeight(previewHeight);
        return;
      }

      const pane = nearestScrollParent(content);
      const capacity = descriptionPaneCapacity(readScrollPaneMetrics(content));
      const leftover = descriptionLeftoverCapacity(capacity, pane ? reservedPaneSiblingHeight(content, pane) : 0);
      setNeedsToggle(descriptionNeedsCollapse(content.scrollHeight, leftover));
      setCollapsedMaxHeight(collapsedDescriptionMaxHeight(leftover));
    };
    updateOverflow();

    const observer = new ResizeObserver(updateOverflow);
    observer.observe(content);
    if (previewHeight != null) {
      return () => observer.disconnect();
    }

    const pane = nearestScrollParent(content);
    if (pane) {
      observer.observe(pane);
      for (const child of Array.from(pane.children)) {
        observer.observe(child);
      }
    }
    return () => observer.disconnect();
  }, [description, files, collapsible, previewHeight]);

  const rendered = rewriteWorkOrderFileRefs(description, files);

  if (!rendered.trim()) {
    return null;
  }

  const showFade = collapsible && needsToggle && !isExpanded;
  const clamp = collapsible && !isExpanded && needsToggle;

  return (
    <section className={className} data-testid="work-order-description">
      <div className="relative">
        <div ref={contentRef} style={clamp ? { maxHeight: `${collapsedMaxHeight}px`, overflow: "hidden" } : undefined}>
          <MarkdownContent content={rendered} variant="workspace" data-testid="work-order-description-markdown" />
        </div>
        {showFade ? (
          <div
            aria-hidden
            className={cn(
              "pointer-events-none absolute inset-x-0 bottom-0 h-14 bg-gradient-to-t to-transparent",
              fadeClassName,
            )}
          />
        ) : null}
      </div>

      {collapsible && needsToggle ? (
        <Button
          type="button"
          variant="ghost"
          className="mt-2 h-auto gap-1 p-0 text-[13px] font-medium text-foreground/80 hover:bg-transparent hover:text-foreground"
          onClick={() => setIsExpanded((prev) => !prev)}
          data-testid="work-order-description-toggle"
        >
          {isExpanded ? "Show less" : "Show more"}
          <ChevronDown className={cn("size-3 transition-transform", isExpanded && "rotate-180")} aria-hidden />
        </Button>
      ) : null}
    </section>
  );
}
