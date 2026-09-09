import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunScreen } from "../useFirstRunSetupFlow";

/** The account picker is a page of the connect screen with its own state. */
export type FirstRunSphereScreen = FirstRunScreen | "organization";

export function sphereFor(
  screen: FirstRunSphereScreen,
  initial: boolean,
  selectedRepo: string | null,
): FirstRunSphereProps | undefined {
  if (!initial) return undefined;
  const copy = FIRST_RUN_COPY.sphere;
  if (screen === "welcome") return { level: 0.14, caption: copy.captionSetup };
  if (screen === "connect") {
    return {
      level: 0.24,
      caption: copy.captionConnect,
      leftChip: { label: copy.discover, value: copy.awaitingCode, tone: "ghost" },
    };
  }
  if (screen === "organization") {
    return {
      level: 0.42,
      caption: copy.captionOrganization,
      leftChip: { label: copy.discover, value: copy.awaitingCode, tone: "ghost" },
    };
  }
  if (screen === "choose") {
    return selectedRepo
      ? {
          level: 0.74,
          caption: selectedRepo,
          captionHighlight: "Repository:",
          leftChip: { label: copy.discover, value: selectedRepo },
        }
      : { level: 0.58, caption: copy.captionAwaitingRepository };
  }
  return {
    level: 0.9,
    caption: copy.captionTickets,
    leftChip: selectedRepo ? { label: copy.discover, value: selectedRepo } : undefined,
    rightChip: { label: copy.verify, value: copy.reviewReadyPr, tone: "ghost" },
  };
}
