import { useCallback, useEffect, useRef } from "react";

const BOTTOM_THRESHOLD_PX = 16;

function isScrolledToBottom(el: HTMLDivElement): boolean {
  const { scrollTop, scrollHeight, clientHeight } = el;
  return scrollHeight - scrollTop - clientHeight <= BOTTOM_THRESHOLD_PX;
}

export function useScrollToBottom(trigger: unknown) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const stickToBottomRef = useRef(true);

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current;
    if (el) {
      el.scrollTop = el.scrollHeight;
    }
  }, []);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) {
      return;
    }

    const handleScroll = () => {
      stickToBottomRef.current = isScrolledToBottom(el);
    };

    el.addEventListener("scroll", handleScroll);
    handleScroll();

    return () => el.removeEventListener("scroll", handleScroll);
  }, []);

  useEffect(() => {
    if (!stickToBottomRef.current) {
      return;
    }
    scrollToBottom();
  }, [trigger, scrollToBottom]);

  return { scrollRef, scrollToBottom };
}
