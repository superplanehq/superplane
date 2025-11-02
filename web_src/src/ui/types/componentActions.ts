// Shared interface for component action handlers

export interface ComponentActionsProps {
  onRun?: () => void;
  // When true, shows Run as disabled with tooltip
  runDisabled?: boolean;
  runDisabledTooltip?: string;
  onDuplicate?: () => void;
  onEdit?: () => void;
  onConfigure?: () => void;
  onDeactivate?: () => void;
  onToggleView?: () => void;
  onDelete?: () => void;
  onToggleCollapse?: () => void;
  isCompactView?: boolean;
}
