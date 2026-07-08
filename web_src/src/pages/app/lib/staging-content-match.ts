import { canvasesDescribeCanvasVersion, type CanvasesCanvasVersion } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { hasDraftVersusLiveConsoleDiff } from "../draftConsoleDiff";
import { hasDraftVersusLiveGraphDiff } from "../draftNodeDiff";
import {
  consoleSpecFromCanvasSpec,
  consoleSpecFromYaml,
  fetchRepositorySpecFileContent,
} from "./repository-spec-files";
import { dematerializeCanvasSpec } from "./workflow-spec-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "./workflow-spec-paths";

export function committedCanvasMatchesYaml(
  committedSpec: CanvasesCanvasVersion["spec"] | undefined,
  nextCanvasYaml: string,
): boolean {
  const nextSpec = dematerializeCanvasSpec(nextCanvasYaml);
  if (!committedSpec || !nextSpec) {
    return false;
  }

  return !hasDraftVersusLiveGraphDiff(
    { spec: committedSpec } as CanvasesCanvasVersion,
    { spec: nextSpec } as CanvasesCanvasVersion,
  );
}

export function committedConsoleMatchesYaml(
  canvasId: string,
  committedSpec: CanvasesCanvasVersion["spec"] | undefined,
  nextConsoleYaml: string,
): boolean {
  const committed = consoleSpecFromCanvasSpec(canvasId, committedSpec);
  const next = consoleSpecFromYaml(nextConsoleYaml);
  if (!next) {
    return false;
  }

  return !hasDraftVersusLiveConsoleDiff(committed, next);
}

async function loadCommittedVersionSpec(
  canvasId: string,
  versionId: string,
): Promise<CanvasesCanvasVersion["spec"] | undefined> {
  const response = await canvasesDescribeCanvasVersion(
    withOrganizationHeader({
      path: { canvasId, versionId },
    }),
  );
  return response.data?.version?.spec;
}

export async function matchesCommittedCanvasYaml(
  canvasId: string,
  versionId: string,
  nextCanvasYaml: string,
): Promise<boolean> {
  try {
    const committedSpec = await loadCommittedVersionSpec(canvasId, versionId);
    return committedCanvasMatchesYaml(committedSpec, nextCanvasYaml);
  } catch {
    return false;
  }
}

export async function matchesCommittedConsoleYaml(
  canvasId: string,
  versionId: string,
  nextConsoleYaml: string,
): Promise<boolean> {
  try {
    const committedSpec = await loadCommittedVersionSpec(canvasId, versionId);
    return committedConsoleMatchesYaml(canvasId, committedSpec, nextConsoleYaml);
  } catch {
    return false;
  }
}

export async function matchesCommittedRepositoryFileContent(
  canvasId: string,
  versionId: string,
  path: string,
  nextContent: string,
): Promise<boolean> {
  if (path === CANVAS_YAML_PATH) {
    return matchesCommittedCanvasYaml(canvasId, versionId, nextContent);
  }

  if (path === CONSOLE_YAML_PATH) {
    return matchesCommittedConsoleYaml(canvasId, versionId, nextContent);
  }

  try {
    const committedContent = await fetchRepositorySpecFileContent(canvasId, path, versionId, false);
    return committedContent === nextContent;
  } catch {
    return false;
  }
}
