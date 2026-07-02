import { useEffect, useLayoutEffect, useRef } from "react";
import type { UseInfiniteQueryResult } from "@tanstack/react-query";
import type { AgentMessagesPage } from "@/hooks/useAgentChats";

/** Manages scroll-to-bottom and older-page loading for the chat. */
export function useChatScroll(
  messagesQuery: UseInfiniteQueryResult<{ pages: AgentMessagesPage[]; pageParams: unknown[] }>,
  chatId: string,
  messageCount: number,
  showThinking: boolean,
) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const previousScrollHeight = useRef<number | null>(null);

  const queryRef = useRef(messagesQuery);
  queryRef.current = messagesQuery;

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    const onScroll = () => {
      if (el.scrollTop > 24) return;
      const q = queryRef.current;
      if (!q.hasNextPage || q.isFetchingNextPage) return;
      previousScrollHeight.current = el.scrollHeight;
      void q.fetchNextPage();
    };
    el.addEventListener("scroll", onScroll);
    return () => el.removeEventListener("scroll", onScroll);
  }, []);

  useLayoutEffect(() => {
    const el = scrollRef.current;
    if (!el) return;
    if (previousScrollHeight.current !== null) {
      el.scrollTop = el.scrollHeight - previousScrollHeight.current;
      previousScrollHeight.current = null;
      return;
    }
    el.scrollTop = el.scrollHeight;
  }, [chatId, messageCount, showThinking]);

  return scrollRef;
}
