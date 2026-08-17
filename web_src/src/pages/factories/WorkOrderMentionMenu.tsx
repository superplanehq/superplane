import { createPortal } from "react-dom";
import { Avatar } from "@/components/Avatar/avatar";
import { cn } from "@/lib/utils";
import type { WorkOrderMentionOption } from "./lib/workOrderMentionOptions";

interface WorkOrderMentionMenuProps {
  options: WorkOrderMentionOption[];
  position: { top: number; left: number };
  highlightedIndex: number;
  onHighlightChange: (index: number) => void;
  onSelect: (option: WorkOrderMentionOption) => void;
}

/**
 * Floating, keyboard-navigable member picker anchored under the `@` caret in
 * `WorkOrderCommentComposer`. Purely presentational — filtering and keyboard
 * handling live in the composer so both it and this menu read from the same
 * `useWorkOrderMentionOptions` list.
 */
export function WorkOrderMentionMenu({
  options,
  position,
  highlightedIndex,
  onHighlightChange,
  onSelect,
}: WorkOrderMentionMenuProps) {
  if (options.length === 0) {
    return null;
  }

  return createPortal(
    <div
      id="work-order-mention-menu"
      role="listbox"
      aria-label="Mention a member"
      data-testid="work-order-mention-menu"
      style={{ position: "fixed", top: position.top, left: position.left, zIndex: 60 }}
      className="w-64 overflow-hidden rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-gray-700 dark:bg-gray-800"
    >
      {options.map((option, index) => (
        <button
          key={option.id}
          type="button"
          role="option"
          aria-selected={index === highlightedIndex}
          data-testid="work-order-mention-option"
          className={cn(
            "flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm text-gray-900 dark:text-gray-100",
            index === highlightedIndex ? "bg-gray-100 dark:bg-gray-700" : "hover:bg-gray-50 dark:hover:bg-gray-700/60",
          )}
          onMouseEnter={() => onHighlightChange(index)}
          onMouseDown={(event) => {
            // Prevent the textarea from losing focus/selection before onSelect runs.
            event.preventDefault();
            onSelect(option);
          }}
        >
          <Avatar src={option.avatarUrl} initials={option.initials} alt={option.name} className="size-5" />
          <span className="truncate">{option.name}</span>
        </button>
      ))}
    </div>,
    document.body,
  );
}
