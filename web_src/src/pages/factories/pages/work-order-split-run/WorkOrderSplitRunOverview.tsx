import type { ReactNode } from "react";

import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";

import { CLARITY_CHECK_NAME, CONFIDENCE_CHECK_NAME, isScoreCheckName } from "../../lib/confidenceScore";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import { WorkOrderCheckComment } from "../../WorkOrderCheckComment";
import { WorkOrderIntentDocument, type IntentAnalysisChat } from "./WorkOrderIntentDocument";
import type { SplitRunSource } from "./splitRunSource";
import { WorkOrderSplitRunDescription } from "./WorkOrderSplitRunDescription";
import { WorkOrderSplitRunOverviewSidebar } from "./WorkOrderSplitRunOverviewSidebar";

const SOURCE_ONLY_PANE_CLASS =
  "flex min-h-0 min-w-0 w-full flex-1 flex-col border-b border-border lg:w-[40%] lg:min-w-[14rem] lg:flex-none lg:border-r lg:border-b-0";

/**
 * Description tab. Drafts keep analysis chat on the left. After Start,
 * the left pane shows source, artifacts, and pull requests. A Planning-off
 * draft uses that same source pane and the task description on the right.
 */
export function WorkOrderSplitRunOverview({
  title,
  description,
  artifacts,
  artifactsLoading = false,
  pullRequests = [],
  pullRequestsLoading = false,
  pullRequestsError = null,
  checks,
  isAnalyzing = false,
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  expandFirstCheck = false,
  files,
  resultFooter,
  analysis,
  source,
  showContextSidebar = false,
  sourceOnly = false,
  canEditDescription = false,
  descriptionBusy = false,
  onDescriptionSave,
  sidebarNote,
}: {
  title: string;
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  artifactsLoading?: boolean;
  pullRequests?: FactoriesFactoryPullRequest[];
  pullRequestsLoading?: boolean;
  pullRequestsError?: Error | null;
  checks: WorkOrderCheckPresentation[];
  isAnalyzing?: boolean;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
  expandFirstCheck?: boolean;
  files?: FilesFile[];
  resultFooter?: ReactNode;
  analysis?: IntentAnalysisChat;
  source?: SplitRunSource;
  showContextSidebar?: boolean;
  sourceOnly?: boolean;
  canEditDescription?: boolean;
  descriptionBusy?: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  sidebarNote?: ReactNode;
}) {
  if (sourceOnly) {
    return (
      <SourceOnlyOverview
        description={description}
        artifacts={artifacts}
        artifactsLoading={artifactsLoading}
        pullRequests={pullRequests}
        pullRequestsLoading={pullRequestsLoading}
        pullRequestsError={pullRequestsError}
        organizationId={organizationId}
        factoryId={factoryId}
        orderId={orderId}
        files={files}
        source={source}
        canEditDescription={canEditDescription}
        descriptionBusy={descriptionBusy}
        onDescriptionSave={onDescriptionSave}
        sidebarNote={sidebarNote}
      />
    );
  }
  const clarity = checks.find((check) => check.name === CLARITY_CHECK_NAME);
  const confidence = checks.find((check) => check.name === CONFIDENCE_CHECK_NAME);
  const otherChecks = checks.filter((check) => !isScoreCheckName(check.name));

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="split-run-work-order-tab">
      <WorkOrderIntentDocument
        title={title}
        description={description}
        streamKey={orderId ?? orderNumber}
        streamReady={!artifactsLoading}
        artifacts={artifacts}
        clarity={clarity}
        confidence={confidence}
        isAnalyzing={isAnalyzing}
        files={files}
        resultAfterBody={
          <OverviewOtherChecks
            checks={otherChecks}
            expandFirst={expandFirstCheck && !confidence}
            organizationId={organizationId}
            factoryKey={factoryKey}
            orderNumber={orderNumber}
          />
        }
        resultFooter={resultFooter}
        analysis={analysisWithOrganization(analysis, organizationId)}
        source={source}
        contextSidebar={overviewContextSidebar({
          showContextSidebar,
          source,
          artifacts,
          artifactsLoading,
          pullRequests,
          pullRequestsLoading,
          pullRequestsError,
          sidebarNote,
        })}
      />
    </div>
  );
}

