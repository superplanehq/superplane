import { useState, type ComponentProps } from "react";
import { useFactoryAutomationNavigation } from "@/hooks/useFactoryAutomationNavigation";
import { getApiErrorMessage } from "@/lib/errors";
import { useOptionalFactoriesLayout } from "../layout/factoriesLayoutContext";
import type { ColumnAutomationViewTab } from "../lib/columnAutomations";
import { ReviewRequestDecisionSummary } from "./ReviewRequestDecisionSummary";
import { FactoryAutomationHistory } from "./FactoryAutomationHistory";
import { FactoryAutomationSettingsTabs } from "./FactoryAutomationSettingsTabs";
import { MergeConfidenceDeleteDialog } from "./MergeConfidenceConfigActions";
import type { NodeConfigurationSave } from "./nodeConfigurationCanvas";
import { PlanningSettingsFooter } from "./PlanningSettingsConfiguration";
import { PlanningSettingsMenu } from "./PlanningSettingsMenu";
import { ReviewRequestSettingsConfiguration } from "./ReviewRequestSettingsConfiguration";
import { DEFAULT_SPLIT_CONFIG_DELETE_COPY } from "./splitConfigDeleteCopy";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";
import { useReviewRequestReviewers } from "./useReviewRequestReviewers";
import { useReviewRequestSettingsDraft } from "./useReviewRequestSettingsDraft";
import { PopupHeader, PopupShell } from "./work-order-popup-redesign/popupShared";

export interface ReviewRequestSettingsPopupProps {
  organizationId: string;
  factoryId: string;
  canvasId: string;
  graph: IntakeAutomationGraph;
  loading?: boolean;
  error?: boolean;
  onRetry?: () => void;
  onSaveNode: (update: NodeConfigurationSave) => Promise<void>;
  reviewers: string[];
  reviewersLoading?: boolean;
  reviewersError?: boolean;
  onRetryReviewers?: () => void;
  onDelete?: () => Promise<void> | void;
  deletePending?: boolean;
  initialTab?: ColumnAutomationViewTab;
  onClose: () => void;
}

export function ReviewRequestSettingsPopup(props: ReviewRequestSettingsPopupProps) {
  const history = useFactoryAutomationNavigation({
    organizationId: props.organizationId,
    factoryId: props.factoryId,
    graph: props.graph,
    parameter: "automationTab",
    initialTab: props.initialTab,
    hasAgent: false,
  });
  const editor = useReviewRequestSettingsDraft(props);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleteError, setDeleteError] = useState<string>();
  return (
    <>
      <PopupShell testId="review-request-settings" canvas fixed onDismiss={props.onClose}>
        <PopupHeader
          title="Review request"
          onClose={props.onClose}
          actions={
            <PlanningSettingsMenu
              title="Review request"
              noun="Review request"
              testId="review-request-settings"
              enabled={editor.enabled}
              disabled={editor.saving || editor.unavailable}
              onEnabledChange={(next) => void editor.setEnabled(next)}
              onDelete={props.onDelete ? () => setConfirmDelete(true) : undefined}
            />
          }
        >
          <FactoryAutomationSettingsTabs
            tab={history.tab}
            onTabChange={history.setTab}
            hasAgent={false}
            generalLabel="Configuration"
            historyEnabled={history.enabled}
            label="Review request sections"
            testId="review-request-settings"
          />
        </PopupHeader>
        {history.tab === "runs" ? (
          <FactoryAutomationHistory
            organizationId={props.organizationId}
            factoryId={props.factoryId}
            canvasId={props.canvasId}
            selectedRunId={history.selectedRunId}
            onSelectRun={history.setSelectedRunId}
            loading={props.loading}
            error={props.error}
            onRetry={props.onRetry}
            runRetryEnabled={editor.enabled}
            renderRunSummary={(run, executions) => <ReviewRequestDecisionSummary run={run} executions={executions} />}
          />
        ) : (
          <ReviewRequestSettingsConfiguration {...props} editor={editor} />
        )}
        {history.tab === "general" ? (
          <PlanningSettingsFooter
            error={editor.saveError}
            saving={editor.saving}
            disabled={editor.saving || editor.unavailable || editor.findingsMissingDestination}
            onSave={() => void editor.save()}
            testId="review-request-settings-save"
          />
        ) : null}
      </PopupShell>
      <MergeConfidenceDeleteDialog
        open={confirmDelete}
        pending={props.deletePending ?? false}
        error={deleteError}
        copy={DEFAULT_SPLIT_CONFIG_DELETE_COPY}
        onOpenChange={setConfirmDelete}
        onConfirm={async () => {
          setDeleteError(undefined);
          try {
            await props.onDelete?.();
          } catch (failure) {
            setDeleteError(getApiErrorMessage(failure, "Could not delete Review request. Try again."));
          }
        }}
      />
    </>
  );
}

export function ReviewRequestSettingsHost(
  props: Omit<
    ComponentProps<typeof ReviewRequestSettingsPopup>,
    "reviewers" | "reviewersLoading" | "reviewersError" | "onRetryReviewers"
  >,
) {
  const layout = useOptionalFactoriesLayout();
  const github = props.graph.specNodes?.find((node) => node.id === "request-reviewers")?.integration?.id;
  const reviewers = useReviewRequestReviewers(
    props.organizationId,
    props.factoryId,
    github || layout?.factory?.onboarding?.vcsIntegrationId || "",
  );
  return (
    <ReviewRequestSettingsPopup
      {...props}
      reviewers={reviewers.reviewers}
      reviewersLoading={reviewers.isLoading}
      reviewersError={reviewers.isError}
      onRetryReviewers={reviewers.refetch}
    />
  );
}
