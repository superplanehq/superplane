import { useLayoutEffect, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
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
}

export function WorkOrderDescription({ description, className }: WorkOrderDescriptionProps) {
  const contentRef = useRef<HTMLDivElement | null>(null);
  const [isExpanded, setIsExpanded] = useState(false);
  const [needsToggle, setNeedsToggle] = useState(false);
  const [collapsedMaxHeight, setCollapsedMaxHeight] = useState(FALLBACK_COLLAPSED_MAX_HEIGHT_PX);

  useLayoutEffect(() => {
    const content = contentRef.current;
    if (!content) {
      return;
    }

    const updateOverflow = () => {
      const pane = nearestScrollParent(content);
      const capacity = descriptionPaneCapacity(readScrollPaneMetrics(content));
      const leftover = descriptionLeftoverCapacity(capacity, pane ? reservedPaneSiblingHeight(content, pane) : 0);
      setNeedsToggle(descriptionNeedsCollapse(content.scrollHeight, leftover));
      setCollapsedMaxHeight(collapsedDescriptionMaxHeight(leftover));
    };
    updateOverflow();

    const observer = new ResizeObserver(updateOverflow);
    observer.observe(content);
    const pane = nearestScrollParent(content);
    if (pane) {
      observer.observe(pane);
      for (const child of Array.from(pane.children)) {
        observer.observe(child);
      }
    }
    return () => observer.disconnect();
  }, [description]);

  if (!description.trim()) {
    return null;
  }

  const showFade = needsToggle && !isExpanded;

  return (
    <section className={className} data-testid="work-order-description">
      <div className="relative">
        <div
          ref={contentRef}
          style={!isExpanded && needsToggle ? { maxHeight: `${collapsedMaxHeight}px`, overflow: "hidden" } : undefined}
        >
          <MarkdownContent content={description} variant="workspace" data-testid="work-order-description-markdown" />
        </div>
        {showFade ? (
          <div
            aria-hidden
            className="pointer-events-none absolute inset-x-0 bottom-0 h-14 bg-gradient-to-t from-background via-background/90 to-transparent"
          />
        ) : null}
      </div>

      {needsToggle ? (
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
