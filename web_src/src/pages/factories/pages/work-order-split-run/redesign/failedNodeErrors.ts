import type { SplitRunStreamLine } from "../splitRunMocks";
import { isRunnerComponent } from "../streamNotesFromLiveLog";

export type FailedNodeError = {
  id: string;
  name: string;
  message: string;
  iconSlug?: string;
};

export function failedNonRunnerErrors(lines: SplitRunStreamLine[]): FailedNodeError[] {
  return lines
    .filter((line) => !line.note && line.status === "failed" && !isRunnerComponent(line.component))
    .map((line) => ({
      id: line.id,
      name: line.componentName,
      message: line.detail?.trim() || "This node failed.",
      iconSlug: line.iconSlug,
    }));
}
