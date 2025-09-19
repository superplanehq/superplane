import { useState } from 'react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Dropdown, DropdownButton, DropdownItem, DropdownLabel, DropdownMenu } from '@/components/Dropdown/dropdown';
import { YamlCodeEditor } from './YamlCodeEditor';

interface EditModeActionButtonsProps {
  onSave: () => void;
  onCancel: () => void;
  onDiscard?: () => void;
  onEdit?: () => void; // For non-edit mode
  onDuplicate?: () => void; // For duplicating the node
  entityType?: string; // e.g., "stage", "connection group", "event source"
  entityData?: unknown; // The current entity data for YAML editing
  onYamlApply?: (updatedData: unknown) => void; // Callback when YAML changes are applied
  isEditMode: boolean; // To determine which buttons to show
  isNewNode?: boolean;
}

export function EditModeActionButtons({
  onSave,
  onCancel,
  onDiscard,
  onEdit,
  onDuplicate,
  entityType = "item",
  entityData,
  onYamlApply,
  isEditMode,
  isNewNode
}: EditModeActionButtonsProps) {
  const [isCodeEditorOpen, setIsCodeEditorOpen] = useState(false);

  const handleCodeClick = () => {
    setIsCodeEditorOpen(true);
  };

  const handleYamlApply = (updatedData: unknown) => {
    if (onYamlApply) {
      onYamlApply(updatedData);
    }
    setIsCodeEditorOpen(false);
  };

  if (isEditMode) {
    return (
      <>
        <div
          className="action-buttons absolute z-50 text-sm -top-13 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white dark:bg-zinc-800 shadow-lg rounded-lg px-2 py-[2px] border border-gray-200 dark:border-zinc-700 z-50"
          onClick={(e) => e.stopPropagation()}
        >
          <button
            onClick={handleCodeClick}
            className="flex font-semibold items-center gap-2 px-3 py-2 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
            title="View code"
          >
            <MaterialSymbol name="code" size="md" />
            Code
          </button>

          <button
            onClick={() => onSave()}
            className="flex font-semibold items-center gap-2 px-3 py-2 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
            title="Save"
          >
            <MaterialSymbol name="save" size="md" />
            Save
          </button>

          <button
            onClick={isNewNode ? onDiscard : onCancel}
            className="flex font-semibold items-center gap-2 px-3 py-2 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
            title="Cancel"
          >
            <MaterialSymbol name="cancel" size="md" />
            Cancel
          </button>

          {!isNewNode && onDuplicate && (
            <button
              onClick={onDuplicate}
              className="flex font-semibold items-center gap-2 px-3 py-2 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
              title="Duplicate"
            >
              <MaterialSymbol name="content_copy" size="md" />
              Duplicate
            </button>
          )}

          {!isNewNode && <Dropdown>
            <DropdownButton plain className='flex items-center gap-2'>
              <MaterialSymbol name="more_vert" size="md" />
            </DropdownButton>
            <DropdownMenu anchor="bottom start">
              <DropdownItem className='flex items-center gap-2' onClick={onDiscard}>
                <MaterialSymbol name="delete" size="md" />
                <DropdownLabel>Delete</DropdownLabel>
              </DropdownItem>
            </DropdownMenu>
          </Dropdown>}

        </div>

        {isCodeEditorOpen && entityData && (
          <YamlCodeEditor
            isOpen={isCodeEditorOpen}
            onClose={() => setIsCodeEditorOpen(false)}
            entityType={entityType}
            entityData={entityData}
            onApply={handleYamlApply}
          />
        )}
      </>
    );
  }

  return (
    <div
      className="action-buttons absolute z-50 text-sm -top-13 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white dark:bg-zinc-800 shadow-lg rounded-lg px-2 py-[2px] border border-gray-200 dark:border-zinc-700 z-50"
      onClick={(e) => e.stopPropagation()}
    >
      <button
        onClick={onEdit}
        className="flex font-semibold items-center gap-2 px-3 py-2 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
        title="Edit"
      >
        <MaterialSymbol name="edit" size="md" />
        Edit
      </button>

      {onDuplicate && (
        <button
          onClick={onDuplicate}
          className="flex font-semibold items-center gap-2 px-3 py-2 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
          title="Duplicate"
        >
          <MaterialSymbol name="content_copy" size="md" />
          Duplicate
        </button>
      )}

      <Dropdown>
        <DropdownButton plain className='flex items-center gap-2'>
          <MaterialSymbol name="more_vert" size="md" />
        </DropdownButton>
        <DropdownMenu anchor="bottom start">
          <DropdownItem className='flex items-center gap-2' onClick={onDiscard}>
            <MaterialSymbol name="delete" size="md" />
            <DropdownLabel>Delete</DropdownLabel>
          </DropdownItem>
        </DropdownMenu>
      </Dropdown>
    </div>
  );
}