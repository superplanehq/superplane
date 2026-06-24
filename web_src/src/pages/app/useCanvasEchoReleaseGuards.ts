import { useCallback, type MutableRefObject } from "react";

import type { CanvasEchoRelease } from "./canvasSaveTypes";

const LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS = 5000;

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

      const saveSession = canvasSaveSessionRef.current;
      const currentReleases = ignoredCanvasVersionUpdatedEchoReleasesRef.current.get(savingVersionId) || [];
      let released = false;
      let timeoutId = 0;
      const release = () => {
        if (released) {
          return;
        }

        released = true;
        window.clearTimeout(timeoutId);
        const releases = ignoredCanvasVersionUpdatedEchoReleasesRef.current.get(savingVersionId);
        if (releases) {
          const releaseIndex = releases.indexOf(release);
          if (releaseIndex >= 0) {
            releases.splice(releaseIndex, 1);
          }
          if (releases.length === 0) {
            ignoredCanvasVersionUpdatedEchoReleasesRef.current.delete(savingVersionId);
          }
        }

        if (canvasSaveSessionRef.current !== saveSession) {
          return;
        }
      };

      currentReleases.push(release);
      ignoredCanvasVersionUpdatedEchoReleasesRef.current.set(savingVersionId, currentReleases);
      timeoutId = window.setTimeout(release, LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS);

      return release;
    },
    [canvasSaveSessionRef, ignoredCanvasVersionUpdatedEchoReleasesRef],
  );

  const consumeIgnoredCanvasUpdatedEcho = useCallback(() => {
    const release = ignoredCanvasUpdatedEchoReleasesRef.current.pop();
    if (!release) return false;

    release();
    return true;
  }, [ignoredCanvasUpdatedEchoReleasesRef]);

  const registerIgnoredCreateDraftEcho = useCallback(
    (targetCanvasId: string) => {
      if (!targetCanvasId) {
        return () => undefined;
      }

      const saveSession = canvasSaveSessionRef.current;
      const currentReleases = ignoredCreateDraftEchoReleasesRef.current.get(targetCanvasId) || [];
      let released = false;
      let timeoutId = 0;
      const release = () => {
        if (released) {
          return;
        }

        released = true;
        window.clearTimeout(timeoutId);
        const releases = ignoredCreateDraftEchoReleasesRef.current.get(targetCanvasId);
        if (releases) {
          const releaseIndex = releases.indexOf(release);
          if (releaseIndex >= 0) {
            releases.splice(releaseIndex, 1);
          }
          if (releases.length === 0) {
            ignoredCreateDraftEchoReleasesRef.current.delete(targetCanvasId);
          }
        }

        if (canvasSaveSessionRef.current !== saveSession) {
          return;
        }
      };

      currentReleases.push(release);
      ignoredCreateDraftEchoReleasesRef.current.set(targetCanvasId, currentReleases);
      timeoutId = window.setTimeout(release, LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS);

      return release;
    },
    [canvasSaveSessionRef, ignoredCreateDraftEchoReleasesRef],
  );

  const consumeIgnoredCanvasVersionUpdatedEcho = useCallback(
    (versionId?: string) => {
      if (!versionId) return false;

      const releases = ignoredCanvasVersionUpdatedEchoReleasesRef.current.get(versionId);
      if (!releases) return false;

      const release = releases.pop();
      if (!release) return false;

      if (releases.length === 0) {
        ignoredCanvasVersionUpdatedEchoReleasesRef.current.delete(versionId);
      }

      release();
      return true;
    },
    [ignoredCanvasVersionUpdatedEchoReleasesRef],
  );

  const consumeIgnoredCreateDraftEcho = useCallback(
    (targetCanvasId?: string) => {
      if (!targetCanvasId) return false;

      const releases = ignoredCreateDraftEchoReleasesRef.current.get(targetCanvasId);
      if (!releases) return false;

      const release = releases.pop();
      if (!release) return false;

      if (releases.length === 0) {
        ignoredCreateDraftEchoReleasesRef.current.delete(targetCanvasId);
      }

      release();
      return true;
    },
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
