import type { CanvasesCanvasVersion } from "@/api-client";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useMemo } from "react";
import { summarizeNodeDiff } from "./summarizeNodeDiff";
import { formatTimestamp } from "./versionNodeDiffUtils";
import { VersionNodeDiffAccordion } from "./VersionNodeDiff";

export type CanvasVersionNodeDiffContext = {
  version: CanvasesCanvasVersion;
  previousVersion: CanvasesCanvasVersion;
};

export function CanvasVersionNodeDiffDialog({
  context,
  onOpenChange,
}: {
  context: CanvasVersionNodeDiffContext | null;
  onOpenChange: (open: boolean) => void;
}) {
  const diffSummary = useMemo(() => {
    if (!context) {
      return null;
    }
    return summarizeNodeDiff(context.version, context.previousVersion);
  }, [context]);

  const ownerName = context?.version.metadata?.owner?.name || "Unknown owner";
  const createdAt = formatTimestamp(context?.version.metadata?.createdAt);

  return (
    <Dialog open={!!context} onOpenChange={onOpenChange}>
      <DialogContent className="min-w-[60vw] max-w-5xl max-h-[92vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="text-base">Version Node Diff</DialogTitle>
          {context ? (
            <DialogDescription asChild className="text-[13px] text-slate-600">
              <div className="text-left">
                <p className="m-0 leading-normal">
                  {ownerName}
                  {createdAt ? ` on ${createdAt}` : ""}
                </p>
              </div>
            </DialogDescription>
          ) : null}
        </DialogHeader>

        {!diffSummary ? null : <VersionNodeDiffAccordion summary={diffSummary} />}
      </DialogContent>
    </Dialog>
  );
}
