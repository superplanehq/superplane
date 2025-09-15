import React from 'react';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import CloseAndCheckButtons from './CloseAndCheckButtons';

interface InlineEditorProps {
  isEditing: boolean;
  onSave: () => void;
  onCancel: () => void;
  onEdit: () => void;
  onDelete: () => void;
  displayName: string;
  badge?: React.ReactNode;
  editForm: React.ReactNode;
  className?: string;
}

export function InlineEditor({
  isEditing,
  onSave,
  onCancel,
  onEdit,
  onDelete,
  displayName,
  badge,
  editForm,
  className = ""
}: InlineEditorProps) {
  if (isEditing) {
    return (
      <div className={`border border-zinc-200 dark:border-zinc-700 rounded-lg p-3 space-y-3 ${className}`}>
        {editForm}
        <CloseAndCheckButtons onCancel={onCancel} onConfirm={onSave} />
      </div>
    );
  }

  return (
    <div className={`flex items-center justify-between py-3 px-2 bg-zinc-50 dark:bg-zinc-800 rounded ${className}`}>
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium text-gray-900 dark:text-zinc-100">{displayName}</span>
        {badge}
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={onEdit}
          className="text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300"
        >
          <MaterialSymbol name="edit" size="sm" />
        </button>
        <button
          onClick={onDelete}
          className="text-zinc-500 dark:text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-300"
        >
          <MaterialSymbol name="delete" size="sm" />
        </button>
      </div>
    </div>
  );
}