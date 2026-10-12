import type { FactoriesFactory } from "@/api-client";
import { useLocation, useNavigate } from "react-router";

import {
  factoryHomePath,
  columnAutomationViewCanvasIdFromSearch,
  intakeIdFromSearch,
  intakeSettingsTabFromSearch,
  isIntakeSearchOpen,
  isPRFeedbackSearchOpen,
  isPlanningSearchOpen,
  planningSettingsTabFromSearch,
  prFeedbackHandlerIdFromSearch,
  prFeedbackSettingsTabFromSearch,
} from "../lib/factoryPagePaths";
import { IntakeSettingsHost } from "../pages/IntakeSettingsHost";
import { isIntakeSettingsTab, type IntakeSettingsTab } from "../pages/intakeSourceSettingsModel";
import type { ConfiguredLineIntakeSource } from "../pages/lineIntakeModel";
import { PRFeedbackSettingsHost } from "../pages/PRFeedbackSettingsHost";
import { PlanningSettingsHost } from "../pages/PlanningSettingsHost";
import { isPlanningSettingsTab } from "../pages/planningSettingsModel";
import { ColumnAutomationViewHost } from "../pages/ColumnAutomationViewPopup";
import { isPRFeedbackSettingsTab, type PRFeedbackSettingsTab } from "../pages/prFeedbackSettingsModel";

type FactoryOnboarding = NonNullable<FactoriesFactory["onboarding"]>;

/** Automation settings opened from the board menu URL. */
export function MobileBoardSettings({
  organizationId,
  factoryId,
  routeSegment,
  lineId,
  intakes,
  canUpdate,
  onboarding,
  apps,
}: {
  organizationId: string;
  factoryId: string;
  routeSegment: string;
  lineId: string;
  intakes: ConfiguredLineIntakeSource[];
  canUpdate: boolean;
  onboarding?: FactoryOnboarding;
  apps: Array<{ id?: string; name?: string }>;
}) {
  const navigate = useNavigate();
  const { search } = useLocation();
  const close = () => navigate(factoryHomePath(organizationId, routeSegment, lineId));
  const intakeId = intakeIdFromSearch(search);
  const canvasId = columnAutomationViewCanvasIdFromSearch(search);
  const planningTab = planningSettingsTabFromSearch(search);
  const settingsIntake = isIntakeSearchOpen(search)
    ? intakes.find((intake) => intake.intakeId === intakeId)
    : undefined;

  return (
    <>
      {isPlanningSearchOpen(search) ? (
        <PlanningSettingsHost
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={routeSegment}
          lineId={lineId}
          initialTab={planningSettingsTab(planningTab)}
          onClose={close}
        />
      ) : null}
      {canvasId ? (
        <ColumnAutomationViewHost
          key={canvasId}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={routeSegment}
          lineId={lineId}
          canvasId={canvasId}
          title={automationTitle(apps, canvasId)}
          onClose={close}
        />
      ) : null}
      {settingsIntake ? (
        <IntakeSettingsHost
          key={settingsIntake.intakeId}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={routeSegment}
          lineId={lineId}
          intake={settingsIntake}
          repository={onboarding?.backlogRepository}
          vcsIntegrationId={onboarding?.vcsIntegrationId}
          page
          initialTab={intakeTab(intakeSettingsTabFromSearch(search))}
          onClose={close}
        />
      ) : null}
      {isPRFeedbackSearchOpen(search) ? (
        <PRFeedbackSettingsHost
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={routeSegment}
          githubIntegrationId={onboarding?.vcsIntegrationId?.trim() ?? ""}
          vcsProvider={onboarding?.vcsProvider}
          repository={onboarding?.appRepository?.trim() ?? ""}
          lineId={lineId}
          canUpdate={canUpdate}
          handlerId={prFeedbackHandlerIdFromSearch(search)}
          initialTab={feedbackTab(prFeedbackSettingsTabFromSearch(search))}
          onClose={close}
        />
      ) : null}
    </>
  );
}

function automationTitle(apps: Array<{ id?: string; name?: string }>, canvasId: string): string {
  return apps.find((app) => app.id === canvasId)?.name?.trim() || "Automation";
}

function planningSettingsTab(value: string | null) {
  return isPlanningSettingsTab(value) ? value : "general";
}

function intakeTab(value: string | null): IntakeSettingsTab {
  return isIntakeSettingsTab(value) ? value : "general";
}

function feedbackTab(value: string | null): PRFeedbackSettingsTab {
  return isPRFeedbackSettingsTab(value) ? value : "general";
}
