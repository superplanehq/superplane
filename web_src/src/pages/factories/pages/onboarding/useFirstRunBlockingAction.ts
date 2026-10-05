import { useCallback, useEffect, useRef, useState } from "react";

type FirstRunBlockingAction =
  | "opening-github"
  | "switching-github-account"
  | "saving-repository"
  | "saving-ticket-source"
  | "connecting-jira"
  | "connecting-linear"
  | "finishing-setup";

/** One first-run action at a time. The screen stays disabled while it runs. */
export function useFirstRunBlockingAction() {
  const lock = useRef(false);
  const [action, setAction] = useState<FirstRunBlockingAction | null>(null);
  const begin = useCallback((next: FirstRunBlockingAction): boolean => {
    if (lock.current) return false;
    lock.current = true;
    setAction(next);
    return true;
  }, []);
  const finish = useCallback(() => {
    lock.current = false;
    setAction(null);
  }, []);
  // A page restored from the back-forward cache still holds the action of
  // the navigation that left it, for example "Opening GitHub…".
  useEffect(() => {
    const onPageShow = (event: PageTransitionEvent) => {
      if (event.persisted) finish();
    };
    window.addEventListener("pageshow", onPageShow);
    return () => window.removeEventListener("pageshow", onPageShow);
  }, [finish]);
  const run = useCallback(
    async (next: FirstRunBlockingAction, operation: () => Promise<void>) => {
      if (!begin(next)) return;
      try {
        await operation();
      } finally {
        finish();
      }
    },
    [begin, finish],
  );
  const runUntilNavigation = useCallback(
    async (next: FirstRunBlockingAction, operation: () => Promise<boolean>) => {
      if (!begin(next)) return;
      try {
        const navigationStarted = await operation();
        if (!navigationStarted) finish();
      } catch (error) {
        finish();
        throw error;
      }
    },
    [begin, finish],
  );
  return { action, busy: action !== null, begin, finish, setAction, run, runUntilNavigation };
}

export type FirstRunBlocking = ReturnType<typeof useFirstRunBlockingAction>;
