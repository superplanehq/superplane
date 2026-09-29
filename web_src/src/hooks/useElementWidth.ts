import { useEffect, useRef, useState } from "react";

/**
 * Measures an element's width via `ResizeObserver`, seeded with
 * `initialWidth` so the very first render already reflects a usable width
 * instead of `0` — before the observer's first callback fires, and in
 * environments such as jsdom where `ResizeObserver` never fires.
 */
export function useElementWidth<T extends Element>(initialWidth: number) {
  const ref = useRef<T | null>(null);
  const [width, setWidth] = useState(initialWidth);

  useEffect(() => {
    const element = ref.current;
    if (!element) return;

    const update = () => {
      const measured = element.getBoundingClientRect().width;
      if (measured > 0) {
        setWidth(measured);
      }
    };

    update();
    const observer = new ResizeObserver(update);
    observer.observe(element);
    return () => observer.disconnect();
  }, []);

  return { ref, width };
}
