import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { MultiFileDiff, Virtualizer } from "@pierre/diffs/react";
import { useMemo } from "react";
import type { FileContents } from "@pierre/diffs/react";

import type { FileDiffVersusLive } from "./types";

interface DiffDialogProps {
  fileDiffs: FileDiffVersusLive[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function DiffDialog({ fileDiffs, open, onOpenChange }: DiffDialogProps) {
  const diffFiles = useMemo(
    () =>
      [...fileDiffs]
        .sort((left, right) => left.path.localeCompare(right.path))
        .map((diff) => ({ path: diff.path, ...buildDiffFile(diff) })),
    [fileDiffs],
  );
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

// Diff the live (main) content against the effective draft content. The Files diff
// always uses live as the baseline so it matches the canvas tab's "draft vs live"
// comparison, regardless of whether the change is committed or still staged.
function buildDiffFile(diff: FileDiffVersusLive): { oldFile: FileContents; newFile: FileContents } {
  return {
    oldFile: {
      name: diff.path,
      contents: diff.liveContent,
      cacheKey: `${diff.path}:old:${diff.liveContent}`,
    },
    newFile: {
      name: diff.path,
      contents: diff.draftContent,
      cacheKey: `${diff.path}:new:${diff.draftContent}`,
    },
  };
}
