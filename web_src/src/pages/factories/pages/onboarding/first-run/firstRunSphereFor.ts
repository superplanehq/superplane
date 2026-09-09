import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunScreen } from "../useFirstRunSetupFlow";

export function sphereFor(
  screen: FirstRunScreen,
  initial: boolean,
  selectedRepo: string | null,
): FirstRunSphereProps | undefined {
  if (!initial) return undefined;
  const copy = FIRST_RUN_COPY.sphere;
  if (screen === "welcome") return { level: 0.3, caption: copy.captionSetup };
  if (screen === "connect") {
    return {
      level: 0.34,
      caption: copy.captionConnect,
      leftChip: { label: copy.discover, value: copy.awaitingCode, tone: "ghost" },
    };
  }
  if (screen === "choose") {
    return selectedRepo
      ? {
          level: 0.72,
          caption: selectedRepo,
          captionHighlight: "Repository:",
          leftChip: { label: copy.discover, value: selectedRepo },
        }
      : { level: 0.55, caption: copy.captionAwaitingRepository };
  }
  return {
    level: 0.85,
    caption: copy.captionTickets,
    leftChip: selectedRepo ? { label: copy.discover, value: selectedRepo } : undefined,
    rightChip: { label: copy.verify, value: copy.reviewReadyPr, tone: "ghost" },
  };
}
