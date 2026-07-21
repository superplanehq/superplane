import React, { useMemo } from "react";
import { Copy, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { NOTE_COLORS, type AnnotationColor } from "./noteColors";

export interface NoteActionsToolbarProps {
  activeColor: AnnotationColor;
  onColorChange: (color: AnnotationColor) => void;
  onDuplicate?: () => void;
  onDelete?: () => void;
}

/**
 * Hover toolbar rendered above a note in edit mode. Groups the color swatch
 * picker together with the duplicate and delete actions.
 */
export const NoteActionsToolbar: React.FC<NoteActionsToolbarProps> = ({
  activeColor,
  onColorChange,
  onDuplicate,
  onDelete,
}) => {
  const colorOptions = useMemo(
    () =>
      (Object.keys(NOTE_COLORS) as AnnotationColor[]).map((value) => ({
        value,
        dot: NOTE_COLORS[value].dot,
      })),
    [],
  );

  return (
    <>
      <div className="absolute -top-12 right-0 z-10 h-12 w-44 opacity-0" />
      <div className="absolute -top-8 right-0 z-10 hidden items-center gap-2 group-hover:flex nodrag">
        <div className="group/swatch relative flex items-center px-0.5 py-0.5">
          <button
            type="button"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
            }}
            className={cn("h-4 w-4 rounded-full border transition", NOTE_COLORS[activeColor].dot)}
            aria-label={`Current note color: ${NOTE_COLORS[activeColor].label}`}
          />
          <div className="absolute right-0 top-1/2 hidden -translate-y-1/2 items-center gap-2 pr-0.5 group-hover/swatch:flex">
            {colorOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  onColorChange(option.value);
                }}
                className={cn("h-4 w-4 rounded-full border transition", option.dot)}
                aria-label={`Set note color to ${NOTE_COLORS[option.value].label}`}
              />
            ))}
          </div>
        </div>
        {onDuplicate && (
          <button
            type="button"
            data-testid="node-action-duplicate"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onDuplicate();
            }}
            className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
            aria-label="Duplicate note"
          >
            <Copy size={16} />
          </button>
        )}
        {onDelete && (
          <button
            type="button"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onDelete();
            }}
            className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
            aria-label="Delete note"
          >
            <Trash2 size={16} />
          </button>
        )}
      </div>
    </>
  );
};
