import { useEffect, useRef } from "react";
import type { FactoriesWorkOrderEvent } from "@/api-client";
import { showInfoToast } from "@/lib/toast";

interface CommentAddedPayload {
  body?: string;
  author?: { userId?: string };
  mentions?: { id?: string }[];
}

/**
 * Surfaces an in-app toast the moment a fresh `order.comment.added` event
 * mentioning the current user shows up in an already-loaded `events` list —
 * the "you're already looking at this order" half of mention notifications
 * (the websocket `work_order_updated` message that triggers the refetch
 * carries no payload of its own, see `useFactoryWebsocket`). Not-viewing
 * notifications are a fast-follow (see mentions plan §5); there's no
 * notification center in the app yet to surface them in otherwise.
 *
 * Skips the very first render (so opening a page full of history doesn't
 * toast for old mentions) and self-mentions.
 */
export function useWorkOrderMentionToast(
  events: FactoriesWorkOrderEvent[],
  currentUserId: string | undefined,
  resolveUserName: (userId: string | undefined) => string | undefined,
  orderLabel: string,
): void {
  const seenKeys = useRef<Set<string> | null>(null);

  useEffect(() => {
    const keys = new Set(events.map(commentEventKey));

    if (!seenKeys.current) {
      seenKeys.current = keys;
      return;
    }

    const previouslySeen = seenKeys.current;
    seenKeys.current = keys;

    if (!currentUserId) {
      return;
    }

    for (const event of events) {
      if (event.type !== "order.comment.added" || previouslySeen.has(commentEventKey(event))) {
        continue;
      }

      const payload = (event.event ?? {}) as CommentAddedPayload;
      const authorId = payload.author?.userId;
      const mentionedIds = (payload.mentions ?? [])
        .map((mention) => mention.id)
        .filter((id): id is string => Boolean(id));

      if (authorId === currentUserId || !mentionedIds.includes(currentUserId)) {
        continue;
      }

      const authorName = resolveUserName(authorId) ?? "Someone";
      showInfoToast(`${authorName} mentioned you in ${orderLabel}.`);
    }
  }, [events, currentUserId, resolveUserName, orderLabel]);
}

function commentEventKey(event: FactoriesWorkOrderEvent): string {
  const payload = event.event as { body?: string } | undefined;
  return `${event.type ?? ""}|${event.timestamp ?? ""}|${payload?.body ?? ""}`;
}
