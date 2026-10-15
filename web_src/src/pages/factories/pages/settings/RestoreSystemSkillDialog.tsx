import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { LoadingButton } from "@/components/ui/loading-button";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

export function RestoreSystemSkillDialog({
  open,
  isRestoring,
  onClose,
  onConfirm,
}: {
  open: boolean;
  isRestoring: boolean;
  onClose: () => void;
  onConfirm: () => Promise<void>;
}) {
  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next && !isRestoring) onClose();
      }}
    >
      <DialogContent
        role="alertdialog"
        showCloseButton={!isRestoring}
        onInteractOutside={(event) => event.preventDefault()}
        onEscapeKeyDown={(event) => {
          if (isRestoring) event.preventDefault();
        }}
      >
        <DialogHeader>
          <DialogTitle>{AGENT_RESOURCES_COPY.restoreSkillTitle}</DialogTitle>
          <DialogDescription>{AGENT_RESOURCES_COPY.restoreSkillDescription}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isRestoring}>
            {AGENT_RESOURCES_COPY.cancel}
          </Button>
          <LoadingButton loading={isRestoring} disabled={isRestoring} onClick={() => void onConfirm()}>
            {AGENT_RESOURCES_COPY.restoreSkill}
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
