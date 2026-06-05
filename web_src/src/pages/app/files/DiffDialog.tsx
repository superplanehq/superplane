import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { MultiFileDiff, Virtualizer } from "@pierre/diffs/react";
import { useMemo } from "react";
import type { FileContents } from "@pierre/diffs/react";

import type { PendingFileChange } from "./types";

interface DiffDialogProps {
  changes: PendingFileChange[];
  loadedContentByPath: Record<string, string>;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function DiffDialog({ changes, loadedContentByPath, open, onOpenChange }: DiffDialogProps) {
  const diffFiles = useMemo(
    () =>
      changes.map((change) => ({
        path: change.path,
        ...buildDiffFile(change, loadedContentByPath),
      })),
    [changes, loadedContentByPath],
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

function buildDiffFile(
  change: PendingFileChange,
  loadedContentByPath: Record<string, string>,
): { oldFile: FileContents; newFile: FileContents } {
  const previousContents = change.type === "added" ? "" : (loadedContentByPath[change.path] ?? "");
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
