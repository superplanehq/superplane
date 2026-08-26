import type { FactoriesWorkOrderArtifact } from "@/api-client";

import { factoryAppConfigurePath } from "../../lib/factoryPagePaths";
import { extractArtifactMarkdownBody, toArtifactDataRecord } from "../../lib/workOrderArtifact";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import type { SplitRunFixture, SplitRunPhase, SplitRunPhaseStatus } from "./splitRunMocks";
import { isOriginTicketArtifact, type SplitRunSource } from "./splitRunSource";

export type SplitRunPopupTab = "description" | "log";

/** Description uses a 3/2 reading-to-side split. */
export const SPLIT_RUN_PANE_GRID_CLASSNAME =
  "grid min-h-0 flex-1 grid-cols-1 md:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]";

const DESCRIPTION_NAMES = ["details.md", "description.md"];

const LOG_TAB_DOT: Record<SplitRunPhaseStatus, string> = {
  passed: "bg-[color:var(--status-completed-dot)]",
  running: "bg-[color:var(--status-running-dot)]",
  waiting: "bg-[color:var(--status-waiting-dot)]",
  failed: "bg-[color:var(--status-failed-dot)]",
  pending: "bg-[color:var(--status-draft-dot)]",
};

export function defaultSplitRunPopupTab(fixture: SplitRunFixture): SplitRunPopupTab {
  if (fixture.openPhaseId) {
    return "log";
  }
  return fixture.footer.kind === "draft" || fixture.footer.kind === "done" ? "description" : "log";
}

export function splitRunLogTabDotClass(status: SplitRunPhaseStatus): string {
  return LOG_TAB_DOT[status];
}

function phaseRun(phase: SplitRunPhase | undefined): { appId: string; runId: string } | undefined {
  const appId = phase?.appId;
  const runId = phase?.runId;
  if (!appId || !runId) {
    return undefined;
  }
  return { appId, runId };
}

/**
 * Open the automation run for the selected log phase. If that phase has
 * no run, use the latest phase run or the footer run.
 */
export function splitRunAutomationRunHref(args: {
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  fixture: SplitRunFixture;
  preferredPhaseId?: string | null;
}): string | null {
  const { organizationId, factoryKey, orderNumber, fixture, preferredPhaseId } = args;
  if (!organizationId || !factoryKey) {
    return null;
  }
  const preferred = phaseRun(fixture.phases.find((phase) => phase.id === preferredPhaseId));
  const current = phaseRun(fixture.phases.find((phase) => phase.id === fixture.currentPhaseId));
  const latest = fixture.phases.reduceRight<{ appId: string; runId: string } | undefined>(
    (found, phase) => found ?? phaseRun(phase),
    undefined,
  );
  const run = preferred ?? current ?? latest ?? fixture.footer.run;
  return getWorkOrderRunHref(organizationId, factoryKey, run?.appId, run?.runId, { orderNumber });
}

/** Configure URL of the automation that owns a log phase. */
export function splitRunPhaseAutomationHref(args: {
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  phase: SplitRunPhase;
}): string | undefined {
  const { organizationId, factoryKey, orderNumber, phase } = args;
  if (!organizationId || !factoryKey || !phase.appId) {
    return undefined;
  }
  return factoryAppConfigurePath(organizationId, factoryKey, phase.appId, { orderNumber });
}

export function resolveSplitRunPopupArtifacts(args: {
  fixtureArtifacts: FactoriesWorkOrderArtifact[];
  liveArtifacts?: FactoriesWorkOrderArtifact[];
  useLive: boolean;
}): FactoriesWorkOrderArtifact[] {
  if (args.useLive) {
    return args.liveArtifacts ?? [];
  }
  return args.fixtureArtifacts;
}

export function collectSplitRunArtifacts(fixture: SplitRunFixture): FactoriesWorkOrderArtifact[] {
  const seen = new Set<string>();
  const artifacts: FactoriesWorkOrderArtifact[] = [];
  for (const phase of fixture.phases) {
    for (const artifact of phase.artifacts) {
      const key = artifact.id ?? `${artifact.type}-${artifactName(artifact)}-${artifact.createdAt}`;
      if (seen.has(key)) {
        continue;
      }
      seen.add(key);
      artifacts.push(artifact);
    }
  }
  return artifacts;
}

export function splitRunDescriptionMarkdown(artifacts: FactoriesWorkOrderArtifact[]): string {
  for (const name of DESCRIPTION_NAMES) {
    const artifact = artifacts.find((entry) => artifactName(entry) === name);
    const body = extractArtifactMarkdownBody(toArtifactDataRecord(artifact?.data))?.trim();
    if (body) {
      return body;
    }
  }
  return "";
}

/** Live saves win. Storybook still prefers artifact markdown. */
export function splitRunSourceDescription(args: {
  workOrderDescription?: string;
  artifactDescription?: string;
  preferWorkOrder?: boolean;
}): string {
  const workOrder = args.workOrderDescription?.trim() ?? "";
  const artifact = args.artifactDescription?.trim() ?? "";
  if (args.preferWorkOrder) {
    return workOrder || artifact;
  }
  return artifact || workOrder;
}

/** Files and links that are not already the description body. Oldest first. */
export function splitRunLinkedArtifacts(
  artifacts: FactoriesWorkOrderArtifact[],
  source?: SplitRunSource,
): FactoriesWorkOrderArtifact[] {
  return artifacts
    .filter((artifact) => {
      if (DESCRIPTION_NAMES.includes(artifactName(artifact))) {
        return false;
      }
      return !isOriginTicketArtifact(artifact, source);
    })
    .sort(compareArtifactsByCreatedAt);
}

function compareArtifactsByCreatedAt(left: FactoriesWorkOrderArtifact, right: FactoriesWorkOrderArtifact): number {
  return artifactCreatedAtMs(left) - artifactCreatedAtMs(right);
}

function artifactCreatedAtMs(artifact: FactoriesWorkOrderArtifact): number {
  const parsed = Date.parse(artifact.createdAt ?? "");
  return Number.isFinite(parsed) ? parsed : Number.POSITIVE_INFINITY;
}

function artifactName(artifact: FactoriesWorkOrderArtifact): string {
  const data = toArtifactDataRecord(artifact.data);
  if (typeof data?.name === "string" && data.name.trim()) {
    return data.name.trim();
  }
  if (typeof data?.title === "string" && data.title.trim()) {
    return data.title.trim();
  }
  return "";
}
