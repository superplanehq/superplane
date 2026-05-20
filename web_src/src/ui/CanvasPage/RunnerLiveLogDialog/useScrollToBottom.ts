import { useCallback, useEffect, useRef } from "react";

export function useScrollToBottom(trigger: unknown) {
  const scrollRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [trigger, scrollToBottom]);

  return { scrollRef, scrollToBottom };
}
