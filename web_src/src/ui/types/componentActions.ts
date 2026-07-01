// Shared interface for component action handlers

export interface ComponentActionsProps {
  onDuplicate?: () => void;
  onToggleView?: () => void;
  onShowDiff?: () => void;
  onDelete?: () => void;
  isCompactView?: boolean;
}
