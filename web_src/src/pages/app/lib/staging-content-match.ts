import type { CanvasesCanvasVersion } from "@/api-client";
import { canvasesDescribeCanvasVersion } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

import { hasDraftVersusLiveConsoleDiff } from "../draftConsoleDiff";
import { hasDraftVersusLiveGraphDiff } from "../draftNodeDiff";
import { consoleSpecFromVersion, consoleSpecFromYaml, fetchCanvasVersionWithSpec, fetchRepositorySpecFileContent } from "./repository-spec-files";
import { dematerializeCanvasSpec } from "./workflow-spec-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "./workflow-spec-paths";

export function matchesCommittedCanvasSpec(
  committedVersion: CanvasesCanvasVersion | undefined,
  nextCanvasYaml: string,
): boolean {
  try {
    const nextSpec = dematerializeCanvasSpec(nextCanvasYaml);
    if (!committedVersion?.spec || !nextSpec) {
      return false;
    }

    return !hasDraftVersusLiveGraphDiff(committedVersion, { spec: nextSpec } as CanvasesCanvasVersion);
  } catch {
    return false;
  }
}

export async function matchesCommittedCanvasYaml(
  canvasId: string,
  versionId: string,
  nextCanvasYaml: string,
): Promise<boolean> {
  try {
    const version = await fetchCanvasVersionWithSpec(canvasId, versionId);
    return matchesCommittedCanvasSpec(version, nextCanvasYaml);
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
    const describeResponse = await canvasesDescribeCanvasVersion(
      withOrganizationHeader({
        path: { canvasId, versionId },
      }),
    );
    const committed = consoleSpecFromVersion(canvasId, describeResponse.data?.version);
    const next = consoleSpecFromYaml(nextConsoleYaml);
    if (!committed || !next) {
      const committedYaml = committed?.consoleYaml ?? "";
      return committedYaml === nextConsoleYaml;
    }

    return !hasDraftVersusLiveConsoleDiff(committed, next);
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
