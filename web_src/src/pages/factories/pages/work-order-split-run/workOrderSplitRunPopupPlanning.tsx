import type { ReactNode } from "react";

import type { FactoriesFactory } from "@/api-client";

import { factoryPlanningEnabled, factoryShowsClarity, factoryShowsConfidence } from "../planningSettingsModel";
import type { CreatedTaskHref } from "./CreatedTaskCard";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { classicSplitRunFooter } from "./splitRunFooter";
import { SPLIT_RUN_POPUP_DIALOG_CLASSNAME } from "./splitRunPopupModel";
import type { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";
import type { WorkOrderSplitRunPopupProps } from "./WorkOrderSplitRunBody";
import { createdTaskHref } from "./workOrderPopupActions";

export function analysisDraftChrome(args: {
  factory?: FactoriesFactory;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  lineId?: string;
  fixture: WorkOrderSplitRunPopupProps["fixture"];
  analysis: ReturnType<typeof useAnalysisPlanningSession>;
  draftModel: string;
  draftThinking: string;
  onDraftStartChange: (next: { model: string; thinkingLevel: string }) => void;
  disabled: boolean;
}) {
  if (!factoryPlanningEnabled(args.factory)) {
    return { footerModelSelect: undefined, stripAnalysis: undefined };
  }
  const modelSelects = draftModelSelects({
    organizationId: args.organizationId,
    factoryId: args.factoryId,
    fixture: args.fixture,
    model: args.draftModel,
    thinkingLevel: args.draftThinking,
    onChange: args.onDraftStartChange,
    disabled: args.disabled,
  });
  return {
    footerModelSelect: modelSelects.footer,
    stripAnalysis: draftStripAnalysis(args.fixture.footer.kind, args.analysis, modelSelects.strip, {
      taskHref: createdTaskHref(args.organizationId, args.factoryKey, args.lineId),
      scores: {
        showClarity: factoryShowsClarity(args.factory),
        showConfidence: factoryShowsConfidence(args.factory),
      },
      creditNotice: args.fixture.footer.creditNotice,
    }),
  };
}

/** A draft with Planning on uses the compact review; a source-only draft keeps the classic one. */
export function analysisReviewCompact(
  showSidebarNote: boolean,
  footerKind: WorkOrderSplitRunPopupProps["fixture"]["footer"]["kind"],
  sourceOnly: boolean,
) {
  return showSidebarNote || (footerKind === "draft" && !sourceOnly);
}

/** In the classic tabs, the review renders inside the description tab only. */
export function showsDescriptionReview(sourceOnly: boolean, showSidebarNote: boolean, tab: string) {
  return !sourceOnly && !showSidebarNote && tab === "description";
}

/** In the classic tabs, the review renders below the tabs when the description tab does not carry it. */
export function analysisShellReview(sourceOnly: boolean, showSidebarNote: boolean, tab: string, review: ReactNode) {
  if (sourceOnly || (!showSidebarNote && tab !== "description")) {
    return review;
  }
  return null;
}

/** The unified view is wide; the Planning draft and the classic source-only view keep the refine widths. */
export function analysisPopupClassName(fullPage: boolean, unified: boolean, classicSourceOnly: boolean) {
  if (fullPage || classicSourceOnly) {
    return undefined;
  }
  if (unified) {
    return "h-[min(52rem,calc(100vh-5rem))] w-[min(80rem,calc(100vw-5rem))]";
  }
  return SPLIT_RUN_POPUP_DIALOG_CLASSNAME;
}

/**
 * The refine strip only shows for a draft. It gets the ghost model select
 * and a permalink builder for tasks the agent splits off this one.
 */
export function analysisPopupView(
  fixture: WorkOrderSplitRunPopupProps["fixture"],
  factory: FactoriesFactory | undefined,
) {
  const sourceOnly = fixture.footer.kind === "draft" && !factoryPlanningEnabled(factory);
  return {
    sourceOnly,
    viewFixture: sourceOnly ? { ...fixture, footer: classicSplitRunFooter(fixture.footer) } : fixture,
  };
}

function draftStripAnalysis(
  footerKind: WorkOrderSplitRunPopupProps["fixture"]["footer"]["kind"],
  analysis: ReturnType<typeof useAnalysisPlanningSession>,
  modelSelect: ReactNode | undefined,
  extras: {
    taskHref: CreatedTaskHref;
    scores: { showClarity: boolean; showConfidence: boolean };
    creditNotice: WorkOrderSplitRunPopupProps["fixture"]["footer"]["creditNotice"];
  },
) {
  if (footerKind !== "draft") {
    return undefined;
  }
  return { ...analysis, modelSelect, ...extras.scores, taskHref: extras.taskHref, creditNotice: extras.creditNotice };
}

/**
 * One model select per surface: `labeled` for the footer capsule under the
 * plan, `ghost` for the refine strip settings row. Only a draft with Start
 * gets one.
 */
function draftModelSelects(args: {
  organizationId?: string;
  factoryId?: string;
  fixture: WorkOrderSplitRunPopupProps["fixture"];
  model: string;
  thinkingLevel: string;
  onChange: (next: { model: string; thinkingLevel: string }) => void;
  disabled: boolean;
}): { footer?: ReactNode; strip?: ReactNode } {
  const { fixture, ...select } = args;
  if (fixture.footer.kind !== "draft" || !fixture.footer.actions.some((action) => action.kind === "start")) {
    return {};
  }
  return {
    footer: <DraftStartModelSelect {...select} lineName={fixture.lineName} appearance="labeled" />,
    strip: <DraftStartModelSelect {...select} lineName={fixture.lineName} appearance="ghost" />,
  };
}
