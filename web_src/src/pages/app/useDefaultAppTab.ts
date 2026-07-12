import { useEffect, useRef } from "react";

import { readLastVisitedAppTab, recordLastVisitedAppTab } from "@/lib/lastVisitedAppTab";

import { urlDeepLinksWithoutTabPick, urlViewFlagsToTab, type UrlViewFlags } from "./defaultAppTab";

type UseDefaultAppTabOptions = {
  canvasId: string | undefined;
  urlViewFlags: UrlViewFlags;
  searchParams: URLSearchParams;
};

/**
 * Persists the current app tab to localStorage. First-visit resolution and
 * the last-visited redirect live in AppDefaultTabGate, before AppPage mounts;
 * by the time this hook runs the URL is already on the correct tab.
 *
 * Two landings are exempt from persistence: closing run inspection (which
 * lands on Canvas without the user picking it) and deep links that land on a
 * tab without a `view` param (`?version=`, `?edit=`, `?sidebar=`/`?node=`,
 * `?file=`). Later tab changes on the same visit are recorded as usual.
 */
export function useDefaultAppTab({ canvasId, urlViewFlags, searchParams }: UseDefaultAppTabOptions) {
  const currentTab = urlViewFlagsToTab(urlViewFlags);

  // Deep-link landing on mount must not be persisted; consumed once by the
  // record effect below and then reset for later tab changes on the same
  // visit.
  const deepLinkLandingRef = useRef(urlDeepLinksWithoutTabPick(searchParams));
  // Whether the current render is in run inspection (`?run=`). Closing a run
  // lands the user on Canvas without them picking a tab; that landing must
  // not overwrite the stored tab.
  const inRunInspectionRef = useRef(currentTab === null);

  // React Router reuses the same component across app navigations (e.g. via
  // the command palette). Reset the refs when the canvas id changes so a new
  // app starts fresh — otherwise a stale deep-link/run flag would suppress
  // recording the new app's landing.
  const refsOwnerCanvasIdRef = useRef(canvasId);
  if (refsOwnerCanvasIdRef.current !== canvasId) {
    refsOwnerCanvasIdRef.current = canvasId;
    deepLinkLandingRef.current = urlDeepLinksWithoutTabPick(searchParams);
    inRunInspectionRef.current = currentTab === null;
  }

  useEffect(() => {
    if (!canvasId) return;

    if (currentTab === null) {
      inRunInspectionRef.current = true;
      return;
    }

    if (inRunInspectionRef.current) {
      inRunInspectionRef.current = false;
      return;
    }

    if (deepLinkLandingRef.current) {
      deepLinkLandingRef.current = false;
      return;
    }

    if (readLastVisitedAppTab(canvasId) === currentTab) return;

    recordLastVisitedAppTab(canvasId, currentTab);
  }, [canvasId, currentTab]);
}
