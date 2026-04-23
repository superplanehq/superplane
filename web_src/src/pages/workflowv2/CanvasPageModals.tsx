import type {
  CanvasesCanvas,
  CanvasesCanvasVersion,
  OrganizationsOrganization,
  RolesRole,
  SuperplaneUsersUser,
} from "@/api-client";
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
import { CanvasSettingsForm } from "@/pages/canvas/settings/CanvasSettingsForm";
import { CreateChangeRequestModal } from "./CreateChangeRequestModal";
import type { DraftNodeDiffSummary } from "./draftNodeDiff";

interface CanvasPageModalsProps {
  organizationId: string;
  canvas?: CanvasesCanvas | null;
  canvasVersionId?: string;
  organization?: OrganizationsOrganization | null;
  organizationUsers: SuperplaneUsersUser[];
  organizationRoles: RolesRole[];
  isUseTemplateOpen: boolean;
  onCloseUseTemplate: () => void;
  onUseTemplateSubmit: (data: { name: string; description?: string; templateId?: string }) => Promise<void>;
  isCreateCanvasPending: boolean;
  isCanvasSettingsOpen: boolean;
  onCanvasSettingsOpenChange: (open: boolean) => void;
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
  canvasVersionId,
  organization,
  organizationUsers,
  organizationRoles,
  isUseTemplateOpen,
  onCloseUseTemplate,
  onUseTemplateSubmit,
  isCreateCanvasPending,
  isCanvasSettingsOpen,
  onCanvasSettingsOpenChange,
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
          fromTemplate
        />
      ) : null}
      <Dialog open={isCanvasSettingsOpen} onOpenChange={onCanvasSettingsOpenChange}>
        <DialogContent className="flex max-h-[85vh] max-w-4xl flex-col overflow-hidden p-0">
          <DialogHeader className="border-b border-slate-200 px-6 py-4">
            <DialogTitle>Canvas settings</DialogTitle>
            <DialogDescription>Edit draft canvas settings.</DialogDescription>
          </DialogHeader>
          <div className="min-h-0 flex-1 overflow-y-auto">
            {canvas && canvasVersionId && organization ? (
              <CanvasSettingsForm
                organizationId={organizationId}
                canvasId={canvas.metadata?.id || ""}
                versionId={canvasVersionId}
                canvas={canvas}
                organization={organization}
                organizationUsers={organizationUsers}
                organizationRoles={organizationRoles}
                onClose={() => onCanvasSettingsOpenChange(false)}
              />
            ) : null}
          </div>
        </DialogContent>
      </Dialog>
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
