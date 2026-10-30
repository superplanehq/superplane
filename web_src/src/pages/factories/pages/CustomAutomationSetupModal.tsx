import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { useDeleteFactoryAutomation } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import { useCustomAutomationSetupPage } from "./useCustomAutomationSetupPage";

export function CustomAutomationSetupModal({
  lineId,
  automationId,
  onClose,
}: {
  lineId: string;
  automationId?: string;
  onClose: () => void;
}) {
  const { organizationId, factoryId, routeSegment, factory } = useFactoriesLayout();
  const model = useCustomAutomationSetupPage({
    organizationId,
    factoryId,
    factoryKey: routeSegment,
    factory: factory ?? undefined,
    lineId,
    automationId,
  });
  const editing = Boolean(automationId);
  const deleteAutomation = useDeleteFactoryAutomation(organizationId, factoryId);
  const deleting = deleteAutomation.isPending;
  const handleDelete = async () => {
    if (!automationId) {
      return;
    }
    try {
      await deleteAutomation.mutateAsync(automationId);
      showSuccessToast("Automation deleted.");
      onClose();
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to delete automation"));
    }
  };

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !model.saving && !deleting) {
          onClose();
        }
      }}
    >
      <DialogContent
        size="large"
        className="flex h-[min(52rem,calc(100vh-5rem))] w-[min(84rem,calc(100vw-5rem))] !max-w-none flex-col gap-0 overflow-hidden p-0"
        showCloseButton={!model.saving && !deleting}
        data-testid="custom-automation-setup-modal"
      >
        <DialogHeader className="shrink-0 border-b border-border px-5 py-4 pr-12">
          <DialogTitle>{editing ? "Edit automation" : "Create automation"}</DialogTitle>
        </DialogHeader>
        {editing && model.described.isPending ? (
          <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-muted-foreground" role="status">
            Loading automation…
          </div>
        ) : editing && model.described.isError ? (
          <div className="flex min-h-0 flex-1 items-center justify-center px-6">
            <div className="flex max-w-md flex-col items-center gap-3 text-center">
              <p className="text-sm text-destructive">
                {getApiErrorMessage(model.described.error, "SuperPlane could not load the automation.")}
              </p>
              <div className="flex gap-2">
                <Button type="button" variant="outline" onClick={onClose}>
                  Close
                </Button>
                <Button type="button" onClick={() => void model.described.refetch()}>
                  Try again
                </Button>
              </div>
            </div>
          </div>
        ) : (
          <CustomAutomationSetupForm
            name={model.name}
            icon={model.icon}
            mode={model.mode}
            schedule={model.schedule}
            onScheduleChange={model.setSchedule}
            draft={model.draft}
            organizationId={organizationId}
            factoryId={factoryId}
            factoryKey={routeSegment}
            saving={model.saving}
            deleting={deleting}
            error={model.saveError}
            onNameChange={model.setName}
            onIconChange={model.setIcon}
            onModeChange={model.setMode}
            onDraftChange={model.setDraft}
            onSave={() => void model.save().then((saved) => saved && onClose())}
            onDelete={editing ? () => void handleDelete() : undefined}
            onClose={onClose}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}
