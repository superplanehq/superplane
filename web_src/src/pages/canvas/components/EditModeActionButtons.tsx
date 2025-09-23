import { useState } from 'react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { YamlCodeEditor } from './YamlCodeEditor';

interface EditModeActionButtonsProps {
  onSave: () => void;
  onCancel: () => void;
  onDiscard?: () => void;
  onEdit?: () => void; // For non-edit mode
  onDuplicate?: () => void; // For duplicating the node
  onSend?: () => void; // For manually emitting events
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
  onSend,
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
          className="action-buttons absolute z-50 text-sm -top-13 left-1/2 transform -translate-x-1/2 flex gap-2 bg-white dark:bg-zinc-800 shadow-lg rounded-lg px-2 py-[2px] border border-gray-200 dark:border-zinc-700 z-50"
          onClick={(e) => e.stopPropagation()}
        >
          <button
            onClick={handleCodeClick}
            className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
            title="View code"
          >
            <MaterialSymbol name="code" size="lg" />
          </button>

          <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600 self-center" />

          <button
            onClick={() => onSave()}
            className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
            title="Save"
          >
            <MaterialSymbol name="check" size="lg" />
          </button>

          <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600 self-center" />

          <button
            onClick={isNewNode ? onDiscard : onCancel}
            className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
            title="Cancel"
          >
            <MaterialSymbol name="close" size="lg" />
          </button>

          {!isNewNode && onDuplicate && (
            <>
              <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600 self-center" />
              <button
              onClick={onDuplicate}
              className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
              title="Duplicate"
            >
              <MaterialSymbol name="content_copy" size="lg" />
              </button>
            </>
          )}

          {!isNewNode && (
            <>
              <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600 self-center" />
              <button
              onClick={onDiscard}
              className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-md transition-colors focus:outline-none"
              title="Delete"
            >
              <MaterialSymbol name="delete" size="lg" />
              </button>
            </>
          )}

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
      className="action-buttons absolute z-50 text-sm -top-13 left-1/2 transform -translate-x-1/2 flex gap-2 bg-white dark:bg-zinc-800 shadow-lg rounded-lg px-2 py-[2px] border border-gray-200 dark:border-zinc-700 z-50"
      onClick={(e) => e.stopPropagation()}
    >
      <button
        onClick={onEdit}
        className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
        title="Edit"
      >
        <MaterialSymbol name="edit" size="lg" />
      </button>

      {onSend && (
        <>
          <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600 self-center" />
          <button
          onClick={onSend}
          className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
          title="Manually emit an event"
        >
          <MaterialSymbol name="send" size="lg" />
          </button>
        </>
      )}

      {onDuplicate && (
        <>
          <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600 self-center" />
          <button
          onClick={onDuplicate}
          className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-md transition-colors"
          title="Duplicate"
        >
          <MaterialSymbol name="content_copy" size="lg" />
          </button>
        </>
      )}

      <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600 self-center" />

      <button
        onClick={onDiscard}
        className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-red-600 dark:hover:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-md transition-colors focus:outline-none"
        title="Delete"
      >
        <MaterialSymbol name="delete" size="lg" />
      </button>
    </div>
  );
}