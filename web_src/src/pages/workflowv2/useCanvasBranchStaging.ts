import type { CanvasesCanvas, CanvasesCanvasDashboard } from "@/api-client";
import {
  CANVAS_YAML_PATH,
  CONSOLE_YAML_PATH,
  clearStaging,
  getStaging,
  hasStagingFiles,
  putStaging,
  type CanvasStagingRecord,
} from "@/lib/canvas-staging";
import { useCommitCanvasRepositoryFiles } from "@/hooks/useCanvasData";
import {
  fetchCanvasRepositoryFileContent,
  encodeRepositoryFileContent,
} from "@/pages/workflowv2/lib/canvas-repository-files";
import { buildCanvasYamlFromWorkflow, parseCanvasYamlToSpec } from "@/pages/workflowv2/lib/canvas-yaml-staging";
import * as yaml from "js-yaml";
import { dashboardToYaml, parseDashboardYaml } from "@/pages/workflowv2/dashboard/dashboardYaml";
import type { DashboardLayoutItem, DashboardPanel } from "@/hooks/useCanvasData";
import debounce from "lodash.debounce";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

type BranchHead = {
  branch: string;
  headSha: string;
};

function stableStringify(value: unknown): string {
  if (Array.isArray(value)) {
    return `[${value.map(stableStringify).join(",")}]`;
  }
  if (value && typeof value === "object") {
    const entries = Object.entries(value as Record<string, unknown>)
      .filter(([, v]) => v !== undefined)
      .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0));
    return `{${entries.map(([k, v]) => `${JSON.stringify(k)}:${stableStringify(v)}`).join(",")}}`;
  }
  return JSON.stringify(value ?? null);
}

// Produces a canonical representation of a staged file so that purely textual
// differences (key ordering, whitespace, serializer defaults) between the
// backend-generated committed content and the UI-serialized staged content do
// not register as real changes.
function canonicalizeFile(path: string, text: string): string {
  if (!text || !text.trim()) {
    return "";
  }

  if (path === CANVAS_YAML_PATH) {
    const spec = parseCanvasYamlToSpec(text);
    if (!spec) {
      return text.trim();
    }

    let metadata: NonNullable<CanvasesCanvas["metadata"]> = {};
    try {
      const doc = yaml.load(text) as { metadata?: CanvasesCanvas["metadata"] } | undefined;
      metadata = doc?.metadata ?? {};
    } catch {
      metadata = {};
    }

    // Compare structurally with defaults applied so that serializer defaults
    // (e.g. an explicit empty description) and key ordering normalize away while
    // real metadata/spec edits (e.g. renames) are still detected.
    return stableStringify({
      metadata: {
        id: metadata.id || "",
        name: metadata.name || "Canvas",
        description: metadata.description || "",
        isTemplate: metadata.isTemplate ?? false,
      },
      spec: {
        nodes: spec.nodes ?? [],
        edges: spec.edges ?? [],
        changeManagement: spec.changeManagement,
      },
    });
  }

  if (path === CONSOLE_YAML_PATH) {
    try {
      const doc = yaml.load(text) as { spec?: { panels?: unknown; layout?: unknown } } | undefined;
      return stableStringify({
        panels: doc?.spec?.panels ?? [],
        layout: doc?.spec?.layout ?? [],
      });
    } catch {
      return text.trim();
    }
  }

  return text.trim();
}

export function stagingDiffersFromBaseline(
  record: CanvasStagingRecord | null,
  baseline: Record<string, string>,
): boolean {
  if (!record) {
    return false;
  }

  return Object.entries(record.files).some(
    ([path, content]) => canonicalizeFile(path, content) !== canonicalizeFile(path, baseline[path] ?? ""),
  );
}

// Whether a single staged file differs from the branch baseline. Used to drive
// per-surface change indicators (e.g. the Canvas vs Console tab dots) so a staged
// canvas edit does not light up the console indicator and vice versa.
export function stagingFileDiffersFromBaseline(
  record: CanvasStagingRecord | null,
  baseline: Record<string, string>,
  path: string,
): boolean {
  if (!record || !(path in record.files)) {
    return false;
  }

  return canonicalizeFile(path, record.files[path]) !== canonicalizeFile(path, baseline[path] ?? "");
}

type UseCanvasBranchStagingOptions = {
  canvasId: string | undefined;
  activeBranch: string | null;
  headSha?: string;
  enabled: boolean;
};

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

function dashboardPanelsToYaml(
  canvasId: string,
  canvasName: string | undefined,
  panels: DashboardPanel[],
  layout: DashboardLayoutItem[],
): string {
  return dashboardToYaml({ panels, layout, canvasId, canvasName });
}

