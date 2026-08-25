import type { FactoriesWorkOrderArtifact } from "@/api-client";

import { extractArtifactMarkdownBody, toArtifactDataRecord } from "../../lib/workOrderArtifact";
import type { SplitRunFixture, SplitRunPhaseStatus } from "./splitRunMocks";

export type SplitRunPopupTab = "description" | "log";

const DESCRIPTION_NAMES = ["details.md", "description.md"];

const LOG_TAB_DOT: Record<SplitRunPhaseStatus, string> = {
  passed: "bg-[color:var(--status-completed-dot)]",
  running: "bg-[color:var(--status-running-dot)]",
  waiting: "bg-[color:var(--status-waiting-dot)]",
  failed: "bg-[color:var(--status-failed-dot)]",
  pending: "bg-[color:var(--status-draft-dot)]",
};

export function defaultSplitRunPopupTab(fixture: SplitRunFixture): SplitRunPopupTab {
  return fixture.footer.kind === "draft" ? "description" : "log";
}

export function splitRunLogTabDotClass(status: SplitRunPhaseStatus): string {
  return LOG_TAB_DOT[status];
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
