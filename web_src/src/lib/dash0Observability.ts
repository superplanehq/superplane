import { addSignalAttribute, removeSignalAttribute, sendEvent } from "@dash0/sdk-web";
import { isDash0Enabled } from "@/dash0";
import { PAGE_OBSERVABILITY_ATTRIBUTE, resolvePageObservability } from "@/lib/pageObservability";

export type PageReadyAttributes = Record<string, string | number | boolean>;

let pageStartTimeMs: number | null = null;

export function setPageObservabilityTag(pageKey: string): void {
  if (!isDash0Enabled) {
    return;
  }

  removeSignalAttribute(PAGE_OBSERVABILITY_ATTRIBUTE);
  addSignalAttribute(PAGE_OBSERVABILITY_ATTRIBUTE, pageKey);
}

export function clearPageObservabilityTag(): void {
  if (!isDash0Enabled) {
    return;
  }

  removeSignalAttribute(PAGE_OBSERVABILITY_ATTRIBUTE);
}

export function markPageObservabilityStart(): void {
  pageStartTimeMs = performance.now();
}

export function sendPageObservabilityStart(pageKey: string, attributes: Record<string, string>): void {
  if (!isDash0Enabled) {
    return;
  }

  markPageObservabilityStart();
  sendEvent(`${pageKey}.start`, {
    title: `Page start: ${pageKey}`,
    attributes,
  });
}

export function sendPageObservabilityReady(pageKey: string, attributes: PageReadyAttributes): void {
  if (!isDash0Enabled) {
    return;
  }

  const readyAttributes: PageReadyAttributes = { ...attributes };
  if (pageStartTimeMs != null) {
    readyAttributes.duration_ms = Math.round(performance.now() - pageStartTimeMs);
  }

  sendEvent(`${pageKey}.ready`, {
    title: `Page ready: ${pageKey}`,
    attributes: readyAttributes,
  });
}

export function pageObservabilityMetadata(
  pathname: string,
): { title?: string; attributes?: Record<string, string> } | undefined {
  const context = resolvePageObservability(pathname);
  if (!context) {
    return undefined;
  }

  return {
    title: context.pageKey,
    attributes: {
      [PAGE_OBSERVABILITY_ATTRIBUTE]: context.pageKey,
      ...context.attributes,
    },
  };
}
