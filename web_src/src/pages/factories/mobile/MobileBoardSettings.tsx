import type { FactoriesFactory } from "@/api-client";
import { useLocation, useNavigate } from "react-router";

import {
  factoryHomePath,
  intakeIdFromSearch,
  intakeSettingsTabFromSearch,
  isIntakeSearchOpen,
  isPRFeedbackSearchOpen,
  prFeedbackHandlerIdFromSearch,
  prFeedbackSettingsTabFromSearch,
} from "../lib/factoryPagePaths";
import { IntakeSettingsHost } from "../pages/IntakeSettingsHost";
import { isIntakeSettingsTab, type IntakeSettingsTab } from "../pages/intakeSourceSettingsModel";
import type { ConfiguredLineIntakeSource } from "../pages/lineIntakeModel";
import { PRFeedbackSettingsHost } from "../pages/PRFeedbackSettingsHost";
import { isPRFeedbackSettingsTab, type PRFeedbackSettingsTab } from "../pages/prFeedbackSettingsModel";

type FactoryOnboarding = NonNullable<FactoriesFactory["onboarding"]>;

/** Intake and pull request settings opened from the board menu URL. */
export function MobileBoardSettings({
  organizationId,
  factoryId,
  routeSegment,
  lineId,
  intakes,
  canUpdate,
  onboarding,
}: {
  organizationId: string;
  factoryId: string;
  routeSegment: string;
  lineId: string;
  intakes: ConfiguredLineIntakeSource[];
  canUpdate: boolean;
  onboarding?: FactoryOnboarding;
}) {
  const navigate = useNavigate();
  const { search } = useLocation();
  const close = () => navigate(factoryHomePath(organizationId, routeSegment, lineId));
  const intakeId = intakeIdFromSearch(search);
  const settingsIntake = isIntakeSearchOpen(search)
    ? intakes.find((intake) => intake.intakeId === intakeId)
    : undefined;

  return (
    <>
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

function intakeTab(value: string | null): IntakeSettingsTab {
  return isIntakeSettingsTab(value) ? value : "general";
}

function feedbackTab(value: string | null): PRFeedbackSettingsTab {
  return isPRFeedbackSettingsTab(value) ? value : "general";
}
