import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { History, Settings } from "lucide-react";
import { useDeleteFactoryAutomation } from "@/hooks/useFactoryData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import { useCustomAutomationSetupPage } from "./useCustomAutomationSetupPage";
import { useAutomationRunNavigation } from "./useAutomationRunNavigation";
import { FactoryAutomationRunHistory } from "./automationRunHistory/FactoryAutomationRunHistory";

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
  const { tab, setTab, selectedRunId, setSelectedRunId } = useAutomationRunNavigation(editing);
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
        <Tabs
          value={tab}
          onValueChange={(value) => setTab(value === "runs" ? "runs" : "automation")}
          className="min-h-0 flex-1"
        >
          <DialogHeader className="shrink-0 border-b border-border px-5 py-4 pr-12">
            <DialogTitle>{editing ? "Edit automation" : "Create automation"}</DialogTitle>
            {editing ? (
              <TabsList aria-label="Automation sections" className="mt-3">
                <TabsTrigger value="automation">
                  <Settings aria-hidden />
                  Configuration
                </TabsTrigger>
                <TabsTrigger value="runs">
                  <History aria-hidden />
                  Runs
                </TabsTrigger>
              </TabsList>
            ) : null}
          </DialogHeader>
          <TabsContent
            value="automation"
            forceMount
            hidden={tab !== "automation"}
            className="min-h-0 flex flex-col data-[state=inactive]:hidden"
          >
            <CustomAutomationConfiguration
              model={model}
              editing={editing}
              deleting={deleting}
              onDelete={editing ? () => void handleDelete() : undefined}
              onClose={onClose}
            />
          </TabsContent>
          {automationId ? (
            <TabsContent value="runs" className="min-h-0 flex flex-col data-[state=inactive]:hidden">
              <FactoryAutomationRunHistory
                canvasId={automationId}
                organizationId={organizationId}
                factoryId={factoryId}
                selectedRunId={selectedRunId}
                onSelectRun={setSelectedRunId}
              />
            </TabsContent>
          ) : null}
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}

function CustomAutomationConfiguration({
  model,
  editing,
  deleting,
  onDelete,
  onClose,
}: {
  model: ReturnType<typeof useCustomAutomationSetupPage>;
  editing: boolean;
  deleting: boolean;
  onDelete?: () => void;
  onClose: () => void;
}) {
  const { organizationId, factoryId, routeSegment } = useFactoriesLayout();
  if (editing && model.described.isPending) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-muted-foreground" role="status">
        Loading automation…
      </div>
    );
  }
  if (editing && model.described.isError) {
    return (
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
    );
  }
  return (
    <CustomAutomationSetupForm
      name={model.name}
      icon={model.icon}
      mode={model.mode}
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
      onDelete={onDelete}
      onClose={onClose}
    />
  );
}
