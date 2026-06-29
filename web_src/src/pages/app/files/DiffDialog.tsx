import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { MultiFileDiff, Virtualizer } from "@pierre/diffs/react";
import { useMemo } from "react";
import type { FileContents } from "@pierre/diffs/react";

import type { PendingFileChange, StagedFileDiff } from "./types";

interface DiffDialogProps {
  changes: PendingFileChange[];
  committedContentByPath: Record<string, string>;
  loadedContentByPath: Record<string, string>;
  stagedFileDiffs: StagedFileDiff[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function DiffDialog({
  changes,
  committedContentByPath,
  loadedContentByPath,
  stagedFileDiffs,
  open,
  onOpenChange,
}: DiffDialogProps) {
  const diffFiles = useMemo(() => {
    const pendingDiffs = changes.map((change) => ({
      path: change.path,
      ...buildDiffFile(change, committedContentByPath, loadedContentByPath),
    }));
    const stagedDiffs = stagedFileDiffs.map((diff) => ({
      path: diff.path,
      ...buildStagedDiffFile(diff),
    }));
    return [...stagedDiffs, ...pendingDiffs].sort((left, right) => left.path.localeCompare(right.path));
  }, [changes, committedContentByPath, loadedContentByPath, stagedFileDiffs]);
  const diffOptions = useMemo(
    () => ({
      theme: "pierre-light" as const,
      themeType: "light" as const,
      diffStyle: "split" as const,
      stickyHeader: true,
    }),
    [],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="90vw" className="grid grid-rows-[auto_minmax(0,1fr)] gap-4 p-0">
        <DialogHeader className="border-b border-slate-950/15 px-5 py-4">
          <DialogTitle>Diff</DialogTitle>
        </DialogHeader>
        <div className="min-h-0 overflow-hidden">
          {diffFiles.length === 0 ? (
            <div className="flex h-full items-center justify-center text-sm text-slate-500">No changes</div>
          ) : (
            <Virtualizer className="h-full overflow-auto" contentClassName="min-w-0">
              <div className="space-y-4 p-4">
                {diffFiles.map(({ path, oldFile, newFile }) => (
                  <MultiFileDiff
                    key={path}
                    oldFile={oldFile}
                    newFile={newFile}
                    options={diffOptions}
                    className="overflow-hidden rounded border border-slate-950/15 bg-white"
                  />
                ))}
              </div>
            </Virtualizer>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function buildDiffFile(
  change: PendingFileChange,
  committedContentByPath: Record<string, string>,
  loadedContentByPath: Record<string, string>,
): { oldFile: FileContents; newFile: FileContents } {
  // Use committed (stage=false) content as the diff baseline. loadedContentByPath
  // reflects staged content for draft reads, so after autosave to staging it would
  // match the edited content and the diff would appear empty.
  const baselineContents = committedContentByPath[change.path] ?? loadedContentByPath[change.path] ?? "";
  const previousContents = change.type === "added" ? "" : baselineContents;
  const nextContents = change.type === "deleted" ? "" : change.content;

  return {
    oldFile: {
      name: change.path,
      contents: previousContents,
      cacheKey: `${change.path}:old:${previousContents}`,
    },
    newFile: {
      name: change.path,
      contents: nextContents,
      cacheKey: `${change.path}:new:${nextContents}`,
    },
  };
}

function buildStagedDiffFile(diff: StagedFileDiff): { oldFile: FileContents; newFile: FileContents } {
  return {
    oldFile: {
      name: diff.path,
      contents: diff.committedContent,
      cacheKey: `${diff.path}:old:${diff.committedContent}`,
    },
    newFile: {
      name: diff.path,
      contents: diff.effectiveContent,
      cacheKey: `${diff.path}:new:${diff.effectiveContent}`,
    },
  };
}
