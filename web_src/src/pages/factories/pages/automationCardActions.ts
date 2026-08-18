export function duplicateAutomationName(name: string | undefined) {
  const trimmed = name?.trim() || "Unnamed automation";
  return `${trimmed} copy`;
}

export type AutomationCardActions = {
  onEdit: () => void;
  onDuplicate: () => void | Promise<void>;
  onDelete: () => void | Promise<void>;
  canEdit: boolean;
  canDuplicate: boolean;
  canDelete: boolean;
  isDuplicating?: boolean;
  isDeleting?: boolean;
};
