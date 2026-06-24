import { useCallback, useRef, type MutableRefObject } from "react";

import {
  armCreateDraftEcho,
  consumeCreateDraftEcho,
  consumeIgnoredMapEcho,
  type CreateDraftEchoMap,
  LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS,
  registerCreateDraftEcho,
  registerIgnoredMapEcho,
} from "./lib/echo";
import type { CanvasEchoRelease } from "./canvasSaveTypes";

type UseCanvasEchoReleaseGuardsOptions = {
  canvasSaveSessionRef: MutableRefObject<number>;
  ignoredCanvasUpdatedEchoReleasesRef: MutableRefObject<Array<CanvasEchoRelease>>;
  ignoredCanvasVersionUpdatedEchoReleasesRef: MutableRefObject<Map<string, Array<CanvasEchoRelease>>>;
};

export function useCanvasEchoReleaseGuards({
  canvasSaveSessionRef,
  ignoredCanvasUpdatedEchoReleasesRef,
  ignoredCanvasVersionUpdatedEchoReleasesRef,
}: UseCanvasEchoReleaseGuardsOptions) {
  const ignoredCreateDraftEchoReleasesRef = useRef<CreateDraftEchoMap>(new Map());

  const resetLifecycleEchoGuards = useCallback(() => {
    ignoredCanvasUpdatedEchoReleasesRef.current = [];
    ignoredCanvasVersionUpdatedEchoReleasesRef.current.clear();
    ignoredCreateDraftEchoReleasesRef.current.clear();
  }, [ignoredCanvasUpdatedEchoReleasesRef, ignoredCanvasVersionUpdatedEchoReleasesRef]);

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

      return registerCreateDraftEcho(ignoredCreateDraftEchoReleasesRef, canvasSaveSessionRef, targetCanvasId);
    },
    [canvasSaveSessionRef],
  );

  const armIgnoredCreateDraftEcho = useCallback(
    (targetCanvasId: string, versionId: string, release: CanvasEchoRelease) => {
      armCreateDraftEcho(ignoredCreateDraftEchoReleasesRef, targetCanvasId, versionId, release);
    },
    [],
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
    (targetCanvasId?: string, eventVersionId?: string) =>
      consumeCreateDraftEcho(ignoredCreateDraftEchoReleasesRef, targetCanvasId, eventVersionId),
    [],
  );

  return {
    registerIgnoredCanvasUpdatedEcho,
    registerIgnoredCanvasVersionUpdatedEcho,
    registerIgnoredCreateDraftEcho,
    armIgnoredCreateDraftEcho,
    consumeIgnoredCanvasUpdatedEcho,
    consumeIgnoredCanvasVersionUpdatedEcho,
    consumeIgnoredCreateDraftEcho,
    resetLifecycleEchoGuards,
  };
}
