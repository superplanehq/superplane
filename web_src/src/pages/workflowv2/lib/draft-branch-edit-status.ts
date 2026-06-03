import type { CanvasesCanvasDashboard, CanvasesCanvasVersion } from "@/api-client";
import type { DraftBranchEditStatus } from "@/components/CanvasToolSidebar/DraftBranchRow";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH, getStaging } from "@/lib/canvas-staging";
import { parseDashboardYaml } from "@/pages/workflowv2/dashboard/dashboardYaml";
import { hasDraftVersusLiveConsoleDiff } from "@/pages/workflowv2/draftConsoleDiff";
import { hasDraftVersusLiveGraphDiff } from "@/pages/workflowv2/draftNodeDiff";
import { fetchCanvasRepositoryFileContent } from "@/pages/workflowv2/lib/canvas-repository-files";
import { parseCanvasYamlToSpec } from "@/pages/workflowv2/lib/canvas-yaml-staging";
import { stagingFileDiffersFromBaseline } from "@/pages/workflowv2/useCanvasBranchStaging";
import type { DraftChangeIndicators } from "@/pages/workflowv2/lib/version-action-state";
import { getDraftChangeIndicators } from "@/pages/workflowv2/lib/version-action-state";

function dashboardFromYamlText(text: string): CanvasesCanvasDashboard | null {
  const parsed = parseDashboardYaml(text);
  if (!parsed.ok) {
    return null;
  }

  return {
    panels: parsed.data.spec.panels.map((panel) => ({
      id: panel.id,
      type: panel.type,
      content: panel.content,
    })),
    layout: parsed.data.spec.layout.map((item) => ({
      i: item.i,
      x: item.x,
      y: item.y,
      w: item.w,
      h: item.h,
      ...(item.minW !== undefined ? { minW: item.minW } : {}),
      ...(item.minH !== undefined ? { minH: item.minH } : {}),
    })),
  };
}

export type DraftBranchChangeDetail = {
  editStatus?: DraftBranchEditStatus;
  hasUncommittedCanvas: boolean;
  hasUncommittedConsole: boolean;
  hasCommittedCanvasVersusLive: boolean;
  hasCommittedConsoleVersusLive: boolean;
};

function branchCommittedDiffVersusLive(
  liveCanvasVersion: CanvasesCanvasVersion | undefined,
  liveDashboard: CanvasesCanvasDashboard | null | undefined,
  canvasYaml: string,
  consoleYaml: string,
): { hasCommittedCanvasVersusLive: boolean; hasCommittedConsoleVersusLive: boolean } {
  const headSpec = canvasYaml ? parseCanvasYamlToSpec(canvasYaml) : null;
  const hasCommittedCanvasVersusLive =
    !!headSpec && hasDraftVersusLiveGraphDiff(liveCanvasVersion, { spec: headSpec } as CanvasesCanvasVersion);
  const headDashboard = consoleYaml ? dashboardFromYamlText(consoleYaml) : null;
  const hasCommittedConsoleVersusLive = hasDraftVersusLiveConsoleDiff(liveDashboard, headDashboard);

  return { hasCommittedCanvasVersusLive, hasCommittedConsoleVersusLive };
}

