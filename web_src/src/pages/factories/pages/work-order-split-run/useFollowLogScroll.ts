import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";

import { followAfterRunningPhaseChange, isNearLogBottom } from "./followLogScroll";

export type FollowLogScrollOptions = {
  resumeOnBottom?: boolean;
};

/**
 * Follow pins the log scroller to the bottom. Live runner notes grow
 * inside the phase card, so the hook watches the scroller DOM rather
 * than only a parent stream-length tick. A layout resize (plan pane
 * open or wrap) must not look like the user scrolled away.
 */
export function useFollowLogScroll<T extends HTMLElement = HTMLElement>(
  runningPhaseId: string | null,
  contentTick: unknown,
  options?: FollowLogScrollOptions,
) {
  const resumeOnBottom = options?.resumeOnBottom === true;
  // Auto-scroll starts on so the log opens pinned to the newest line and the
  // "Jump to latest" pill stays hidden until the user scrolls up, whether or
  // not a phase is still running.
  const [following, setFollowing] = useState(true);
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

  const releaseScrollIgnore = useCallback(() => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        ignoreScrollRef.current = false;
        const node = scrollRef.current;
        if (node && !isNearLogBottom(node.scrollTop, node.scrollHeight, node.clientHeight)) {
          setFollowing(false);
        }
      });
    });
  }, []);

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
    });
    releaseScrollIgnore();
  }, [releaseScrollIgnore]);

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

  useLayoutEffect(
    () => observeLogMutations(scrollRef.current, following, followingRef, scrollToBottom),
    [following, scrollToBottom],
  );

  useLayoutEffect(
    () => observeLogResize(scrollRef.current, followingRef, ignoreScrollRef, scrollToBottom, releaseScrollIgnore),
    [releaseScrollIgnore, scrollToBottom],
  );

  useEffect(() => bindUserScrollStop(scrollRef.current, followingRef, setFollowing), []);

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

function observeLogMutations(
  el: HTMLElement | null,
  following: boolean,
  followingRef: { current: boolean },
  scrollToBottom: () => void,
) {
  if (!following || !el) {
    return;
  }
  const observer = new MutationObserver(() => {
    if (followingRef.current) {
      scrollToBottom();
    }
  });
  observer.observe(el, { childList: true, subtree: true });
  return () => observer.disconnect();
}

function observeLogResize(
  el: HTMLElement | null,
  followingRef: { current: boolean },
  ignoreScrollRef: { current: boolean },
  scrollToBottom: () => void,
  releaseScrollIgnore: () => void,
) {
  if (!el || typeof ResizeObserver === "undefined") {
    return;
  }
  const observer = new ResizeObserver(() => {
    if (followingRef.current) {
      scrollToBottom();
      return;
    }
    ignoreScrollRef.current = true;
    releaseScrollIgnore();
  });
  observer.observe(el);
  return () => observer.disconnect();
}

function bindUserScrollStop(
  el: HTMLElement | null,
  followingRef: { current: boolean },
  setFollowing: (next: boolean) => void,
) {
  if (!el) {
    return;
  }
  const stopFollowOnUserScroll = () => {
    if (!followingRef.current) {
      return;
    }
    if (isNearLogBottom(el.scrollTop, el.scrollHeight, el.clientHeight)) {
      return;
    }
    setFollowing(false);
  };
  el.addEventListener("wheel", stopFollowOnUserScroll, { passive: true });
  el.addEventListener("touchmove", stopFollowOnUserScroll, { passive: true });
  return () => {
    el.removeEventListener("wheel", stopFollowOnUserScroll);
    el.removeEventListener("touchmove", stopFollowOnUserScroll);
  };
}
