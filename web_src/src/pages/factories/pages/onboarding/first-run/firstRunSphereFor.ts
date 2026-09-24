import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunScreen } from "../useFirstRunSetupFlow";

/** The account picker is a page of the connect screen with its own state. */
export type FirstRunSphereScreen = FirstRunScreen | "organization";

/**
 * The Discover chip keeps the latest confirmed context: the organization
 * once it is chosen, then the repository once it is selected. The chip is a
 * ghost placeholder until the first choice exists.
 */
function discoverChip(value: string | null | undefined): FirstRunSphereProps["leftChip"] {
  const copy = FIRST_RUN_COPY.sphere;
  if (!value) return { label: copy.discover, value: copy.awaitingCode, tone: "ghost" };
  return { label: copy.discover, value };
}

/** Sphere for the analysis screen: fully lit, flickering while scoring runs. */
export function analysisSphereFor(selectedRepo: string | null, total: number): FirstRunSphereProps {
  const copy = FIRST_RUN_COPY.sphere;
  return {
    level: 1,
    animate: true,
    phasesLit: true,
    caption: selectedRepo ?? "",
    captionHighlight: "Scoring:",
    leftChip:
      total > 0
        ? { label: copy.discover, value: copy.ticketsFound(total), tone: "amber" }
        : { label: copy.discover, value: copy.awaitingCode, tone: "ghost" },
    rightChip: { label: copy.verify, value: copy.reviewReadyPr, tone: "amber" },
  };
}

export function sphereFor(
  screen: FirstRunSphereScreen,
  selectedRepo: string | null,
  organization?: string,
): FirstRunSphereProps {
  const copy = FIRST_RUN_COPY.sphere;
  if (screen === "welcome") return { level: 0.14, caption: copy.captionSetup };
  if (screen === "connect") {
    return {
      level: 0.24,
      caption: copy.captionConnect,
      leftChip: discoverChip(null),
    };
  }
  if (screen === "organization") {
    return {
      level: 0.42,
      caption: copy.captionOrganization,
      leftChip: discoverChip(null),
    };
  }
  if (screen === "choose") {
    return selectedRepo
      ? {
          level: 0.74,
          caption: selectedRepo,
          captionHighlight: "Repository:",
          leftChip: discoverChip(selectedRepo),
        }
      : { level: 0.58, caption: copy.captionAwaitingRepository, leftChip: discoverChip(organization) };
  }
  return {
    level: 0.9,
    caption: copy.captionTickets,
    leftChip: discoverChip(selectedRepo ?? organization),
    rightChip: { label: copy.verify, value: copy.reviewReadyPr, tone: "ghost" },
  };
}
