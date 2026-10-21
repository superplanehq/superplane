import type { LineIntakeSourceId } from "./lineIntakeModel";
import { INTAKE_SETTINGS_COPY, type IntakeSettingsTab } from "./intakeSourceSettingsModel";
import { FactoryAutomationSettingsTabs } from "./FactoryAutomationSettingsTabs";
import { IntakeSourceTitleLogo } from "./IntakeSourceTitleLogo";
import { PopupHeader } from "./work-order-popup-redesign/popupShared";

export function IntakeSettingsHeader({
  name,
  sourceId,
  history,
  hasAgent,
  onClose,
}: {
  name: string;
  sourceId: LineIntakeSourceId;
  history: { tab: IntakeSettingsTab; setTab: (tab: IntakeSettingsTab) => void; enabled: boolean };
  hasAgent: boolean;
  onClose: () => void;
}) {
  return (
    <PopupHeader title={`Intake ${name}`} onClose={onClose} leading={<IntakeSourceTitleLogo sourceId={sourceId} />}>
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
