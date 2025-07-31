import { useState } from 'react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Dropdown, DropdownButton, DropdownItem, DropdownLabel, DropdownMenu } from '@/components/Dropdown/dropdown';
import { YamlCodeEditor } from './YamlCodeEditor';

interface EditModeActionButtonsProps {
  onSave: (saveAsDraft: boolean) => void;
  onCancel: () => void;
  onDiscard?: () => void;
  entityType?: string; // e.g., "stage", "connection group", "event source"
  entityData?: unknown; // The current entity data for YAML editing
  onYamlApply?: (updatedData: unknown) => void; // Callback when YAML changes are applied
}

export function EditModeActionButtons({
  onSave,
  onCancel,
  onDiscard,
  entityType = "item",
  entityData,
  onYamlApply
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

  return (
    <>
      <div
        className="action-buttons absolute z-50 text-sm -top-13 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white shadow-lg rounded-lg px-2 py-[2px] border border-gray-200 z-50"
        onClick={(e) => e.stopPropagation()}
      >
        <button
          onClick={handleCodeClick}
          className="flex font-semibold items-center gap-2 px-3 py-2 hover:text-gray-800 hover:bg-gray-100 rounded-md transition-colors"
          title="View code"
        >
          <MaterialSymbol name="code" size="md" />
          Code
        </button>

      <Dropdown>
        <DropdownButton plain className='flex items-center gap-2'>
          <MaterialSymbol name="save" size="md" />
          Save
          <MaterialSymbol name="expand_more" size="md" />
        </DropdownButton>
        <DropdownMenu anchor="bottom start">
          <DropdownItem className='flex items-center gap-2' onClick={() => onSave(false)}>
            <DropdownLabel>Save & Commit</DropdownLabel>
          </DropdownItem>
          <DropdownItem className='flex items-center gap-2' onClick={() => onSave(true)}>
            <DropdownLabel>Save as Draft</DropdownLabel>
          </DropdownItem>
          <DropdownItem className='flex items-center gap-2' onClick={onCancel}>
            <DropdownLabel>Discard {entityType}</DropdownLabel>
          </DropdownItem>
        </DropdownMenu>
      </Dropdown>

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