function analysisWithOrganization(analysis: IntentAnalysisChat | undefined, organizationId?: string) {
  if (!analysis || !organizationId) {
    return analysis;
  }
  return { ...analysis, organizationId };
}

function overviewContextSidebar({
  showContextSidebar,
  source,
  artifacts,
  artifactsLoading,
  pullRequests,
  pullRequestsLoading,
  pullRequestsError,
  sidebarNote,
}: {
  showContextSidebar: boolean;
  source?: SplitRunSource;
  artifacts: FactoriesWorkOrderArtifact[];
  artifactsLoading: boolean;
  pullRequests: FactoriesFactoryPullRequest[];
  pullRequestsLoading: boolean;
  pullRequestsError: Error | null;
  sidebarNote?: ReactNode;
}) {
  if (!showContextSidebar) {
    return undefined;
  }
  return (
    <WorkOrderSplitRunOverviewSidebar
      source={source}
      artifacts={artifacts}
      artifactsLoading={artifactsLoading}
      pullRequests={pullRequests}
      pullRequestsLoading={pullRequestsLoading}
      pullRequestsError={pullRequestsError}
      sidebarNote={sidebarNote}
    />
  );
}

function OverviewOtherChecks({
  checks,
  expandFirst,
  organizationId,
  factoryKey,
  orderNumber,
}: {
  checks: WorkOrderCheckPresentation[];
  expandFirst: boolean;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
}) {
  if (checks.length === 0) {
    return null;
  }
  return (
    <section className="mt-6" data-testid="split-run-overview-other-checks" aria-label="Checks">
      <h3 className="workspace-section-label">Checks</h3>
      <div className="mt-1">
        {checks.map((check, index) => (
          <WorkOrderCheckComment
            key={check.id}
            check={check}
            defaultOpen={expandFirst && index === 0}
            runHref={
              organizationId && factoryKey
                ? getWorkOrderRunHref(organizationId, factoryKey, check.appId, check.runId, { orderNumber })
                : null
            }
          />
        ))}
      </div>
    </section>
  );
}

function SourceOnlyOverview({
  description,
  artifacts,
  artifactsLoading,
  pullRequests,
  pullRequestsLoading,
  pullRequestsError,
  organizationId,
  factoryId,
  orderId,
  files,
  source,
  canEditDescription,
  descriptionBusy,
  onDescriptionSave,
  sidebarNote,
}: {
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  artifactsLoading: boolean;
  pullRequests: FactoriesFactoryPullRequest[];
  pullRequestsLoading: boolean;
  pullRequestsError: Error | null;
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  files?: FilesFile[];
  source?: SplitRunSource;
  canEditDescription: boolean;
  descriptionBusy: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  sidebarNote?: ReactNode;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="split-run-work-order-tab">
      <article className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="split-run-intent-document">
        <div className="flex min-h-0 flex-1 flex-col overflow-hidden lg:flex-row">
          <div className={SOURCE_ONLY_PANE_CLASS} data-testid="split-run-intent-request">
            <WorkOrderSplitRunOverviewSidebar
              source={source}
              artifacts={artifacts}
              artifactsLoading={artifactsLoading}
              pullRequests={pullRequests}
              pullRequestsLoading={pullRequestsLoading}
              pullRequestsError={pullRequestsError}
              sidebarNote={sidebarNote}
            />
          </div>
          <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden" data-testid="split-run-intent-result">
            <div className="min-h-0 flex-1 overflow-y-auto px-8 py-6">
              <WorkOrderSplitRunDescription
                description={description}
                canEdit={canEditDescription}
                busy={descriptionBusy}
                onSave={onDescriptionSave}
                files={files}
                organizationId={organizationId}
                factoryId={factoryId}
                orderId={orderId}
              />
            </div>
          </div>
        </div>
      </article>
    </div>
  );
}
