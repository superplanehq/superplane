import { useState } from 'react';
import Tippy from '@tippyjs/react';
import 'tippy.js/dist/tippy.css';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { YamlCodeEditor } from '@/pages/canvas/components/YamlCodeEditor';

interface NodeActionButtonsProps {
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

export function NodeActionButtons({
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
}: NodeActionButtonsProps) {
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
          className="action-buttons absolute z-50 text-sm -top-13 left-1/2 transform -translate-x-1/2 flex bg-white dark:bg-zinc-800 shadow-lg rounded-lg border border-gray-200 dark:border-zinc-700 z-50"
          onClick={(e) => e.stopPropagation()}
        >
          <Tippy content="View code" placement="top" theme="dark" arrow>
            <button
              onClick={handleCodeClick}
              className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-l-md transition-colors"
            >
              <MaterialSymbol name="code" size="lg" />
            </button>
          </Tippy>

          <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600" />

          <Tippy content="Save" placement="top" theme="dark" arrow>
            <button
              onClick={() => onSave()}
              className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 transition-colors"
            >
              <MaterialSymbol name="check" size="lg" />
            </button>
          </Tippy>

          <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600" />

          <Tippy content="Cancel" placement="top" theme="dark" arrow>
            <button
              onClick={isNewNode ? onDiscard : onCancel}
              className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-r-md transition-colors"
            >
              <MaterialSymbol name="close" size="lg" />
            </button>
          </Tippy>
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
      className="action-buttons absolute z-50 text-sm -top-13 left-1/2 transform -translate-x-1/2 flex bg-white dark:bg-zinc-800 shadow-lg rounded-lg border border-gray-200 dark:border-zinc-700 z-50"
      onClick={(e) => e.stopPropagation()}
    >
      {onSend && (
        <>
          <Tippy content="Manually emit an event" placement="top" theme="dark" arrow>
            <button
              onClick={onSend}
              className={`flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 transition-colors rounded-l-md`}
            >
              <MaterialSymbol name="send" size="lg" />
            </button>
          </Tippy>
        </>
      )}

      {onSend && <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600" />}

      <Tippy content="Edit" placement="top" theme="dark" arrow>
        <button
          onClick={onEdit}
          className={`flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 transition-colors ${
            !onSend ? 'rounded-l-md' : ''
          }`}
        >
          <MaterialSymbol name="edit" size="lg" />
        </button>
      </Tippy>

      {onDuplicate && (
        <>
          <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600" />
          <Tippy content="Duplicate" placement="top" theme="dark" arrow>
            <button
              onClick={onDuplicate}
              className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 transition-colors"
            >
              <MaterialSymbol name="content_copy" size="lg" />
            </button>
          </Tippy>
        </>
      )}

      <div className="w-px h-8 bg-gray-300 dark:bg-zinc-600" />
      <Tippy content="Delete" placement="top" theme="dark" arrow>
        <button
          onClick={onDiscard}
          className="flex font-semibold items-center justify-center w-8 h-8 text-gray-900 dark:text-zinc-100 hover:text-gray-800 dark:hover:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded-r-md transition-colors focus:outline-none"
        >
          <MaterialSymbol name="delete" size="lg" />
        </button>
      </Tippy>
    </div>
  );
}