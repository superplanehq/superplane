import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { History, Settings } from "lucide-react";
import { useState, type ReactNode } from "react";
import {
  useCustomAutomationSource,
  useDeleteCustomAutomationSource,
  useSaveCustomAutomationSource,
} from "@/hooks/useCustomAutomationSource";
import { useDeleteFactoryAutomation } from "@/hooks/useFactoryData";
import { useProtectCustomAutomationDraft } from "@/hooks/useProtectCustomAutomationDraft";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/ui/alertDialog";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import type { CustomAutomationColumn } from "../lib/customAutomation";
import { automationDefinitionYaml, newAutomationDefinition } from "../lib/customAutomationDefinition";
import { FactoryAutomationRunHistory } from "./automationRunHistory/FactoryAutomationRunHistory";
import { CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import { CustomAutomationSourceEditor } from "./CustomAutomationSourceEditor";
import { useCustomAutomationSourceDraft } from "@/hooks/useCustomAutomationSourceDraft";
import { useAutomationRunNavigation } from "./useAutomationRunNavigation";
import { useCustomAutomationSetupPage } from "./useCustomAutomationSetupPage";

type CustomAutomationSetupModalProps = {
  lineId: string;
  automationId?: string;
  columnKey?: CustomAutomationColumn;
  yamlEnabled?: boolean;
  onClose: () => void;
};

export function CustomAutomationSetupModal(props: CustomAutomationSetupModalProps) {
  return props.yamlEnabled ? <AutomationSourceModal {...props} /> : <LegacyAutomationModal {...props} />;
}

function LegacyAutomationModal({ lineId, automationId, columnKey, onClose }: CustomAutomationSetupModalProps) {
  const { organizationId, factoryId, routeSegment, factory } = useFactoriesLayout();
  const model = useCustomAutomationSetupPage({
    organizationId,
    factoryId,
    factoryKey: routeSegment,
    factory: factory ?? undefined,
    lineId,
    automationId,
    columnKey,
  });
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
      showErrorToast(getApiErrorMessage(error, "SuperPlane could not delete the automation."));
    }
  };

  const loading = Boolean(automationId && model.described.isPending);
  const failure = automationId && model.described.isError ? model.described.error : null;
  return (
    <AutomationDialog automationId={automationId} busy={model.saving || deleting} onClose={onClose}>
      {loading || failure ? (
        <AutomationLoadState
          loading={loading}
          error={failure}
          onClose={onClose}
          onRetry={() => void model.described.refetch()}
        />
      ) : (
        <CustomAutomationSetupForm
          name={model.name}
          icon={model.icon}
          triggers={model.triggers}
          runsOnTask={model.runsOnTask}
          columnKey={model.columnKey}
          draft={model.draft}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={routeSegment}
          saving={model.saving}
          deleting={deleting}
          error={model.saveError}
          onNameChange={model.setName}
          onIconChange={model.setIcon}
          onTriggersChange={model.setTriggers}
          onRunsOnTaskChange={(runsOnTask) => model.setRunsOnTask(runsOnTask ?? true)}
          onDraftChange={model.setDraft}
          onSave={() => void model.save().then((saved) => saved && onClose())}
          onDelete={automationId ? () => void handleDelete() : undefined}
          onClose={onClose}
        />
      )}
    </AutomationDialog>
  );
}

function AutomationSourceModal({ automationId, columnKey = "verify", onClose }: CustomAutomationSetupModalProps) {
  const { organizationId, factoryId } = useFactoriesLayout();
  const source = useCustomAutomationSource(organizationId, factoryId, automationId);
  if (automationId && (!source.data || !source.isFetchedAfterMount)) {
    return (
      <AutomationDialog automationId={automationId} onClose={onClose}>
        <AutomationLoadState
          loading={!source.isError || source.isFetching}
          error={source.error}
          onClose={onClose}
          onRetry={() => void source.refetch()}
        />
      </AutomationDialog>
    );
  }

  return (
    <AutomationSourceDraftModal
      automationId={automationId}
      stage={source.data?.stage ?? columnKey}
      initial={source.data?.yaml ?? automationDefinitionYaml(newAutomationDefinition(columnKey))}
      onClose={onClose}
    />
  );
}

