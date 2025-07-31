import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { Dropdown, DropdownButton, DropdownItem, DropdownLabel, DropdownMenu } from '@/components/Dropdown/dropdown';

interface EditModeActionButtonsProps {
  onSave: (saveAsDraft: boolean) => void;
  onCancel: () => void;
  onDiscard?: () => void;
  showDiscard?: boolean;
  entityType?: string; // e.g., "stage", "connection group", "event source"
}

export function EditModeActionButtons({ 
  onSave, 
  onCancel, 
  onDiscard, 
  showDiscard = false,
  entityType = "item"
}: EditModeActionButtonsProps) {
  return (
    <div
      className="action-buttons absolute z-50 -top-13 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white shadow-lg rounded-lg px-2 py-1 border border-gray-200 z-50"
      onClick={(e) => e.stopPropagation()}
    >
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
        </DropdownMenu>
      </Dropdown>

      <button
        onClick={onCancel}
        className="flex items-center gap-2 px-3 py-2 text-gray-600 hover:text-gray-800 hover:bg-gray-50 rounded-md transition-colors"
        title="Cancel changes"
      >
        <MaterialSymbol name="close" size="md" />
        Cancel
      </button>

      {showDiscard && onDiscard && (
        <button
          onClick={onDiscard}
          className="flex items-center gap-2 px-3 py-2 text-red-600 hover:text-red-800 hover:bg-red-50 rounded-md transition-colors"
          title={`Discard this ${entityType}`}
        >
          <MaterialSymbol name="delete" size="md" />
          Discard
        </button>
      )}
    </div>
  );
}