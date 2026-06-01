import { cn } from "@/lib/utils";
import { useTruncatedText } from "@/hooks/useTruncatedText";
import { useState, type MouseEvent, type ReactNode } from "react";

const descriptionClassName = "text-left text-sm leading-normal text-gray-800 dark:text-gray-400";

export function CanvasCardDescription({ description }: { description: string }) {
  const [isExpanded, setIsExpanded] = useState(false);
  const { containerRef, isOverflowing, measureClassName, measureRef, truncateAt } = useTruncatedText(description, {
    isExpanded,
    measureClassName: descriptionClassName,
  });

  const handleToggle = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault();
    event.stopPropagation();
    setIsExpanded((current) => !current);
  };

  return (
    <div ref={containerRef} className="pointer-events-auto relative mt-1 mb-3">
      <p ref={measureRef} aria-hidden className={cn("invisible absolute w-full", measureClassName)} />
      <p className={descriptionClassName}>
        {isExpanded ? (
          isOverflowing ? (
            <>
              {`${description} `}
              <DescriptionToggle onClick={handleToggle}>/ show less</DescriptionToggle>
            </>
          ) : (
            description
          )
        ) : isOverflowing ? (
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
