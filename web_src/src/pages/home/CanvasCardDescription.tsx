import { cn } from "@/lib/utils";
import { useCallback, useEffect, useLayoutEffect, useRef, useState, type MouseEvent, type ReactNode } from "react";

const descriptionClassName = "text-left text-sm leading-normal text-gray-800 dark:text-gray-400";

export function CanvasCardDescription({ description }: { description: string }) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [isOverflowing, setIsOverflowing] = useState(false);
  const [truncateAt, setTruncateAt] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const measureRef = useRef<HTMLParagraphElement>(null);

  const remeasure = useCallback(() => {
    const element = measureRef.current;
    if (!element) {
      return;
    }

    const lineHeight = Number.parseFloat(getComputedStyle(element).lineHeight);
    const maxHeight = lineHeight * 2;

    const fitsContent = (length: number) => {
      element.replaceChildren();

      const visibleText = length >= description.length ? description : `${description.slice(0, length).trimEnd()} `;
      element.append(document.createTextNode(visibleText));

      if (length < description.length) {
        const toggle = document.createElement("span");
        toggle.textContent = "…more";
        element.append(toggle);
      }

      return element.scrollHeight <= maxHeight + 1;
    };

    if (fitsContent(description.length)) {
      setIsOverflowing(false);
      return;
    }

    setIsOverflowing(true);

    if (isExpanded) {
      return;
    }

    let low = 0;
    let high = description.length;
    let best = 0;

    while (low <= high) {
      const mid = Math.floor((low + high) / 2);
      if (fitsContent(mid)) {
        best = mid;
        low = mid + 1;
      } else {
        high = mid - 1;
      }
    }

    setTruncateAt(best);
  }, [description, isExpanded]);

  useLayoutEffect(() => {
    remeasure();
  }, [remeasure]);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) {
      return;
    }

    const observer = new ResizeObserver(remeasure);
    observer.observe(element);

    return () => observer.disconnect();
  }, [remeasure]);

  const handleToggle = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsExpanded((current) => !current);
  };

  if (isExpanded) {
    return (
      <div ref={containerRef} className="pointer-events-auto mt-1 mb-3">
        <p ref={measureRef} aria-hidden className={cn("invisible absolute w-full", descriptionClassName)} />
        <p className={descriptionClassName}>
          {isOverflowing ? (
            <>
              {`${description} `}
              <DescriptionToggle onClick={handleToggle}>/ show less</DescriptionToggle>
            </>
          ) : (
            description
          )}
        </p>
      </div>
    );
  }

  return (
    <div ref={containerRef} className="pointer-events-auto relative mt-1 mb-3">
      <p ref={measureRef} aria-hidden className={cn("invisible absolute w-full", descriptionClassName)} />
      <p className={descriptionClassName}>
        {isOverflowing ? (
          <>
            {`${description.slice(0, truncateAt).trimEnd()} `}
            <DescriptionToggle onClick={handleToggle}>…more</DescriptionToggle>
          </>
        ) : (
          description
        )}
      </p>
    </div>
  );
}

function DescriptionToggle({
  children,
  onClick,
}: {
  children: ReactNode;
  onClick: (event: MouseEvent<HTMLButtonElement>) => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline text-gray-500 hover:text-gray-700 dark:hover:text-gray-400"
    >
      {children}
    </button>
  );
}
