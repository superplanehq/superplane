import React from 'react';
import { Button } from '@/components/Button/button';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';

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
        <div className="flex justify-end gap-2 pt-2">
          <Button outline onClick={onCancel}>
            Cancel
          </Button>
          <Button color="blue" onClick={onSave}>
            <MaterialSymbol name="save" size="sm" data-slot="icon" />
            Save
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className={`flex items-center justify-between p-2 hover:bg-zinc-50 dark:hover:bg-zinc-800 rounded ${className}`}>
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium">{displayName}</span>
        {badge}
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={onEdit}
          className="text-zinc-500 hover:text-zinc-700"
        >
          <span className="material-symbols-outlined text-sm">edit</span>
        </button>
        <button
          onClick={onDelete}
          className="text-red-600 hover:text-red-700"
        >
          <span className="material-symbols-outlined text-sm">delete</span>
        </button>
      </div>
    </div>
  );
}