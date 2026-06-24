import type { MutableRefObject } from "react";

import type { CanvasEchoRelease } from "../canvasSaveTypes";

export const LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS = 5000;

export function registerIgnoredMapEcho(
  echoMap: MutableRefObject<Map<string, Array<CanvasEchoRelease>>>,
  canvasSaveSessionRef: MutableRefObject<number>,
  key: string,
): () => void {
  const saveSession = canvasSaveSessionRef.current;
  const currentReleases = echoMap.current.get(key) || [];
  let released = false;
  let timeoutId = 0;
  const release = () => {
    if (released) {
      return;
    }

    released = true;
    window.clearTimeout(timeoutId);
    const releases = echoMap.current.get(key);
    if (releases) {
      const releaseIndex = releases.indexOf(release);
      if (releaseIndex >= 0) {
        releases.splice(releaseIndex, 1);
      }
      if (releases.length === 0) {
        echoMap.current.delete(key);
      }
    }

    if (canvasSaveSessionRef.current !== saveSession) {
      return;
    }
  };

  currentReleases.push(release);
  echoMap.current.set(key, currentReleases);
  timeoutId = window.setTimeout(release, LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS);

  return release;
}

export function consumeIgnoredMapEcho(
  echoMap: MutableRefObject<Map<string, Array<CanvasEchoRelease>>>,
  key?: string,
): boolean {
  if (!key) {
    return false;
  }

  const releases = echoMap.current.get(key);
  if (!releases) {
    return false;
  }

  const release = releases.pop();
  if (!release) {
    return false;
  }

  if (releases.length === 0) {
    echoMap.current.delete(key);
  }

  release();
  return true;
}
