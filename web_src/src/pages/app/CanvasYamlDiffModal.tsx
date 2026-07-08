import { MultiFileDiff, type FileContents, type FileDiffMetadata } from "@pierre/diffs/react";
import { FileCode, XIcon } from "lucide-react";
import { useCallback, useMemo } from "react";

import { Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { useTheme } from "@/contexts/useTheme";
import { getCanvasYamlDiffOptions } from "./canvasYamlDiffOptions";

export type CanvasYamlDiffModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  liveYamlText: string;
  draftYamlText: string;
  filename: string;
  title?: string;
  dialogTitle?: string;
  description?: string;
  liveLabel?: string;
  draftLabel?: string;
};

export function CanvasYamlDiffModal({
  open,
  onOpenChange,
  dialogTitle = "Canvas YAML diff",
  description = "Side-by-side YAML comparison between live and draft canvas versions.",
  ...bodyProps
}: CanvasYamlDiffModalProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className="flex h-[90vh] w-[94vw] max-w-[1600px] flex-col gap-0 overflow-hidden p-0"
        showCloseButton={false}
      >
        <DialogTitle className="sr-only">{dialogTitle}</DialogTitle>
        <DialogDescription className="sr-only">{description}</DialogDescription>
        {/* Body is only mounted by DialogContent while open, so theme/diff hooks never run when closed. */}
        <CanvasYamlDiffBody {...bodyProps} />
      </DialogContent>
    </Dialog>
  );
}

type CanvasYamlDiffBodyProps = Omit<CanvasYamlDiffModalProps, "open" | "onOpenChange" | "dialogTitle" | "description">;

function CanvasYamlDiffBody({
  liveYamlText,
  draftYamlText,
  filename,
  title = "Show Diff",
  liveLabel = "Live",
  draftLabel = "Draft",
}: CanvasYamlDiffBodyProps) {
  const { resolvedTheme } = useTheme();
  const diffOptions = useMemo(() => getCanvasYamlDiffOptions(resolvedTheme), [resolvedTheme]);

  const { oldFile, newFile } = useMemo(() => {
    const liveFile: FileContents = {
      name: filename,
      contents: liveYamlText,
      lang: "yaml",
    };
    const draftFile: FileContents = {
      name: filename,
      contents: draftYamlText,
      lang: "yaml",
    };

    return { oldFile: liveFile, newFile: draftFile };
  }, [draftYamlText, filename, liveYamlText]);

  const renderDiffHeader = useCallback(
    (fileDiff: FileDiffMetadata) => {
      const { additions, deletions } = fileDiff.hunks.reduce(
        (totals, hunk) => ({
          additions: totals.additions + hunk.additionLines,
          deletions: totals.deletions + hunk.deletionLines,
        }),
        { additions: 0, deletions: 0 },
      );

      return (
        <div className="w-full">
          <div className="flex min-h-11 items-center justify-between gap-4 border-b border-slate-200 bg-white px-4 py-2 dark:border-gray-700/70 dark:bg-gray-900">
            <div className="flex min-w-0 items-center gap-2">
              <FileCode className="h-4 w-4 shrink-0 text-slate-500 dark:text-gray-400" aria-hidden />
              <span className="truncate font-sans text-sm font-medium text-slate-900 dark:text-gray-100">
                {filename}
              </span>
            </div>
            <div className="flex shrink-0 items-center gap-2 font-mono text-xs">
              <span className="text-red-600 dark:text-red-400">-{deletions}</span>
              <span className="text-emerald-600 dark:text-emerald-400">+{additions}</span>
            </div>
          </div>
          <div className="grid grid-cols-2 font-mono text-xs font-semibold">
            <div className="border-r border-slate-200 bg-red-50/70 px-4 py-1.5 text-red-700 dark:border-gray-700/70 dark:bg-red-950/40 dark:text-red-300">
              {liveLabel}
            </div>
            <div className="bg-emerald-50/70 px-4 py-1.5 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300">
              {draftLabel}
            </div>
          </div>
        </div>
      );
    },
    [draftLabel, filename, liveLabel],
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col bg-slate-50 dark:bg-gray-900">
      <div className="relative flex items-center border-b border-slate-200 bg-white px-5 py-3 pr-12 dark:border-gray-700/70 dark:bg-gray-900">
        <div className="flex min-w-0 items-center gap-3">
          <h2 className="truncate text-sm font-medium text-slate-900 dark:text-gray-100">{title}</h2>
        </div>
        <DialogClose className="absolute top-1/2 right-2 flex h-6 w-6 -translate-y-1/2 cursor-pointer items-center justify-center rounded leading-none hover:bg-slate-950/5 focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none dark:hover:bg-white/10">
          <XIcon className="h-4 w-4" />
          <span className="sr-only">Close</span>
        </DialogClose>
      </div>

      <div className="min-h-0 flex-1 overflow-auto">
        <div className="min-w-[980px] border-x border-b border-slate-200 bg-white dark:border-gray-700/70 dark:bg-gray-900">
          <MultiFileDiff
            oldFile={oldFile}
            newFile={newFile}
            disableWorkerPool
            renderCustomHeader={renderDiffHeader}
            options={diffOptions}
          />
        </div>
      </div>
    </div>
  );
}
