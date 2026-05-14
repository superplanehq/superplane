import { useCallback, useEffect, useRef } from "react";

export function useScrollToBottom(text: string) {
  const scrollRef = useRef<HTMLPreElement>(null);

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [text, scrollToBottom]);

  return { scrollRef, scrollToBottom };
}
