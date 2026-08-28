import type { IntakeAutomationRun, IntakeSettingsTab } from "./intakeSourceSettingsModel";
import type {
  AddIntakeTemplate,
  ConfiguredLineIntakeSource,
  LineIntakeAnalyzingTicket,
  LineIntakeSource,
} from "./lineIntakeModel";

export interface LineIntakeDrawerProps {
  onClose: () => void;
  initialIntakeId?: string;
  initialSettingsOpen?: boolean;
  initialSettingsTab?: IntakeSettingsTab;
  configuredSources?: ConfiguredLineIntakeSource[];
  /** Shown under a source that has no runs yet, for Storybook and first run. */
  analyzingTickets?: LineIntakeAnalyzingTicket[];
  onOpenTicket?: (ticket: LineIntakeAnalyzingTicket) => void;
  onSelectIntakeTemplate?: (template: AddIntakeTemplate) => void;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  editAutomationHref?: string;
  editAutomationHrefFor?: (intake: ConfiguredLineIntakeSource) => string | undefined;
  /** Preview an unconfigured source, used by the picker in Storybook. */
  previewSource?: LineIntakeSource;
  onSettingsSaved?: () => void;
  /** Show Add intake in Storybook. Hidden on the board until the flow is ready. */
  showAddIntakeControl?: boolean;
  /** Bump to highlight the window and expand a collapsed intake. */
  focusNonce?: number;
}

export interface LineIntakeDrawerPopupsProps {
  pickerOpen: boolean;
  initialSettingsTab: IntakeSettingsTab;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  settingsIntake?: ConfiguredLineIntakeSource;
  editAutomationHref?: string;
  previewSource?: LineIntakeSource;
  previewAppId?: string;
  openTicket: LineIntakeAnalyzingTicket | null;
  onClosePicker: () => void;
  onSelectTemplate: (template: AddIntakeTemplate) => void;
  onOpenRun: (run: IntakeAutomationRun) => void;
  onSettingsSaved?: () => void;
  onCloseSettings: () => void;
  onClosePreview: () => void;
  onCloseOpenTicket: () => void;
}
