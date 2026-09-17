import { useEffect, useState } from "react";
import { TriangleAlert } from "lucide-react";

import { Frame, FrameHeader, FramePanel } from "@/components/reui/frame";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogTitle,
} from "@/ui/alertDialog";

import { START_CONFIRM_COPY, startConfirmBody, type StartConfirmScores } from "./startConfirm";

const ACTION_CLASS = "rounded-md";

export function StartConfirmDialog({
  open,
  scores,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  scores: StartConfirmScores;
  onOpenChange: (open: boolean) => void;
  onConfirm: (skipNext: boolean) => void;
}) {
  const [skipNext, setSkipNext] = useState(false);
  const body = startConfirmBody(scores) ?? START_CONFIRM_COPY.missing;

  useEffect(() => {
    if (open) {
      setSkipNext(false);
    }
  }, [open]);

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent className="overflow-hidden p-0 ring-0">
        <Frame variant="inverse" dense className="border-0 bg-popover" data-testid="split-run-start-confirm">
          <FrameHeader>
            <div className="flex items-start gap-3">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-full border border-amber-100 bg-amber-50 text-amber-500 dark:bg-amber-950 dark:text-amber-300">
                <TriangleAlert className="size-5" aria-hidden />
              </div>
              <div className="flex flex-col justify-center gap-1">
                <AlertDialogTitle className="text-sm font-semibold">{START_CONFIRM_COPY.title}</AlertDialogTitle>
                <AlertDialogDescription className="text-sm text-muted-foreground">{body}</AlertDialogDescription>
              </div>
            </div>
          </FrameHeader>
          <FramePanel>
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div className="flex items-center gap-2">
                <Checkbox
                  id="split-run-start-dont-ask"
                  checked={skipNext}
                  onChange={(event) => setSkipNext(event.currentTarget.checked)}
                  data-testid="split-run-start-dont-ask"
                />
                <Label htmlFor="split-run-start-dont-ask" className="font-normal text-muted-foreground">
                  {START_CONFIRM_COPY.skip}
                </Label>
              </div>
              <div className="flex items-center gap-2">
                <AlertDialogCancel className={ACTION_CLASS}>{START_CONFIRM_COPY.cancel}</AlertDialogCancel>
                <AlertDialogAction className={ACTION_CLASS} onClick={() => onConfirm(skipNext)}>
                  {START_CONFIRM_COPY.confirm}
                </AlertDialogAction>
              </div>
            </div>
          </FramePanel>
        </Frame>
      </AlertDialogContent>
    </AlertDialog>
  );
}
