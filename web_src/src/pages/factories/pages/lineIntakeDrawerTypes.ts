import type { IntakeSettingsTab } from "./intakeSourceSettingsModel";
import type {
  AddIntakeTemplate,
  ConfiguredLineIntakeSource,
  LineIntakeAnalyzingTicket,
  LineIntakeSource,
  LineIntakeSourceId,
} from "./lineIntakeModel";

export interface LineIntakeDrawerProps {
  onClose: () => void;
  initialSourceId?: LineIntakeSourceId;
  initialSettingsOpen?: boolean;
  initialSettingsTab?: IntakeSettingsTab;
  sources?: LineIntakeSource[];
  configuredSources?: ConfiguredLineIntakeSource[];
  analyzingTickets?: LineIntakeAnalyzingTicket[];
  onOpenTicket?: (ticket: LineIntakeAnalyzingTicket) => void;
  onSelectIntakeTemplate?: (template: AddIntakeTemplate) => void;
  organizationId?: string;
  editAutomationHref?: string;
  editAutomationHrefFor?: (source: LineIntakeSource) => string | undefined;
  onSettingsSaved?: () => void;
}
