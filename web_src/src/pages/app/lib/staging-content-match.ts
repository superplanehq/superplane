import type { CanvasesCanvasVersion } from "@/api-client";

import { hasDraftVersusLiveConsoleDiff } from "../draftConsoleDiff";
import { hasDraftVersusLiveGraphDiff } from "../draftNodeDiff";
import { consoleSpecFromYaml, fetchRepositorySpecFileContent } from "./repository-spec-files";
import { dematerializeCanvasMetadata, dematerializeCanvasSpec } from "./workflow-spec-files";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "./workflow-spec-paths";

function canvasYamlMetadataMatches(committedYaml: string, nextYaml: string): boolean {
  const committedMeta = dematerializeCanvasMetadata(committedYaml);
  const nextMeta = dematerializeCanvasMetadata(nextYaml);
  if (!committedMeta && !nextMeta) {
    return true;
  }
  if (!committedMeta || !nextMeta) {
    return false;
  }
  return (
    (committedMeta.name ?? "").trim() === (nextMeta.name ?? "").trim() &&
    (committedMeta.description ?? "").trim() === (nextMeta.description ?? "").trim()
  );
}

export async function matchesCommittedCanvasYaml(
  canvasId: string,
  versionId: string,
  nextCanvasYaml: string,
): Promise<boolean> {
  try {
    const committedYaml = await fetchRepositorySpecFileContent(canvasId, CANVAS_YAML_PATH, versionId, false);
    if (!canvasYamlMetadataMatches(committedYaml, nextCanvasYaml)) {
      return false;
    }

    const committedSpec = dematerializeCanvasSpec(committedYaml);
    const nextSpec = dematerializeCanvasSpec(nextCanvasYaml);
    if (!committedSpec || !nextSpec) {
      return committedYaml === nextCanvasYaml;
    }

    return !hasDraftVersusLiveGraphDiff(
      { spec: committedSpec } as CanvasesCanvasVersion,
      { spec: nextSpec } as CanvasesCanvasVersion,
    );
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
    const committedYaml = await fetchRepositorySpecFileContent(canvasId, CONSOLE_YAML_PATH, versionId, false);
    const committed = consoleSpecFromYaml(committedYaml);
    const next = consoleSpecFromYaml(nextConsoleYaml);
    if (!committed || !next) {
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
