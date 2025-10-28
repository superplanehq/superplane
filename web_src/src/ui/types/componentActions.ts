// Shared interface for component action handlers

export interface ComponentActionsProps {
  onRun?: () => void;
  onDuplicate?: () => void;
  onDeactivate?: () => void;
  onToggleView?: () => void;
  onDelete?: () => void;
  isCompactView?: boolean;
}