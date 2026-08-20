export function duplicateLineName(name: string | undefined) {
  const trimmed = name?.trim() || "Unnamed line";
  return `${trimmed} copy`;
}

export type LineCardActions = {
  onEdit: () => void;
  onDuplicate: () => void | Promise<void>;
  canEdit: boolean;
  canDuplicate: boolean;
  isDuplicating?: boolean;
};
