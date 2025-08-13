import React, { useState, useRef, useEffect } from 'react';

interface InlineEditableProps {
  value: string;
  onSave: (value: string) => void;
  placeholder?: string;
  className?: string;
  multiline?: boolean;
  isEditMode?: boolean;
  autoFocus?: boolean;
  dataTestId?: string;
}

export function InlineEditable({
  value,
  onSave,
  placeholder = 'Click to edit...',
  className = '',
  multiline = false,
  isEditMode = false,
  autoFocus = false,
  dataTestId
}: InlineEditableProps) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(value);
  const [isHovered, setIsHovered] = useState(false);
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement>(null);

  useEffect(() => {
    setEditValue(value);
  }, [value]);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      // Required to ensure the input is focused after the component is mounted
      setTimeout(() => {
        inputRef.current?.focus();
        inputRef.current?.select();
      }, 100);
    }
  }, [isEditing]);

  // Auto-focus for new items
  useEffect(() => {
    if (autoFocus && isEditMode && !value) {
      setIsEditing(true);
    }
  }, [autoFocus, isEditMode, value]);

  const handleClick = () => {
    if (isEditMode) {
      setIsEditing(true);
    }
  };

  const handleSave = () => {
    onSave(editValue);
    setIsEditing(false);
  };

  const handleCancel = () => {
    setEditValue(value);
    setIsEditing(false);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !multiline) {
      e.preventDefault();
      handleSave();
    } else if (e.key === 'Enter' && multiline && e.ctrlKey) {
      e.preventDefault();
      handleSave();
    } else if (e.key === 'Escape') {
      handleCancel();
    }
  };

  const handleBlur = () => {
    handleSave();
  };

  const displayValue = value || placeholder;
  const showHoverEffect = isEditMode && isHovered && !isEditing;

  if (isEditing) {
    if (multiline) {
      return (
        <textarea
          ref={inputRef as React.RefObject<HTMLTextAreaElement>}
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onKeyDown={handleKeyPress}
          onBlur={handleBlur}
          className={`${className} w-full border border-blue-300 dark:border-blue-600 rounded px-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-zinc-800 text-gray-900 dark:text-zinc-100`}
          placeholder={placeholder}
          rows={multiline ? 2 : undefined}
        />
      );
    }

    return (
      <input
        ref={inputRef as React.RefObject<HTMLInputElement>}
        value={editValue}
        onChange={(e) => setEditValue(e.target.value)}
        onKeyDown={handleKeyPress}
        onBlur={handleBlur}
        className={`${className} w-full border border-blue-300 rounded px-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent`}
        placeholder={placeholder}
        data-testid={dataTestId}
      />
    );
  }

  return (
    <div
      onClick={handleClick}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
      className={`${className} ${showHoverEffect ? 'bg-gray-100 dark:bg-zinc-700 rounded py-1' : ''} ${isEditMode ? 'cursor-pointer' : ''} transition-colors duration-200`}
      title={isEditMode ? 'Click to edit' : undefined}
    >
      {displayValue}
    </div>
  );
}