export async function computeDraftBranchChangeDetail(
  canvasId: string,
  branchName: string,
  tipSha: string | undefined,
  liveCanvasVersion: CanvasesCanvasVersion | undefined,
  liveDashboard: CanvasesCanvasDashboard | null | undefined,
): Promise<DraftBranchChangeDetail> {
  const [canvasYaml, consoleYaml, staging] = await Promise.all([
    fetchCanvasRepositoryFileContent(canvasId, CANVAS_YAML_PATH, branchName).catch(() => ""),
    fetchCanvasRepositoryFileContent(canvasId, CONSOLE_YAML_PATH, branchName).catch(() => ""),
    getStaging(canvasId, branchName),
  ]);

  const baselineFiles = {
    [CANVAS_YAML_PATH]: canvasYaml,
    [CONSOLE_YAML_PATH]: consoleYaml,
  };

  const stagingMatchesTip = !!staging && (!tipSha || staging.baseHeadSha === tipSha);
  const hasUncommittedCanvas =
    stagingMatchesTip && stagingFileDiffersFromBaseline(staging, baselineFiles, CANVAS_YAML_PATH);
  const hasUncommittedConsole =
    stagingMatchesTip && stagingFileDiffersFromBaseline(staging, baselineFiles, CONSOLE_YAML_PATH);
  const { hasCommittedCanvasVersusLive, hasCommittedConsoleVersusLive } = branchCommittedDiffVersusLive(
    liveCanvasVersion,
    liveDashboard,
    canvasYaml,
    consoleYaml,
  );

  if (hasUncommittedCanvas || hasUncommittedConsole) {
    return {
      editStatus: "uncommitted",
      hasUncommittedCanvas,
      hasUncommittedConsole,
      hasCommittedCanvasVersusLive,
      hasCommittedConsoleVersusLive,
    };
  }

  if (hasCommittedCanvasVersusLive || hasCommittedConsoleVersusLive) {
    return {
      editStatus: "ready",
      hasUncommittedCanvas: false,
      hasUncommittedConsole: false,
      hasCommittedCanvasVersusLive,
      hasCommittedConsoleVersusLive,
    };
  }

  return {
    editStatus: "no-changes",
    hasUncommittedCanvas: false,
    hasUncommittedConsole: false,
    hasCommittedCanvasVersusLive: false,
    hasCommittedConsoleVersusLive: false,
  };
}

export function aggregateDraftTabIndicators(
  detailsByBranch: Record<string, DraftBranchChangeDetail>,
): DraftChangeIndicators {
  const details = Object.values(detailsByBranch);
  const hasUncommittedCanvas = details.some((detail) => detail.hasUncommittedCanvas);
  const hasUncommittedConsole = details.some((detail) => detail.hasUncommittedConsole);
  const hasCommittedCanvas = details.some((detail) => detail.hasCommittedCanvasVersusLive);
  const hasCommittedConsole = details.some((detail) => detail.hasCommittedConsoleVersusLive);

  return getDraftChangeIndicators({
    suppressUnpublishedDraftDiscard: false,
    hasLatestDraftVersion: details.length > 0,
    hasDraftGraphDiffVersusLive: hasCommittedCanvas,
    hasDraftConsoleDiffVersusLive: hasCommittedConsole,
    hasDraftDiffVersusLive: hasCommittedCanvas || hasCommittedConsole,
    hasCanvasStagingChanges: hasUncommittedCanvas,
    hasConsoleStagingChanges: hasUncommittedConsole,
  });
}

export function detailToEditStatus(detail: DraftBranchChangeDetail): DraftBranchEditStatus {
  return detail.editStatus ?? "no-changes";
}

export function resolveDraftBranchEditStatus(
  hasUncommitted: boolean,
  readyToPublish: boolean,
): DraftBranchEditStatus {
  if (hasUncommitted) {
    return "uncommitted";
  }

  if (readyToPublish) {
    return "ready";
  }

  return "no-changes";
}

export function resolveActiveBranchChangeDetail(
  hasUncommittedCanvas: boolean,
  hasUncommittedConsole: boolean,
  hasCommittedCanvasVersusLive: boolean,
  hasCommittedConsoleVersusLive: boolean,
): DraftBranchChangeDetail {
  const hasUncommitted = hasUncommittedCanvas || hasUncommittedConsole;
  const hasCommitted = hasCommittedCanvasVersusLive || hasCommittedConsoleVersusLive;

  return {
    editStatus: hasUncommitted ? "uncommitted" : hasCommitted ? "ready" : "no-changes",
    hasUncommittedCanvas,
    hasUncommittedConsole,
    hasCommittedCanvasVersusLive,
    hasCommittedConsoleVersusLive,
  };
}
