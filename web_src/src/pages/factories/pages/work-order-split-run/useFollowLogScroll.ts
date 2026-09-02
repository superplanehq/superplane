import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";

import { followAfterRunningPhaseChange, isNearLogBottom } from "./followLogScroll";

export type FollowLogScrollOptions = {
  resumeOnBottom?: boolean;
};

/**
 * Follow pins the log scroller to the bottom. Live runner notes grow
 * inside the phase card, so the hook watches the scroller DOM rather
 * than only a parent stream-length tick.
 */
export function useFollowLogScroll<T extends HTMLElement = HTMLElement>(
  runningPhaseId: string | null,
  contentTick: unknown,
  options?: FollowLogScrollOptions,
) {
  const resumeOnBottom = options?.resumeOnBottom === true;
  const [following, setFollowing] = useState(() => runningPhaseId != null);
  const followingRef = useRef(following);
  followingRef.current = following;
  const previousRunningPhaseIdRef = useRef(runningPhaseId);
  const scrollRef = useRef<T>(null);
  const ignoreScrollRef = useRef(false);

  useEffect(() => {
    const previousRunningPhaseId = previousRunningPhaseIdRef.current;
    previousRunningPhaseIdRef.current = runningPhaseId;
    if (previousRunningPhaseId === runningPhaseId) {
      return;
    }
    setFollowing((wasFollowing) => followAfterRunningPhaseChange(wasFollowing, previousRunningPhaseId, runningPhaseId));
  }, [runningPhaseId]);

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current;
    if (!el) {
      return;
    }
    ignoreScrollRef.current = true;
    el.scrollTop = el.scrollHeight;
    requestAnimationFrame(() => {
      const node = scrollRef.current;
      if (node && followingRef.current) {
        node.scrollTop = node.scrollHeight;
      }
      requestAnimationFrame(() => {
        ignoreScrollRef.current = false;
      });
    });
  }, []);

  const setFollow = useCallback(
    (next: boolean) => {
      setFollowing(next);
      if (next) {
        scrollToBottom();
      }
    },
    [scrollToBottom],
  );

  useLayoutEffect(() => {
    if (!following) {
      return;
    }
    scrollToBottom();
  }, [contentTick, following, scrollToBottom]);

  useLayoutEffect(() => {
    if (!following) {
      return;
    }
    const el = scrollRef.current;
    if (!el) {
      return;
    }
    const observer = new MutationObserver(() => {
      if (followingRef.current) {
        scrollToBottom();
      }
    });
    observer.observe(el, { childList: true, subtree: true });
    return () => observer.disconnect();
  }, [following, scrollToBottom]);

  const onScroll = useCallback(() => {
    if (ignoreScrollRef.current) {
      return;
    }
    const el = scrollRef.current;
    if (!el) {
      return;
    }
    if (!isNearLogBottom(el.scrollTop, el.scrollHeight, el.clientHeight)) {
      setFollowing(false);
      return;
    }
    if (resumeOnBottom) {
      setFollowing(true);
    }
  }, [resumeOnBottom]);

  return { following, setFollowing: setFollow, scrollRef, onScroll };
}
