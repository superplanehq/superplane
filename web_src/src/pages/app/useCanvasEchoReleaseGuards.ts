import { useCallback, type MutableRefObject } from "react";

import { consumeIgnoredMapEcho, LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS, registerIgnoredMapEcho } from "./lib/echo";
import type { CanvasEchoRelease } from "./canvasSaveTypes";

type UseCanvasEchoReleaseGuardsOptions = {
  canvasSaveSessionRef: MutableRefObject<number>;
  ignoredCanvasUpdatedEchoReleasesRef: MutableRefObject<Array<CanvasEchoRelease>>;
  ignoredCanvasVersionUpdatedEchoReleasesRef: MutableRefObject<Map<string, Array<CanvasEchoRelease>>>;
  ignoredCreateDraftEchoReleasesRef: MutableRefObject<Map<string, Array<CanvasEchoRelease>>>;
};

export function useCanvasEchoReleaseGuards({
  canvasSaveSessionRef,
  ignoredCanvasUpdatedEchoReleasesRef,
  ignoredCanvasVersionUpdatedEchoReleasesRef,
  ignoredCreateDraftEchoReleasesRef,
}: UseCanvasEchoReleaseGuardsOptions) {
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

  const registerIgnoredCanvasVersionUpdatedEcho = useCallback(
    (savingVersionId?: string) => {
      if (!savingVersionId) {
        return () => undefined;
      }

      return registerIgnoredMapEcho(ignoredCanvasVersionUpdatedEchoReleasesRef, canvasSaveSessionRef, savingVersionId);
    },
    [canvasSaveSessionRef, ignoredCanvasVersionUpdatedEchoReleasesRef],
  );

  const registerIgnoredCreateDraftEcho = useCallback(
    (targetCanvasId: string) => {
      if (!targetCanvasId) {
        return () => undefined;
      }

      return registerIgnoredMapEcho(ignoredCreateDraftEchoReleasesRef, canvasSaveSessionRef, targetCanvasId);
    },
    [canvasSaveSessionRef, ignoredCreateDraftEchoReleasesRef],
  );

  const consumeIgnoredCanvasUpdatedEcho = useCallback(() => {
    const release = ignoredCanvasUpdatedEchoReleasesRef.current.pop();
    if (!release) return false;

    release();
    return true;
  }, [ignoredCanvasUpdatedEchoReleasesRef]);

  const consumeIgnoredCanvasVersionUpdatedEcho = useCallback(
    (versionId?: string) => consumeIgnoredMapEcho(ignoredCanvasVersionUpdatedEchoReleasesRef, versionId),
    [ignoredCanvasVersionUpdatedEchoReleasesRef],
  );

  const consumeIgnoredCreateDraftEcho = useCallback(
    (targetCanvasId?: string) => consumeIgnoredMapEcho(ignoredCreateDraftEchoReleasesRef, targetCanvasId),
    [ignoredCreateDraftEchoReleasesRef],
  );

  return {
    registerIgnoredCanvasUpdatedEcho,
    registerIgnoredCanvasVersionUpdatedEcho,
    registerIgnoredCreateDraftEcho,
    consumeIgnoredCanvasUpdatedEcho,
    consumeIgnoredCanvasVersionUpdatedEcho,
    consumeIgnoredCreateDraftEcho,
  };
}
