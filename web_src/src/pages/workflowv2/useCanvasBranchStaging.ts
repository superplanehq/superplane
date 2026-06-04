import type { CanvasesCanvas, CanvasesCanvasDashboard } from "@/api-client";
import {
  CANVAS_YAML_PATH,
  CONSOLE_YAML_PATH,
  clearStaging,
  getStaging,
  hasStagingFiles,
  putStaging,
  stagingMatchesBranchHead,
  type CanvasStagingRecord,
} from "@/lib/canvas-staging";
import { useCommitCanvasRepositoryFiles } from "@/hooks/useCanvasData";
import {
  fetchCanvasRepositoryFileContent,
  encodeRepositoryFileContent,
} from "@/pages/workflowv2/lib/canvas-repository-files";
import { buildCanvasYamlFromWorkflow, parseCanvasYamlToSpec } from "@/pages/workflowv2/lib/canvas-yaml-staging";
import { resolveStagedBranchYaml } from "@/pages/workflowv2/lib/resolve-staged-branch-yaml";
import { branchHasCommittedRepositoryFilesVersusLive } from "@/pages/workflowv2/lib/repository-files-branch-diff";
import { hasStagedRepositoryFileChanges } from "@/pages/workflowv2/lib/workflow-files-staging";
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

  if ((record.deletedPaths ?? []).some((path) => path in baseline)) {
    return true;
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
  const [hasCommittedRepositoryFilesVersusLive, setHasCommittedRepositoryFilesVersusLive] = useState(false);
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
  const hasRepositoryFilesStagingChanges = useMemo(
    () => hasStagedRepositoryFileChanges(stagingRecord),
    [stagingRecord],
  );

  // Staging writes are merged per-key against the persisted IndexedDB record so a
  // write only ever touches the keys it intends to change. A blind full-record
  // overwrite (built from a possibly-stale in-memory snapshot) used to drop other
  // staged files — e.g. the canvas auto-save rewriting canvas.yaml would erase a
  // staged README.md when its in-memory ref had not loaded yet.
  const pendingMutationRef = useRef<{
    setFiles: Record<string, string>;
    removeFiles: Set<string>;
    deletedPaths?: string[];
  } | null>(null);

  const flushPendingMutation = useCallback(async () => {
    const mutation = pendingMutationRef.current;
    if (!mutation || !canvasId || !activeBranch) {
      return;
    }
    pendingMutationRef.current = null;

    const stored = await getStaging(canvasId, activeBranch);
    const baseHeadSha = branchHeadRef.current?.headSha || headSha || stored?.baseHeadSha || "";
    if (!baseHeadSha) {
      // Re-queue until the branch head is known so we never persist an orphaned record.
      pendingMutationRef.current = mutation;
      return;
    }

    const files = { ...(stored?.files ?? {}) };
    for (const [path, content] of Object.entries(mutation.setFiles)) {
      files[path] = content;
    }
    for (const path of mutation.removeFiles) {
      delete files[path];
    }
    const deletedPaths = mutation.deletedPaths ?? stored?.deletedPaths ?? [];

    const record: CanvasStagingRecord = {
      canvasId,
      branch: activeBranch,
      baseHeadSha,
      files,
      deletedPaths,
      updatedAt: Date.now(),
    };

    if (hasStagingFiles(record)) {
      await putStaging(record);
      setStagingRecord(record);
      return;
    }

    await clearStaging(canvasId, activeBranch);
    setStagingRecord(null);
  }, [activeBranch, canvasId, headSha]);

  const loadBranchContent = useCallback(async () => {
    if (!canvasId || !activeBranch || !enabled) {
      setStagingRecord(null);
      setBaselineFiles({});
      setBranchDashboard(null);
      setBranchCanvasSpec(null);
      setHasCommittedRepositoryFilesVersusLive(false);
      return;
    }

    setIsLoadingBranchContent(true);
    try {
      const currentHeadSha = headSha || "";
      branchHeadRef.current = { branch: activeBranch, headSha: currentHeadSha };

      const [canvasYaml, consoleYaml, committedFilesVersusLive] = await Promise.all([
        fetchCanvasRepositoryFileContent(canvasId, CANVAS_YAML_PATH, activeBranch).catch(() => ""),
        fetchCanvasRepositoryFileContent(canvasId, CONSOLE_YAML_PATH, activeBranch).catch(() => ""),
        branchHasCommittedRepositoryFilesVersusLive(canvasId, activeBranch),
      ]);
      setHasCommittedRepositoryFilesVersusLive(committedFilesVersusLive);

      setBaselineFiles({
        [CANVAS_YAML_PATH]: canvasYaml,
        [CONSOLE_YAML_PATH]: consoleYaml,
      });

      const existingStaging = await getStaging(canvasId, activeBranch);
      if (existingStaging && stagingMatchesBranchHead(existingStaging, currentHeadSha)) {
        let staging = existingStaging;
        if (currentHeadSha && staging.baseHeadSha !== currentHeadSha) {
          staging = { ...staging, baseHeadSha: currentHeadSha, updatedAt: Date.now() };
          await putStaging(staging);
        }

        setStagingRecord(staging);
        const { canvasYaml: stagedCanvasYaml, consoleYaml: stagedConsoleYaml } = resolveStagedBranchYaml(
          staging,
          canvasYaml,
          consoleYaml,
        );
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

  useEffect(() => {
    if (!canvasId || !activeBranch || !headSha) {
      return;
    }

    const record = stagingRecordRef.current;
    if (!record || record.baseHeadSha === headSha) {
      return;
    }

    if (!record.baseHeadSha) {
      const repaired: CanvasStagingRecord = { ...record, baseHeadSha: headSha, updatedAt: Date.now() };
      branchHeadRef.current = { branch: activeBranch, headSha };
      void putStaging(repaired);
      setStagingRecord(repaired);
    }
  }, [activeBranch, canvasId, headSha]);

  const scheduleFlush = useMemo(
    () =>
      debounce(() => {
        void flushPendingMutation();
      }, 500),
    [flushPendingMutation],
  );

  const queueStagingMutation = useCallback(
    (delta: { setFiles?: Record<string, string>; removeFiles?: string[]; deletedPaths?: string[] }) => {
      const pending = pendingMutationRef.current ?? {
        setFiles: {},
        removeFiles: new Set<string>(),
      };

      for (const [path, content] of Object.entries(delta.setFiles ?? {})) {
        pending.setFiles[path] = content;
        pending.removeFiles.delete(path);
      }
      for (const path of delta.removeFiles ?? []) {
        pending.removeFiles.add(path);
        delete pending.setFiles[path];
      }
      if (delta.deletedPaths) {
        pending.deletedPaths = delta.deletedPaths;
      }

      pendingMutationRef.current = pending;
      scheduleFlush();
    },
    [scheduleFlush],
  );

  // Applies a staging delta optimistically to the in-memory record (for snappy UI
  // and accurate change indicators) and queues a merge-based persist to IndexedDB.
  const commitStagingDelta = useCallback(
    (delta: { setFiles?: Record<string, string>; removeFiles?: string[]; deletedPaths?: string[] }) => {
      if (!canvasId || !activeBranch) {
        return;
      }

      setStagingRecord((prev) => {
        const files = { ...(prev?.files ?? {}) };
        for (const [path, content] of Object.entries(delta.setFiles ?? {})) {
          files[path] = content;
        }
        for (const path of delta.removeFiles ?? []) {
          delete files[path];
        }
        const deletedPaths = delta.deletedPaths ?? prev?.deletedPaths ?? [];
        const baseHeadSha = prev?.baseHeadSha || branchHeadRef.current?.headSha || headSha || "";
        const next: CanvasStagingRecord = {
          canvasId,
          branch: activeBranch,
          baseHeadSha,
          files,
          deletedPaths,
          updatedAt: Date.now(),
        };
        return hasStagingFiles(next) ? next : null;
      });

      queueStagingMutation(delta);
    },
    [activeBranch, canvasId, headSha, queueStagingMutation],
  );

  const stageRepositoryFile = useCallback(
    (path: string, content: string) => {
      const currentDeleted = (stagingRecordRef.current?.deletedPaths ?? []).filter(
        (deletedPath) => deletedPath !== path,
      );
      commitStagingDelta({ setFiles: { [path]: content }, deletedPaths: currentDeleted });
    },
    [commitStagingDelta],
  );

  const stageRepositoryFileDelete = useCallback(
    (path: string, existsInRepository: boolean) => {
      if (!existsInRepository) {
        const currentDeleted = (stagingRecordRef.current?.deletedPaths ?? []).filter(
          (deletedPath) => deletedPath !== path,
        );
        commitStagingDelta({ removeFiles: [path], deletedPaths: currentDeleted });
        return;
      }

      const currentDeleted = [...(stagingRecordRef.current?.deletedPaths ?? [])];
      if (!currentDeleted.includes(path)) {
        currentDeleted.push(path);
      }
      commitStagingDelta({ removeFiles: [path], deletedPaths: currentDeleted });
    },
    [commitStagingDelta],
  );

  const unstageRepositoryFile = useCallback(
    (path: string) => {
      const currentDeleted = (stagingRecordRef.current?.deletedPaths ?? []).filter(
        (deletedPath) => deletedPath !== path,
      );
      commitStagingDelta({ removeFiles: [path], deletedPaths: currentDeleted });
    },
    [commitStagingDelta],
  );

  useEffect(() => {
    return () => {
      scheduleFlush.cancel();
    };
  }, [scheduleFlush]);

  const setLocalBranchCanvasSpec = useCallback((spec: CanvasesCanvas["spec"] | null | undefined) => {
    setBranchCanvasSpec(spec ?? null);
  }, []);

  const stageCanvasWorkflow = useCallback(
    (workflow: CanvasesCanvas, canvasName?: string) => {
      if (!canvasId || !activeBranch) {
        return;
      }

      const setFiles: Record<string, string> = {
        [CANVAS_YAML_PATH]: buildCanvasYamlFromWorkflow(workflow),
      };

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
        setFiles[CONSOLE_YAML_PATH] = dashboardPanelsToYaml(canvasId, canvasName, panels, layout);
      }

      commitStagingDelta({ setFiles });
      setBranchCanvasSpec(workflow.spec ?? null);
    },
    [activeBranch, branchDashboard, canvasId, commitStagingDelta],
  );

  const stageConsoleDashboard = useCallback(
    (input: { panels: DashboardPanel[]; layout: DashboardLayoutItem[] }, canvasName?: string) => {
      if (!canvasId || !activeBranch) {
        return;
      }

      commitStagingDelta({
        setFiles: { [CONSOLE_YAML_PATH]: dashboardPanelsToYaml(canvasId, canvasName, input.panels, input.layout) },
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
    [activeBranch, canvasId, commitStagingDelta],
  );

  const discardStaging = useCallback(async () => {
    if (!canvasId || !activeBranch) {
      return;
    }

    scheduleFlush.cancel();
    pendingMutationRef.current = null;
    await clearStaging(canvasId, activeBranch);
    setStagingRecord(null);
    await loadBranchContent();
  }, [activeBranch, canvasId, loadBranchContent, scheduleFlush]);

  // Flush any pending debounced write so the latest staged content is persisted
  // to IndexedDB before leaving the branch. Staging is keyed per branch, so it is
  // kept and reapplied when the user returns to this draft.
  const flushStaging = useCallback(async () => {
    scheduleFlush.cancel();
    await flushPendingMutation();
  }, [flushPendingMutation, scheduleFlush]);

  const commitStaging = useCallback(
    async (message = "Update canvas files") => {
      if (!canvasId || !activeBranch) {
        return null;
      }

      scheduleFlush.cancel();
      await flushPendingMutation();

      // Read the authoritative persisted record so the commit reflects every staged
      // file, not a possibly-stale in-memory snapshot.
      const record = (await getStaging(canvasId, activeBranch)) ?? stagingRecordRef.current;
      if (!record || !hasStagingFiles(record)) {
        return null;
      }

      const response = await commitFilesMutation.mutateAsync({
        message,
        branch: activeBranch,
        expectedHeadSha: record.baseHeadSha || headSha,
        operations: [
          ...(record.deletedPaths ?? []).map((path) => ({ path, delete: true as const })),
          ...Object.entries(record.files).map(([path, fileContent]) => ({
            path,
            content: encodeRepositoryFileContent(fileContent),
          })),
        ],
      });

      const commitSha = response?.commitSha;
      if (commitSha) {
        branchHeadRef.current = { branch: activeBranch, headSha: commitSha };
      }

      await clearStaging(canvasId, activeBranch);
      setStagingRecord(null);
      await loadBranchContent();
      return response;
    },
    [activeBranch, canvasId, commitFilesMutation, flushPendingMutation, headSha, loadBranchContent, scheduleFlush],
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
    baselineFiles,
    hasStagingChanges,
    hasCanvasStagingChanges,
    hasConsoleStagingChanges,
    hasRepositoryFilesStagingChanges,
    hasCommittedRepositoryFilesVersusLive,
    branchCanvasSpec,
    branchDashboard,
    branchHeadCanvasSpec,
    branchHeadDashboard,
    isLoadingBranchContent,
    stageCanvasWorkflow,
    setLocalBranchCanvasSpec,
    stageConsoleDashboard,
    stageRepositoryFile,
    stageRepositoryFileDelete,
    unstageRepositoryFile,
    discardStaging,
    flushStaging,
    commitStaging,
    reloadBranchContent: loadBranchContent,
    updateConsoleMutation,
    setBranchDashboard,
  };
}

export type CanvasBranchStagingState = ReturnType<typeof useCanvasBranchStaging>;
