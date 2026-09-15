import type { FactoriesFactoryLine } from "@/api-client";

export type AutomationLineRef = {
  id: string;
  name: string;
};

function lineUsesAutomation(line: FactoriesFactoryLine, appId: string): boolean {
  return (line.steps ?? []).some((step) => step.app?.app?.trim() === appId);
}

/**
 * Lines that run `appId` as a step, in factory order. A line appears once
 * even when several of its steps use the same automation.
 */
export function linesUsingAutomation(
  lines: FactoriesFactoryLine[] | undefined | null,
  appId: string | undefined | null,
): AutomationLineRef[] {
  const trimmedAppId = appId?.trim();
  if (!trimmedAppId) {
    return [];
  }

  return (lines ?? []).flatMap((line) => {
    const id = line.id?.trim();
    if (!id || !lineUsesAutomation(line, trimmedAppId)) {
      return [];
    }
    return [{ id, name: line.name ?? "" }];
  });
}
