// Shared interface for component action handlers

export interface ComponentActionsProps {
  runDisabled?: boolean;
  runDisabledTooltip?: string;
  onTogglePause?: () => void;
  onToggleOnError?: () => void;
  isOnErrorNode?: boolean;
  onDuplicate?: () => void;
  onEdit?: () => void;
  onDeactivate?: () => void;
  onToggleView?: () => void;
  onShowDiff?: () => void;
  onDelete?: () => void;
  isCompactView?: boolean;
}
