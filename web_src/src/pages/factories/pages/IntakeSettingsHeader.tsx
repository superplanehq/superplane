import type { LineIntakeSourceId } from "./lineIntakeModel";
import { INTAKE_SETTINGS_COPY, intakeSourceSettingsTitle, type IntakeSettingsTab } from "./intakeSourceSettingsModel";
import { FactoryAutomationSettingsTabs } from "./FactoryAutomationSettingsTabs";
import { IntakeSourceTitleLogo } from "./IntakeSourceTitleLogo";
import { PopupHeader } from "./work-order-popup-redesign/popupShared";

export function IntakeSettingsHeader({
  name,
  sourceId,
  repository,
  history,
  hasAgent,
  onClose,
}: {
  name: string;
  sourceId: LineIntakeSourceId;
  repository?: string;
  history: { tab: IntakeSettingsTab; setTab: (tab: IntakeSettingsTab) => void; enabled: boolean };
  hasAgent: boolean;
  onClose: () => void;
}) {
  return (
    <PopupHeader
      title={intakeSourceSettingsTitle(sourceId, name)}
      onClose={onClose}
      leading={<IntakeSourceTitleLogo sourceId={sourceId} />}
    >
      {sourceId === "github-issues" && repository ? (
        <p className="mt-1 text-sm text-muted-foreground">{repository}</p>
      ) : null}
      <FactoryAutomationSettingsTabs
        tab={history.tab}
        onTabChange={history.setTab}
        hasAgent={hasAgent}
        historyEnabled={history.enabled}
        label={INTAKE_SETTINGS_COPY.tabsLabel}
        testId="intake-settings"
      />
    </PopupHeader>
  );
}
