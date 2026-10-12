import type { FirstRunArtScene } from "./firstRunArtScene";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunScreen } from "../useFirstRunSetupFlow";

export type FirstRunSphereScreen = FirstRunScreen;

const WELCOME_ART = { mode: "school", background: "#87ae9d", arrowColor: "#eeede9" } as const;
const CONNECT_ART = { mode: "globe", background: "#b7b174", arrowColor: "#eeede9", count: 100 } as const;
const ORGANIZATION_ART = { mode: "globe", background: "#b09532", arrowColor: "#eeede9", count: 400 } as const;
const REPOSITORY_ART = { mode: "globe", background: "#c5ebc3", arrowColor: "#11110e", count: 700 } as const;
const ANALYSIS_ART = { mode: "globe", background: "#EF8D0B", arrowColor: "#11110e", count: 1000 } as const;

function awaitingPill(): NonNullable<FirstRunArtScene["pill"]> {
  const copy = FIRST_RUN_COPY.sphere;
  return { label: copy.discover, value: copy.awaitingCode, light: false };
}

function namedPill(value: string): NonNullable<FirstRunArtScene["pill"]> {
  return { label: FIRST_RUN_COPY.sphere.discover, value, light: true };
}

function sphere(level: number, caption: string, art: FirstRunArtScene, captionHighlight?: string): FirstRunSphereProps {
  return { level, caption, captionHighlight, art };
}

/** Repository, tickets, and agent share the light green globe. */
export function repositorySphereFor(name?: string | null): FirstRunSphereProps {
  const copy = FIRST_RUN_COPY.sphere;
  const label = name?.trim() ?? "";
  return sphere(
    0.74,
    label || copy.captionAwaitingRepository,
    {
      ...REPOSITORY_ART,
      pill: label ? namedPill(label) : awaitingPill(),
    },
    label ? "Repository:" : undefined,
  );
}

/** Sphere for the analysis screen: orange globe with Discover and Verify badges. */
export function analysisSphereFor(selectedRepo: string | null, total: number): FirstRunSphereProps {
  const copy = FIRST_RUN_COPY.sphere;
  const discover = total > 0 ? copy.ticketsFound(total) : selectedRepo || copy.awaitingCode;
  return {
    level: 1,
    animate: true,
    phasesLit: true,
    caption: selectedRepo ?? "",
    captionHighlight: "Scoring:",
    art: {
      ...ANALYSIS_ART,
      badges: { discover, verify: copy.reviewReadyPr },
    },
  };
}

export function sphereFor(
  screen: FirstRunSphereScreen,
  selectedRepo: string | null,
  organization?: string,
): FirstRunSphereProps {
  const copy = FIRST_RUN_COPY.sphere;
  if (screen === "welcome") {
    return sphere(0.14, copy.captionSetup, { ...WELCOME_ART, pill: awaitingPill() });
  }
  if (screen === "host" || screen === "connect") {
    return sphere(0.24, copy.captionConnect, { ...CONNECT_ART, pill: awaitingPill() });
  }
  if (screen === "choose") {
    return selectedRepo
      ? repositorySphereFor(selectedRepo)
      : sphere(0.58, copy.captionAwaitingRepository, { ...ORGANIZATION_ART, pill: awaitingPill() });
  }
  return repositorySphereFor(selectedRepo ?? organization);
}