function AutomationSourceDraftModal({
  automationId,
  stage,
  initial,
  onClose,
}: {
  automationId?: string;
  stage: CustomAutomationColumn;
  initial: string;
  onClose: () => void;
}) {
  const { organizationId, factoryId } = useFactoriesLayout();
  const draft = useCustomAutomationSourceDraft(initial, stage);
  const save = useSaveCustomAutomationSource(organizationId, factoryId, automationId);
  const remove = useDeleteCustomAutomationSource(organizationId, factoryId, automationId);
  const [confirmDiscard, setConfirmDiscard] = useState(false);
  const busy = save.isPending || remove.isPending;
  const requestClose = () => {
    if (busy) return;
    if (draft.dirty) {
      setConfirmDiscard(true);
      return;
    }
    onClose();
  };

  useProtectCustomAutomationDraft(draft.dirty);

  const handleSave = () => {
    if (busy) return;
    save.mutate({ stage, yaml: draft.source }, { onSuccess: onClose });
  };
  const handleDelete = () => {
    if (busy) return;
    remove.mutate(undefined, {
      onSuccess: () => {
        showSuccessToast("Automation deleted.");
        onClose();
      },
      onError: (error) => showErrorToast(getApiErrorMessage(error, "SuperPlane could not delete the automation.")),
    });
  };

  return (
    <>
      <AutomationDialog automationId={automationId} busy={busy} onClose={requestClose}>
        <CustomAutomationSourceEditor
          model={draft}
          stage={stage}
          saving={save.isPending}
          deleting={remove.isPending}
          error={save.error && save.variables?.yaml === draft.source ? save.error.message : ""}
          onSave={handleSave}
          onDelete={automationId ? handleDelete : undefined}
          onClose={requestClose}
        />
      </AutomationDialog>
      <AlertDialog open={confirmDiscard} onOpenChange={setConfirmDiscard}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Discard unsaved changes?</AlertDialogTitle>
            <AlertDialogDescription>Your changes to this automation will be lost.</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Keep editing</AlertDialogCancel>
            <AlertDialogAction onClick={onClose}>Discard changes</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

function AutomationDialog({
  automationId,
  busy = false,
  onClose,
  children,
}: {
  automationId?: string;
  busy?: boolean;
  onClose: () => void;
  children: ReactNode;
}) {
  const { organizationId, factoryId } = useFactoriesLayout();
  const editing = Boolean(automationId);
  const { tab, setTab, selectedRunId, setSelectedRunId } = useAutomationRunNavigation(editing);
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !busy) onClose();
      }}
    >
      <DialogContent
        size="large"
        className="flex h-[min(52rem,calc(100vh-5rem))] w-[min(84rem,calc(100vw-5rem))] !max-w-none flex-col gap-0 overflow-hidden p-0"
        showCloseButton={!busy}
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
            {children}
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

function AutomationLoadState({
  loading,
  error,
  onClose,
  onRetry,
}: {
  loading: boolean;
  error: unknown;
  onClose: () => void;
  onRetry: () => void;
}) {
  if (loading) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-muted-foreground" role="status">
        Loading automation…
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 items-center justify-center px-6">
      <div className="flex max-w-md flex-col items-center gap-3 text-center">
        <p className="whitespace-pre-line text-sm text-destructive" role="alert">
          {getApiErrorMessage(error, "SuperPlane could not load the automation.")}
        </p>
        <div className="flex gap-2">
          <Button type="button" variant="outline" onClick={onClose}>
            Close
          </Button>
          <Button type="button" onClick={onRetry}>
            Try again
          </Button>
        </div>
      </div>
    </div>
  );
}
