import type { CanvasesCanvasRun, FactoriesWorkOrder } from "@/api-client";
import { shortId } from "@/ui/Runs/runPresentation";

/** Title shown when a factory run has no linked task yet. */
export function factoryAutomationRunTitle(run: CanvasesCanvasRun): string {
  const customName = run.rootEvent?.customName?.trim();
  if (customName) {
    return customName;
  }
  return run.id ? `Run ${shortId(run.id)}` : "Run";
}

export function factoryAutomationRunMatchesQuery(
  query: string,
  run: CanvasesCanvasRun,
  order?: FactoriesWorkOrder,
): boolean {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    return true;
  }

  const haystack = [factoryAutomationRunTitle(run), order?.title, order?.key, run.id]
    .filter((value): value is string => Boolean(value))
    .join(" ")
    .toLowerCase();

  return haystack.includes(needle);
}
