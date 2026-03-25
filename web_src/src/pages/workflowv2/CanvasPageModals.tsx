import { CanvasesCanvas, CanvasesCanvasVersion } from "@/api-client";
import { CreateCanvasModal } from "@/components/CreateCanvasModal";
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
import { DraftNodeDiffSummary } from "./draftNodeDiff";

interface CanvasPageModalsProps {
  organizationId: string;
  canvas?: CanvasesCanvas | null;
  isUseTemplateOpen: boolean;
  onCloseUseTemplate: () => void;
  onUseTemplateSubmit: (data: { name: string; description?: string; templateId?: string }) => Promise<void>;
  isCreateCanvasPending: boolean;
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
  organizationId,
  canvas,
  isUseTemplateOpen,
  onCloseUseTemplate,
  onUseTemplateSubmit,
  isCreateCanvasPending,
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
      {canvas ? (
        <CreateCanvasModal
          organizationId={organizationId}
          isOpen={isUseTemplateOpen}
          onClose={onCloseUseTemplate}
          onSubmit={onUseTemplateSubmit}
          isLoading={isCreateCanvasPending}
          templates={[
            {
              id: canvas.metadata?.id || "",
              name: canvas.metadata?.name || "Untitled template",
              description: canvas.metadata?.description,
            },
          ]}
          defaultTemplateId={canvas.metadata?.id || ""}
          mode="create"
          fromTemplate
        />
      ) : null}
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
