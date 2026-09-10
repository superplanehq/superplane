import type { ReactNode } from "react";

import type { FactoriesWorkOrderArtifact, FilesFile } from "@/api-client";

import { CONFIDENCE_CHECK_NAME } from "../../lib/confidenceScore";
import type { WorkOrderCheckPresentation } from "../../lib/workOrderChecks";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import { WorkOrderCheckComment } from "../../WorkOrderCheckComment";
import { WorkOrderIntentDocument, type IntentAnalysisChat } from "./WorkOrderIntentDocument";

/**
 * Description tab: original request as a chat and the generated plan.
 */
export function WorkOrderSplitRunOverview({
  title,
  description,
  artifacts,
  checks,
  isAnalyzing = false,
  organizationId,
  factoryKey,
  orderNumber,
  expandFirstCheck = false,
  files,
  resultFooter,
  analysis,
}: {
  title: string;
  description: string;
  artifacts: FactoriesWorkOrderArtifact[];
  checks: WorkOrderCheckPresentation[];
  isAnalyzing?: boolean;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  expandFirstCheck?: boolean;
  files?: FilesFile[];
  resultFooter?: ReactNode;
  analysis?: IntentAnalysisChat;
}) {
  const confidence = checks.find((check) => check.name === CONFIDENCE_CHECK_NAME);
  const otherChecks = checks.filter((check) => check.name !== CONFIDENCE_CHECK_NAME);

  return (
    <div className="flex min-h-0 flex-1 flex-col" data-testid="split-run-work-order-tab">
      <WorkOrderIntentDocument
        title={title}
        description={description}
        artifacts={artifacts}
        confidence={confidence}
        isAnalyzing={isAnalyzing}
        files={files}
        resultAfterBody={
          otherChecks.length > 0 ? (
            <section className="mt-6" data-testid="split-run-overview-other-checks" aria-label="Checks">
              <h3 className="workspace-section-label">Checks</h3>
              <div className="mt-1">
                {otherChecks.map((check, index) => (
                  <WorkOrderCheckComment
                    key={check.id}
                    check={check}
                    defaultOpen={expandFirstCheck && !confidence && index === 0}
                    runHref={
                      organizationId && factoryKey
                        ? getWorkOrderRunHref(organizationId, factoryKey, check.appId, check.runId, { orderNumber })
                        : null
                    }
                  />
                ))}
              </div>
            </section>
          ) : null
        }
        resultFooter={resultFooter}
        analysis={
          analysis && organizationId ? { ...analysis, organizationId } : analysis
        }
      />
    </div>
  );
}
