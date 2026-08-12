import React from "react";
import { resolveIcon } from "@/lib/utils";

type NodeHoverActionsProps = {
  showHeader?: boolean;
  onDuplicate?: () => void;
  onDelete?: () => void;
  onToggleView?: () => void;
  isCompactView?: boolean;
};

export function NodeHoverActions({
  showHeader = true,
  onDuplicate,
  onDelete,
  onToggleView,
  isCompactView,
}: NodeHoverActionsProps) {
  const DuplicateIcon = React.useMemo(() => resolveIcon("copy"), []);
  const DeleteIcon = React.useMemo(() => resolveIcon("trash-2"), []);
  const ToggleViewIcon = React.useMemo(
    () => resolveIcon(isCompactView ? "chevrons-up-down" : "chevrons-down-up"),
    [isCompactView],
  );

  if (!showHeader) {
    return null;
  }

  return (
    <>
      <div className="absolute -top-8 right-0 z-10 h-8 w-44 opacity-0" />
      <div className="absolute -top-8 right-0 z-10 hidden items-center gap-2 group-hover:flex nodrag">
        {onDuplicate ? (
          <button
            type="button"
            data-testid="node-action-duplicate"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onDuplicate();
            }}
            className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
          >
            <DuplicateIcon className="h-4 w-4" />
          </button>
        ) : null}
        {onToggleView ? (
          <button
            type="button"
            data-testid="node-action-toggle-view"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onToggleView();
            }}
            className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
          >
            <ToggleViewIcon className="h-4 w-4" />
          </button>
        ) : null}
        {onDelete ? (
          <button
            type="button"
            data-testid="node-action-delete"
            onClick={(event) => {
              event.preventDefault();
              event.stopPropagation();
              onDelete();
            }}
            className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
          >
            <DeleteIcon className="h-4 w-4" />
          </button>
        ) : null}
      </div>
    </>
  );
}
