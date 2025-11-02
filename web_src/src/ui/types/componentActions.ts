// Shared interface for component action handlers

export interface ComponentActionsProps {
  onRun?: () => void;
  onDuplicate?: () => void;
  onEdit?: () => void;
  onConfigure?: () => void;
  onDeactivate?: () => void;
  onToggleView?: () => void;
  onDelete?: () => void;
  onToggleCollapse?: () => void;
  isCompactView?: boolean;
}
