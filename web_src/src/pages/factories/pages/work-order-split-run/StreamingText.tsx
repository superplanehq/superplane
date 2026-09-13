import { memo, useEffect, useLayoutEffect, useRef, useState, type HTMLAttributes, type ReactNode } from "react";

import { prefersReducedMotion, streamGapMs, streamUnits, visibleGeneratedMarkdown } from "@/lib/streamWords";
import { cn } from "@/lib/utils";

const streamMemory = new Map<string, string>();

type StreamingTextProps = Omit<HTMLAttributes<HTMLDivElement>, "children"> & {
  content: string;
  contentKey?: string;
  memoryKey?: string;
  ready?: boolean;
  children: (visible: string) => ReactNode;
};

/** Test helper. Strict Mode and popup remounts share this memory. */
export function resetStreamMemoryForTests(): void {
  streamMemory.clear();
}

/**
 * Play after this pane has already seen a value. The previous value is stored
 * in an effect so React Strict Mode cannot flip play back to false.
 */
export function useStreamOnUpdate(value: string, memoryKey?: string, ready = true): boolean {
  const seen = useRef<string | undefined>(undefined);
  if (ready && memoryKey && seen.current === undefined && streamMemory.has(memoryKey)) {
    seen.current = streamMemory.get(memoryKey);
  }
  const play = ready && seen.current !== undefined && seen.current !== value;
  useLayoutEffect(() => {
    if (!ready) {
      return;
    }
    seen.current = value;
    if (memoryKey) {
      streamMemory.set(memoryKey, value);
    }
  }, [memoryKey, ready, value]);
  return play;
}

/**
 * Write spec markdown as a file being generated. Headings and bullets arrive
 * with their text. Later blocks do not exist yet. Opening a card stays still.
 */
export const StreamingText = memo(function StreamingText({
  content,
  contentKey = content,
  memoryKey,
  ready = true,
  children,
  className,
  ...rest
}: StreamingTextProps) {
  const shouldStart = useStreamOnUpdate(contentKey, memoryKey, ready) && !prefersReducedMotion();
  const [pass, setPass] = useState(0);
  const [revealed, setRevealed] = useState(Number.POSITIVE_INFINITY);
  const streaming = Number.isFinite(revealed);
  const visible = streaming ? visibleGeneratedMarkdown(content, revealed) : content;

  useLayoutEffect(() => {
    if (!shouldStart) {
      return;
    }
    setPass((current) => current + 1);
    setRevealed(1);
  }, [contentKey, shouldStart]);

  useEffect(() => {
    if (pass === 0) {
      return;
    }
    const total = streamUnits(content).length;
    const gap = streamGapMs();
    const timer = window.setInterval(() => {
      setRevealed((current) => {
        const next = (Number.isFinite(current) ? current : 0) + 1;
        if (next >= total) {
          window.clearInterval(timer);
          return Number.POSITIVE_INFINITY;
        }
        return next;
      });
    }, gap);
    return () => window.clearInterval(timer);
  }, [content, contentKey, pass]);

  return (
    <div className={cn(streaming && "sp-streaming", className)} data-streaming={streaming ? "" : undefined} {...rest}>
      {children(visible)}
    </div>
  );
});
