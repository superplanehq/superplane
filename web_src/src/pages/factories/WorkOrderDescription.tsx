import { useLayoutEffect, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import { MarkdownContent } from "@/pages/app/Markdown";
import { cn } from "@/lib/utils";

interface WorkOrderDescriptionProps {
  description: string;
  className?: string;
}

const COLLAPSED_MAX_HEIGHT_PX = 220;

/**
 * Render the work order description as markdown with a collapsible fade.
 * Uses a ref to measure the rendered markdown so we only show the "Show more"
 * toggle when content actually exceeds the collapsed height.
 */
export function WorkOrderDescription({ description, className }: WorkOrderDescriptionProps) {
  const contentRef = useRef<HTMLDivElement | null>(null);
  const [isExpanded, setIsExpanded] = useState(false);
  const [needsToggle, setNeedsToggle] = useState(false);

  useLayoutEffect(() => {
    const el = contentRef.current;
    if (!el) return;
    setNeedsToggle(el.scrollHeight > COLLAPSED_MAX_HEIGHT_PX + 4);
  }, [description]);

  if (!description.trim()) {
    return null;
  }

  const showFade = needsToggle && !isExpanded;

  return (
    <section
      className={cn(
        "rounded-lg border border-gray-200 bg-white px-5 py-4 dark:border-gray-700/70 dark:bg-transparent",
        className,
      )}
      data-testid="work-order-description"
    >
      <div className="relative">
        <div
          ref={contentRef}
          style={
            !isExpanded && needsToggle ? { maxHeight: `${COLLAPSED_MAX_HEIGHT_PX}px`, overflow: "hidden" } : undefined
          }
        >
          <MarkdownContent content={description} data-testid="work-order-description-markdown" />
        </div>
        {showFade ? (
          <div
            aria-hidden
            className="pointer-events-none absolute inset-x-0 bottom-0 h-16 bg-gradient-to-t from-white to-transparent dark:from-gray-900"
          />
        ) : null}
      </div>

      {needsToggle ? (
        <div className="mt-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-xs text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-gray-100"
            onClick={() => setIsExpanded((prev) => !prev)}
            data-testid="work-order-description-toggle"
          >
            {isExpanded ? "Show less" : "Show more"}
          </Button>
        </div>
      ) : null}
    </section>
  );
}
