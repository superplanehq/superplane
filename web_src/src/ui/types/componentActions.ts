// Shared interface for component action handlers

export interface ComponentActionsProps {
  runDisabled?: boolean;
  runDisabledTooltip?: string;
  onTogglePause?: () => void;
  onDuplicate?: () => void;
  onEdit?: () => void;
  onDeactivate?: () => void;
  onToggleView?: () => void;
  onDelete?: () => void;
  isCompactView?: boolean;
}
