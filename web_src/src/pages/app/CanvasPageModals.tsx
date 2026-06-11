import type { CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { CreateChangeRequestModal } from "./CreateChangeRequestModal";
import type { DraftNodeDiffSummary } from "./draftNodeDiff";

interface CanvasPageModalsProps {
  isCreateChangeRequestMode: boolean;
  onCreateChangeRequestModeChange: (open: boolean) => void;
  isCreateChangeRequestPending: boolean;
  createChangeRequestVersion?: CanvasesCanvasVersion;
  createChangeRequestTitle: string;
  createChangeRequestDescription: string;
  onCreateChangeRequestTitleChange: (value: string) => void;
  onCreateChangeRequestDescriptionChange: (value: string) => void;
  createChangeRequestNodeDiffSummary: DraftNodeDiffSummary;
  isCreateChangeRequestDraftOutdated: boolean;
  onSubmitCreateChangeRequest: () => void;
  canvasDeletedRemotely: boolean;
  onGoToCanvases: () => void;
}

export function CanvasPageModals({
  isCreateChangeRequestMode,
  onCreateChangeRequestModeChange,
  isCreateChangeRequestPending,
  createChangeRequestVersion,
  createChangeRequestTitle,
  createChangeRequestDescription,
  onCreateChangeRequestTitleChange,
  onCreateChangeRequestDescriptionChange,
  createChangeRequestNodeDiffSummary,
  isCreateChangeRequestDraftOutdated,
  onSubmitCreateChangeRequest,
  canvasDeletedRemotely,
  onGoToCanvases,
}: CanvasPageModalsProps) {
  return (
    <>
      <CreateChangeRequestModal
        open={isCreateChangeRequestMode}
        onOpenChange={onCreateChangeRequestModeChange}
        pending={isCreateChangeRequestPending}
        version={createChangeRequestVersion}
        title={createChangeRequestTitle}
        description={createChangeRequestDescription}
        onTitleChange={onCreateChangeRequestTitleChange}
        onDescriptionChange={onCreateChangeRequestDescriptionChange}
        diffSummary={createChangeRequestNodeDiffSummary}
        isDraftOutdated={isCreateChangeRequestDraftOutdated}
        onPublish={onSubmitCreateChangeRequest}
      />
      <Dialog open={canvasDeletedRemotely} onOpenChange={() => {}}>
        <DialogContent showCloseButton={false}>
          <DialogHeader>
            <DialogTitle>Canvas deleted</DialogTitle>
            <DialogDescription>
              This canvas was deleted from another session. You can no longer edit or run it.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button onClick={onGoToCanvases}>Go to canvases</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
