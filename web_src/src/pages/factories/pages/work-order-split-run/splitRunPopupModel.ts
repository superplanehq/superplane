import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";

import { factoryAppConfigurePath, factoryAppSplitRunPath } from "../../lib/factoryPagePaths";
import {
  composeIntentDocument,
  INTENT_ARTIFACT_NAME,
  SPEC_ARTIFACT_NAME,
  type IntentDocument,
  parseIntentDocument,
} from "../../lib/intentDocument";
import {
  extractArtifactMarkdownBody,
  extractArtifactName,
  extractArtifactTitle,
  toArtifactDataRecord,
} from "../../lib/workOrderArtifact";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import type { SplitRunFixture, SplitRunPhase } from "./splitRunMocks";
import { isOriginTicketArtifact, type SplitRunSource } from "./splitRunSource";

export type SplitRunPopupTab = "description" | "log";

/** Task popup size. Wider than the default overlay so Summary and Plan have room. */
export const SPLIT_RUN_POPUP_DIALOG_CLASSNAME =
  "max-h-[min(62rem,calc(100vh-3rem))] h-[min(56rem,calc(100vh-3rem))] w-[min(90rem,calc(100vw-3rem))]";

const DESCRIPTION_NAMES = ["details.md", "description.md"];
const PLAN_NAMES = ["plan.md"];

export function defaultSplitRunPopupTab(fixture: SplitRunFixture): SplitRunPopupTab {
  if (fixture.openPhaseId) {
    return "log";
  }
  return fixture.footer.kind === "draft" || fixture.footer.kind === "done" ? "description" : "log";
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

/** Full-screen run page with the log in the sidebar. */
export function splitRunPhaseRunHref(args: {
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
  phase: SplitRunPhase;
}): string | undefined {
  const { organizationId, factoryKey, orderNumber, lineId, phase } = args;
  if (!organizationId || !factoryKey || !phase.appId) {
    return undefined;
  }
  return factoryAppSplitRunPath(organizationId, factoryKey, phase.appId, {
    from: "task",
    orderNumber,
    lineId,
    runId: phase.runId,
    canvas: phase.canvasKey ?? undefined,
  });
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

export function resolveSplitRunPopupPullRequests(args: {
  fixturePullRequests: FactoriesFactoryPullRequest[];
  livePullRequests?: FactoriesFactoryPullRequest[];
  useLive: boolean;
}): FactoriesFactoryPullRequest[] {
  if (args.useLive) {
    return args.livePullRequests ?? [];
  }
  return args.fixturePullRequests;
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

/** Unique pull requests from every phase stream, first occurrence wins. */
export function collectSplitRunPullRequests(fixture: SplitRunFixture): FactoriesFactoryPullRequest[] {
  const seen = new Set<string>();
  const pullRequests: FactoriesFactoryPullRequest[] = [];
  for (const phase of fixture.phases) {
    for (const line of phase.stream) {
      const pullRequest = line.pullRequest;
      if (!pullRequest) {
        continue;
      }
      const key = pullRequest.id ?? pullRequest.url ?? String(pullRequest.number ?? "");
      if (!key || seen.has(key)) {
        continue;
      }
      seen.add(key);
      pullRequests.push(pullRequest);
    }
  }
  return pullRequests;
}

export function splitRunDescriptionMarkdown(artifacts: FactoriesWorkOrderArtifact[]): string {
  return firstArtifactMarkdown(artifacts, DESCRIPTION_NAMES);
}

export function splitRunIntentMarkdown(artifacts: FactoriesWorkOrderArtifact[]): string {
  return firstArtifactMarkdown(artifacts, [SPEC_ARTIFACT_NAME, INTENT_ARTIFACT_NAME]);
}

export function splitRunIntentDocument(args: {
  artifacts: FactoriesWorkOrderArtifact[];
  description: string;
}): IntentDocument {
  const intent = splitRunIntentMarkdown(args.artifacts);
  if (intent) {
    return parseIntentDocument(intent);
  }
  return composeIntentDocument(args.description, firstArtifactMarkdown(args.artifacts, PLAN_NAMES));
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
      if (
        DESCRIPTION_NAMES.includes(artifactName(artifact)) ||
        artifactName(artifact) === INTENT_ARTIFACT_NAME ||
        artifactName(artifact) === SPEC_ARTIFACT_NAME
      ) {
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

function firstArtifactMarkdown(artifacts: FactoriesWorkOrderArtifact[], names: readonly string[]): string {
  for (const name of names) {
    const artifact = artifacts.find((entry) => artifactName(entry) === name);
    const body = extractArtifactMarkdownBody(toArtifactDataRecord(artifact?.data))?.trim();
    if (body) {
      return body;
    }
  }
  return "";
}

function artifactName(artifact: FactoriesWorkOrderArtifact): string {
  const data = toArtifactDataRecord(artifact.data);
  return extractArtifactName(data)?.trim() || extractArtifactTitle(data)?.trim() || "";
}
