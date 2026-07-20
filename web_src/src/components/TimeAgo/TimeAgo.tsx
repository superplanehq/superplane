import React, { useState, useEffect, useRef, useCallback } from "react";
import { formatTimeAgo } from "@/lib/date";

const globalListeners = new Set<() => void>();
let globalIntervalId: ReturnType<typeof setInterval> | null = null;

function startGlobalTimer() {
  if (globalIntervalId) return;
  globalIntervalId = setInterval(() => {
    globalListeners.forEach((cb) => cb());
  }, 1000);
}

function stopGlobalTimer() {
  if (globalIntervalId && globalListeners.size === 0) {
    clearInterval(globalIntervalId);
    globalIntervalId = null;
  }
}

interface TimeAgoProps {
  date: Date | string;
  className?: string;
  includeAgo?: boolean;
}

/**
 * @deprecated Use `Timestamp` from `@/components/Timestamp` so users get the
 * standardized hover details and copy affordance from issue #5150.
 */
export const TimeAgo = React.memo(function TimeAgo({ date, className, includeAgo = true }: TimeAgoProps) {
  const d = typeof date === "string" ? new Date(date) : date;
  const dateMs = d.getTime();
  const [text, setText] = useState(() => formatTimeAgo(new Date(dateMs), includeAgo));
  const lastTextRef = useRef(text);

  const update = useCallback(() => {
    const newText = formatTimeAgo(new Date(dateMs), includeAgo);
    if (newText !== lastTextRef.current) {
      lastTextRef.current = newText;
      setText(newText);
    }
  }, [dateMs, includeAgo]);

  useEffect(() => {
    update();
    globalListeners.add(update);
    startGlobalTimer();
    return () => {
      globalListeners.delete(update);
      stopGlobalTimer();
    };
  }, [update]);

  return <span className={className}>{text}</span>;
});
