import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";

type UseTruncatedTextOptions = {
  isExpanded: boolean;
  maxLines?: number;
  measureClassName: string;
  moreLabel?: string;
};

export function useTruncatedText(
  text: string,
  { isExpanded, maxLines = 2, measureClassName, moreLabel = "…more" }: UseTruncatedTextOptions,
) {
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
    const maxHeight = lineHeight * maxLines;

    const fitsContent = (length: number) => {
      element.replaceChildren();

      const visibleText = length >= text.length ? text : `${text.slice(0, length).trimEnd()} `;
      element.append(document.createTextNode(visibleText));

      if (length < text.length) {
        const toggle = document.createElement("span");
        toggle.textContent = moreLabel;
        element.append(toggle);
      }

      return element.scrollHeight <= maxHeight + 1;
    };

    if (fitsContent(text.length)) {
      setIsOverflowing(false);
      return;
    }

    setIsOverflowing(true);

    if (isExpanded) {
      return;
    }

    let low = 0;
    let high = text.length;
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
  }, [isExpanded, maxLines, moreLabel, text]);

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

  return {
    containerRef,
    isOverflowing,
    measureClassName,
    measureRef,
    truncateAt,
  };
}