export function useCanvasBranchStaging({ canvasId, activeBranch, headSha, enabled }: UseCanvasBranchStagingOptions) {
  const commitFilesMutation = useCommitCanvasRepositoryFiles(canvasId ?? "");
  const [stagingRecord, setStagingRecord] = useState<CanvasStagingRecord | null>(null);
  const [baselineFiles, setBaselineFiles] = useState<Record<string, string>>({});
  const [branchDashboard, setBranchDashboard] = useState<CanvasesCanvasDashboard | null>(null);
  const [branchCanvasSpec, setBranchCanvasSpec] = useState<CanvasesCanvas["spec"] | null>(null);
  const [isLoadingBranchContent, setIsLoadingBranchContent] = useState(false);
  const stagingRecordRef = useRef(stagingRecord);
  stagingRecordRef.current = stagingRecord;
  const branchHeadRef = useRef<BranchHead | null>(null);

  const hasStagingChanges = useMemo(
    () => stagingDiffersFromBaseline(stagingRecord, baselineFiles),
    [stagingRecord, baselineFiles],
  );
  const hasCanvasStagingChanges = useMemo(
    () => stagingFileDiffersFromBaseline(stagingRecord, baselineFiles, CANVAS_YAML_PATH),
    [stagingRecord, baselineFiles],
  );
  const hasConsoleStagingChanges = useMemo(
    () => stagingFileDiffersFromBaseline(stagingRecord, baselineFiles, CONSOLE_YAML_PATH),
    [stagingRecord, baselineFiles],
  );

  const persistStaging = useCallback(
    async (files: Record<string, string>, baseHeadSha: string) => {
      if (!canvasId || !activeBranch) {
        return;
      }

      const nextRecord: CanvasStagingRecord = {
        canvasId,
        branch: activeBranch,
        baseHeadSha,
        files,
        updatedAt: Date.now(),
      };

      await putStaging(nextRecord);
      setStagingRecord(nextRecord);
    },
    [activeBranch, canvasId],
  );

  const loadBranchContent = useCallback(async () => {
    if (!canvasId || !activeBranch || !enabled) {
      setStagingRecord(null);
      setBaselineFiles({});
      setBranchDashboard(null);
      setBranchCanvasSpec(null);
      return;
    }

    setIsLoadingBranchContent(true);
    try {
      const currentHeadSha = headSha || "";
      branchHeadRef.current = { branch: activeBranch, headSha: currentHeadSha };

      const [canvasYaml, consoleYaml] = await Promise.all([
        fetchCanvasRepositoryFileContent(canvasId, CANVAS_YAML_PATH, activeBranch).catch(() => ""),
        fetchCanvasRepositoryFileContent(canvasId, CONSOLE_YAML_PATH, activeBranch).catch(() => ""),
      ]);

      setBaselineFiles({
        [CANVAS_YAML_PATH]: canvasYaml,
        [CONSOLE_YAML_PATH]: consoleYaml,
      });

      const existingStaging = await getStaging(canvasId, activeBranch);
      if (existingStaging && existingStaging.baseHeadSha === currentHeadSha && hasStagingFiles(existingStaging)) {
        setStagingRecord(existingStaging);
        const stagedCanvasYaml = existingStaging.files[CANVAS_YAML_PATH];
        const stagedConsoleYaml = existingStaging.files[CONSOLE_YAML_PATH];
        setBranchCanvasSpec(stagedCanvasYaml ? parseCanvasYamlToSpec(stagedCanvasYaml) : null);
        setBranchDashboard(stagedConsoleYaml ? dashboardFromYamlText(stagedConsoleYaml) : null);
        return;
      }

      setStagingRecord(null);
      setBranchCanvasSpec(canvasYaml ? parseCanvasYamlToSpec(canvasYaml) : null);
      setBranchDashboard(consoleYaml ? dashboardFromYamlText(consoleYaml) : null);
    } finally {
      setIsLoadingBranchContent(false);
    }
  }, [activeBranch, canvasId, enabled, headSha]);

  useEffect(() => {
    void loadBranchContent();
  }, [loadBranchContent]);

  const stageFilesDebounced = useMemo(
    () =>
      debounce(async (files: Record<string, string>, baseHeadSha: string) => {
        await persistStaging(files, baseHeadSha);
      }, 500),
    [persistStaging],
  );

  useEffect(() => {
    return () => {
      stageFilesDebounced.cancel();
    };
  }, [stageFilesDebounced]);

  const setLocalBranchCanvasSpec = useCallback((spec: CanvasesCanvas["spec"] | null | undefined) => {
    setBranchCanvasSpec(spec ?? null);
  }, []);

  const stageCanvasWorkflow = useCallback(
    (workflow: CanvasesCanvas, canvasName?: string) => {
      if (!canvasId || !activeBranch) {
        return;
      }

      const baseHeadSha = branchHeadRef.current?.headSha || headSha || "";
      const currentFiles = { ...(stagingRecordRef.current?.files ?? {}) };
      currentFiles[CANVAS_YAML_PATH] = buildCanvasYamlFromWorkflow(workflow);

      if (branchDashboard) {
        const panels = (branchDashboard.panels ?? []).map((panel) => ({
          id: panel.id ?? "",
          type: panel.type ?? "markdown",
          content: (panel.content as Record<string, unknown>) ?? {},
        }));
        const layout = (branchDashboard.layout ?? []).map((item) => ({
          i: item.i ?? "",
          x: item.x ?? 0,
          y: item.y ?? 0,
          w: item.w ?? 12,
          h: item.h ?? 6,
          ...(item.minW !== undefined ? { minW: item.minW } : {}),
          ...(item.minH !== undefined ? { minH: item.minH } : {}),
        }));
        currentFiles[CONSOLE_YAML_PATH] = dashboardPanelsToYaml(canvasId, canvasName, panels, layout);
      }

      void stageFilesDebounced(currentFiles, baseHeadSha);
      setStagingRecord({
        canvasId,
        branch: activeBranch,
        baseHeadSha,
        files: currentFiles,
        updatedAt: Date.now(),
      });
      setBranchCanvasSpec(workflow.spec ?? null);
    },
    [activeBranch, branchDashboard, canvasId, headSha, stageFilesDebounced],
  );

  const stageConsoleDashboard = useCallback(
    (input: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] }, canvasName?: string) => {
      if (!canvasId || !activeBranch) {
        return;
      }

      const baseHeadSha = branchHeadRef.current?.headSha || headSha || "";
      const currentFiles = { ...(stagingRecordRef.current?.files ?? {}) };
      currentFiles[CONSOLE_YAML_PATH] = dashboardPanelsToYaml(canvasId, canvasName, input.panels, input.layout);

      void stageFilesDebounced(currentFiles, baseHeadSha);
      setStagingRecord({
        canvasId,
        branch: activeBranch,
        baseHeadSha,
        files: currentFiles,
        updatedAt: Date.now(),
      });
      setBranchDashboard({
        panels: input.panels.map((panel) => ({
          id: panel.id,
          type: panel.type,
          content: panel.content,
        })),
        layout: input.layout.map((item) => ({
          i: item.i,
          x: item.x,
          y: item.y,
          w: item.w,
          h: item.h,
          ...(item.minW !== undefined ? { minW: item.minW } : {}),
          ...(item.minH !== undefined ? { minH: item.minH } : {}),
        })),
      });
    },
    [activeBranch, canvasId, headSha, stageFilesDebounced],
  );

  const discardStaging = useCallback(async () => {
    if (!canvasId || !activeBranch) {
      return;
    }

    stageFilesDebounced.cancel();
    await clearStaging(canvasId, activeBranch);
    setStagingRecord(null);
    await loadBranchContent();
  }, [activeBranch, canvasId, loadBranchContent, stageFilesDebounced]);

  // Flush any pending debounced write so the latest staged content is persisted
  // to IndexedDB before leaving the branch. Staging is keyed per branch, so it is
  // kept and reapplied when the user returns to this draft.
  const flushStaging = useCallback(() => {
    stageFilesDebounced.flush();
  }, [stageFilesDebounced]);

  const commitStaging = useCallback(
    async (message = "Update canvas files") => {
      if (!canvasId || !activeBranch || !stagingRecordRef.current) {
        return null;
      }

      stageFilesDebounced.flush();

      const record = stagingRecordRef.current;
      if (!hasStagingFiles(record)) {
        return null;
      }

      const response = await commitFilesMutation.mutateAsync({
        message,
        branch: activeBranch,
        expectedHeadSha: record.baseHeadSha || headSha,
        operations: Object.entries(record.files).map(([path, fileContent]) => ({
          path,
          content: encodeRepositoryFileContent(fileContent),
        })),
      });

      await clearStaging(canvasId, activeBranch);
      setStagingRecord(null);
      await loadBranchContent();
      return response;
    },
    [activeBranch, canvasId, commitFilesMutation, headSha, loadBranchContent, stageFilesDebounced],
  );

  const updateConsoleMutation = useMemo(
    () => ({
      mutate: (input: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] }) => {
        stageConsoleDashboard(input);
      },
      mutateAsync: async (input: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] }) => {
        stageConsoleDashboard(input);
        return branchDashboard;
      },
      isPending: false,
    }),
    [branchDashboard, stageConsoleDashboard],
  );

  const branchHeadCanvasSpec = useMemo(() => {
    const yaml = baselineFiles[CANVAS_YAML_PATH];
    return yaml ? parseCanvasYamlToSpec(yaml) : null;
  }, [baselineFiles]);

  const branchHeadDashboard = useMemo(() => {
    const yaml = baselineFiles[CONSOLE_YAML_PATH];
    return yaml ? dashboardFromYamlText(yaml) : null;
  }, [baselineFiles]);

  return {
    stagingRecord,
    hasStagingChanges,
    hasCanvasStagingChanges,
    hasConsoleStagingChanges,
    branchCanvasSpec,
    branchDashboard,
    branchHeadCanvasSpec,
    branchHeadDashboard,
    isLoadingBranchContent,
    stageCanvasWorkflow,
    setLocalBranchCanvasSpec,
    stageConsoleDashboard,
    discardStaging,
    flushStaging,
    commitStaging,
    reloadBranchContent: loadBranchContent,
    updateConsoleMutation,
    setBranchDashboard,
  };
}

export type CanvasBranchStagingState = ReturnType<typeof useCanvasBranchStaging>;
