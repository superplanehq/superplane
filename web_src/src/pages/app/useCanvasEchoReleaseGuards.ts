import { useCallback, type MutableRefObject } from "react";

import { LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS } from "./lib/echo";
import type { CanvasEchoRelease } from "./canvasSaveTypes";

type UseCanvasEchoReleaseGuardsOptions = {
  canvasSaveSessionRef: MutableRefObject<number>;
  ignoredCanvasUpdatedEchoReleasesRef: MutableRefObject<Array<CanvasEchoRelease>>;
};

export function useCanvasEchoReleaseGuards({
  canvasSaveSessionRef,
  ignoredCanvasUpdatedEchoReleasesRef,
}: UseCanvasEchoReleaseGuardsOptions) {
  const resetLifecycleEchoGuards = useCallback(() => {
    ignoredCanvasUpdatedEchoReleasesRef.current = [];
  }, [ignoredCanvasUpdatedEchoReleasesRef]);

  const registerIgnoredCanvasUpdatedEcho = useCallback(() => {
    const saveSession = canvasSaveSessionRef.current;
    let released = false;
    let timeoutId = 0;
    const release = () => {
      if (released) {
        return;
      }

      released = true;
      window.clearTimeout(timeoutId);
      const releaseIndex = ignoredCanvasUpdatedEchoReleasesRef.current.indexOf(release);
      if (releaseIndex >= 0) {
        ignoredCanvasUpdatedEchoReleasesRef.current.splice(releaseIndex, 1);
      }

      if (canvasSaveSessionRef.current !== saveSession) {
        return;
      }
    };

    ignoredCanvasUpdatedEchoReleasesRef.current.push(release);
    timeoutId = window.setTimeout(release, LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS);

    return release;
  }, [canvasSaveSessionRef, ignoredCanvasUpdatedEchoReleasesRef]);

  const consumeIgnoredCanvasUpdatedEcho = useCallback(() => {
    const release = ignoredCanvasUpdatedEchoReleasesRef.current.pop();
    if (!release) return false;

    release();
    return true;
  }, [ignoredCanvasUpdatedEchoReleasesRef]);

  return {
    registerIgnoredCanvasUpdatedEcho,
    consumeIgnoredCanvasUpdatedEcho,
    resetLifecycleEchoGuards,
  };
}
