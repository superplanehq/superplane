import type { FactoryResolutionStatus } from "../../lib/factoryKeyResolution";

export type InitialWorkspaceOnboardingPhase = "ready" | "loading" | "reresolve" | "error";

export type InitialWorkspaceOnboardingView = "setup" | "loading" | "error";

export type InitialWorkspaceLookup = {
  listError: boolean;
  resolution: FactoryResolutionStatus;
  describeError: boolean;
  describeReady: boolean;
  workspaceOpen: boolean;
  reresolveRunning: boolean;
  reresolveAttempted: boolean;
};

export type InitialWorkspaceOnboardingDecision = {
  phase: InitialWorkspaceOnboardingPhase;
  view: InitialWorkspaceOnboardingView;
};

export function decideInitialWorkspaceOnboarding(lookup: InitialWorkspaceLookup): InitialWorkspaceOnboardingDecision {
  const phase = initialWorkspacePhase(lookup);
  return { phase, view: initialWorkspaceView(phase, lookup.workspaceOpen) };
}

function initialWorkspacePhase(lookup: InitialWorkspaceLookup): InitialWorkspaceOnboardingPhase {
  const lookupFailed = lookup.listError || lookup.resolution === "not-found" || lookup.describeError;
  if (!lookupFailed && lookup.resolution === "found" && lookup.describeReady) {
    return "ready";
  }
  if (lookup.reresolveRunning || (lookupFailed && !lookup.reresolveAttempted)) {
    return "reresolve";
  }
  if (lookupFailed) {
    return "error";
  }
  return "loading";
}

function initialWorkspaceView(
  phase: InitialWorkspaceOnboardingPhase,
  workspaceOpen: boolean,
): InitialWorkspaceOnboardingView {
  if (phase === "error") return "error";
  if (phase === "ready" || workspaceOpen) return "setup";
  return "loading";
}
