import { useCallback, useEffect, useLayoutEffect, useRef, useState } from "react";

import {
  distanceFromLogBottom,
  followAfterRunningPhaseChange,
  isNearLogBottom,
  nextFollowAfterScroll,
  showJumpToLatest,
} from "./followLogScroll";

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
  const [jumpToLatest, setJumpToLatest] = useState(false);
  const followingRef = useRef(following);
  followingRef.current = following;
  const previousRunningPhaseIdRef = useRef(runningPhaseId);
  const scrollRef = useRef<T>(null);
  const ignoreScrollRef = useRef(false);
  const lastScrollTopRef = useRef(0);

  useEffect(() => {
    const previousRunningPhaseId = previousRunningPhaseIdRef.current;
    previousRunningPhaseIdRef.current = runningPhaseId;
    if (previousRunningPhaseId === runningPhaseId) {
      return;
    }
    setFollowing((wasFollowing) => followAfterRunningPhaseChange(wasFollowing, previousRunningPhaseId, runningPhaseId));
  }, [runningPhaseId]);

  const syncJumpToLatest = useCallback((nextFollowing: boolean, node: HTMLElement) => {
    setJumpToLatest(
      showJumpToLatest(nextFollowing, distanceFromLogBottom(node.scrollTop, node.scrollHeight, node.clientHeight)),
    );
  }, []);

  const stopFollow = useCallback(() => {
    followingRef.current = false;
    setFollowing(false);
    const node = scrollRef.current;
    if (node) {
      syncJumpToLatest(false, node);
    }
  }, [syncJumpToLatest]);

  const releaseScrollIgnore = useCallback(() => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        ignoreScrollRef.current = false;
        const node = scrollRef.current;
        if (!node) {
          return;
        }
        lastScrollTopRef.current = node.scrollTop;
        if (!isNearLogBottom(node.scrollTop, node.scrollHeight, node.clientHeight)) {
          stopFollow();
        }
      });
    });
  }, [stopFollow]);

  const scrollToBottom = useCallback(() => {
    const el = scrollRef.current;
    if (!el) {
      return;
    }
    ignoreScrollRef.current = true;
    el.scrollTop = el.scrollHeight;
    lastScrollTopRef.current = el.scrollTop;
    requestAnimationFrame(() => {
      const node = scrollRef.current;
      if (node && followingRef.current) {
        node.scrollTop = node.scrollHeight;
        lastScrollTopRef.current = node.scrollTop;
      }
    });
    releaseScrollIgnore();
  }, [releaseScrollIgnore]);

  const setFollow = useCallback(
    (next: boolean) => {
      followingRef.current = next;
      setFollowing(next);
      if (next) {
        setJumpToLatest(false);
        scrollToBottom();
        return;
      }
      const node = scrollRef.current;
      if (node) {
        syncJumpToLatest(false, node);
      }
    },
    [scrollToBottom, syncJumpToLatest],
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

  useEffect(() => bindUserScrollStop(scrollRef.current, followingRef, stopFollow), [stopFollow]);

  const onScroll = useCallback(() => {
    if (ignoreScrollRef.current) {
      return;
    }
    const el = scrollRef.current;
    if (!el) {
      return;
    }
    const distance = distanceFromLogBottom(el.scrollTop, el.scrollHeight, el.clientHeight);
    const next = nextFollowAfterScroll({
      following: followingRef.current,
      resumeOnBottom,
      distanceFromBottom: distance,
      scrollingUp: el.scrollTop < lastScrollTopRef.current,
    });
    lastScrollTopRef.current = el.scrollTop;
    followingRef.current = next;
    setFollowing(next);
    setJumpToLatest(showJumpToLatest(next, distance));
  }, [resumeOnBottom]);

  return { following, setFollowing: setFollow, showJumpToLatest: jumpToLatest, scrollRef, onScroll };
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

function bindUserScrollStop(el: HTMLElement | null, followingRef: { current: boolean }, stopFollow: () => void) {
  if (!el) {
    return;
  }
  let touchStartY = 0;
  const stopOnUpwardIntent = (upward: boolean) => {
    if (!followingRef.current || !upward) {
      return;
    }
    stopFollow();
  };
  const onWheel = (event: WheelEvent) => {
    stopOnUpwardIntent(event.deltaY < 0);
  };
  const onTouchStart = (event: TouchEvent) => {
    touchStartY = event.touches[0]?.clientY ?? 0;
  };
  const onTouchMove = (event: TouchEvent) => {
    const y = event.touches[0]?.clientY ?? touchStartY;
    stopOnUpwardIntent(y > touchStartY);
  };
  el.addEventListener("wheel", onWheel, { passive: true });
  el.addEventListener("touchstart", onTouchStart, { passive: true });
  el.addEventListener("touchmove", onTouchMove, { passive: true });
  return () => {
    el.removeEventListener("wheel", onWheel);
    el.removeEventListener("touchstart", onTouchStart);
    el.removeEventListener("touchmove", onTouchMove);
  };
}
