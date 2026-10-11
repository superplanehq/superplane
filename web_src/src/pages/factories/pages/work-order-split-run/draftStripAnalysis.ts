import type { ReactNode } from "react";

import type { FactoriesFactory } from "@/api-client";

import { factoryShowsClarity, factoryShowsConfidence } from "../planningSettingsModel";
import { composerCreditVerdict, creditBillingHref } from "./splitRunFooter";
import type { SplitRunFixture } from "./splitRunMocks";
import type { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";

export type DraftStripAnalysisArgs = {
  factory?: FactoriesFactory | null;
  organizationId?: string;
  factoryKey?: string;
  fixture: SplitRunFixture;
  analysis: ReturnType<typeof useAnalysisPlanningSession>;
};

/**
 * Refine chat fields for a draft: the planning session, the model select,
 * the score toggles, and the credit verdict. A task that is not a draft has
 * no refine chat.
 */
export function draftStripAnalysis(args: DraftStripAnalysisArgs, modelSelect: ReactNode | undefined) {
  if (args.fixture.footer.kind !== "draft") {
    return undefined;
  }
  const billingHref = creditBillingHref(args.organizationId, args.factoryKey);
  return {
    ...args.analysis,
    modelSelect,
    showClarity: factoryShowsClarity(args.factory),
    showConfidence: factoryShowsConfidence(args.factory),
    creditVerdict: composerCreditVerdict(args.fixture.footer.note, billingHref),
  };
}
