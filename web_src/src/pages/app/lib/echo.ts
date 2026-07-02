import type { MutableRefObject } from "react";

import type { CanvasEchoRelease } from "../canvasSaveTypes";

export const LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS = 5000;

export type CreateDraftEchoSlot = {
  release: CanvasEchoRelease;
  expectedVersionId?: string;
};

export type CreateDraftEchoRegistration = {
  slots: CreateDraftEchoSlot[];
};

export type CreateDraftEchoMap = Map<string, CreateDraftEchoRegistration>;

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

function removeCreateDraftSlot(
  echoMap: MutableRefObject<CreateDraftEchoMap>,
  canvasId: string,
  release: CanvasEchoRelease,
): void {
  const registration = echoMap.current.get(canvasId);
  if (!registration) {
    return;
  }

  registration.slots = registration.slots.filter((slot) => slot.release !== release);
  if (registration.slots.length === 0) {
    echoMap.current.delete(canvasId);
  }
}

export function registerCreateDraftEcho(
  echoMap: MutableRefObject<CreateDraftEchoMap>,
  canvasSaveSessionRef: MutableRefObject<number>,
  canvasId: string,
): () => void {
  const saveSession = canvasSaveSessionRef.current;
  let released = false;
  let timeoutId = 0;
  const release = () => {
    if (released) {
      return;
    }

    released = true;
    window.clearTimeout(timeoutId);
    removeCreateDraftSlot(echoMap, canvasId, release);

    if (canvasSaveSessionRef.current !== saveSession) {
      return;
    }
  };

  const registration = echoMap.current.get(canvasId) ?? { slots: [] };
  registration.slots.push({ release });
  echoMap.current.set(canvasId, registration);
  timeoutId = window.setTimeout(release, LOCAL_CANVAS_LIFECYCLE_ECHO_TTL_MS);

  return release;
}

export function armCreateDraftEcho(
  echoMap: MutableRefObject<CreateDraftEchoMap>,
  canvasId: string,
  versionId: string,
  release: CanvasEchoRelease,
): void {
  if (!canvasId || !versionId) {
    return;
  }

  const registration = echoMap.current.get(canvasId);
  const slot = registration?.slots.find((candidate) => candidate.release === release);
  if (!slot) {
    return;
  }

  slot.expectedVersionId = versionId;
}

export function consumeCreateDraftEcho(
  echoMap: MutableRefObject<CreateDraftEchoMap>,
  canvasId?: string,
  eventVersionId?: string,
): boolean {
  if (!canvasId || !eventVersionId) {
    return false;
  }

  const registration = echoMap.current.get(canvasId);
  if (!registration) {
    return false;
  }

  const slotIndex = registration.slots.findIndex((slot) => slot.expectedVersionId === eventVersionId);
  if (slotIndex < 0) {
    return false;
  }

  const [slot] = registration.slots.splice(slotIndex, 1);
  if (registration.slots.length === 0) {
    echoMap.current.delete(canvasId);
  }

  slot.release();
  return true;
}
