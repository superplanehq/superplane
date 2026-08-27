import type { IntakeSettingsTab } from "./intakeSourceSettingsModel";
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
  editAutomationHref?: string;
  editAutomationHrefFor?: (intake: ConfiguredLineIntakeSource) => string | undefined;
  /** Preview an unconfigured source, used by the picker in Storybook. */
  previewSource?: LineIntakeSource;
  onSettingsSaved?: () => void;
  /** Show Add intake in Storybook. Hidden on the board until the flow is ready. */
  showAddIntakeControl?: boolean;
}
