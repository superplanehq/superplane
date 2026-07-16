import React from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogTitle } from "@/components/ui/dialog";
import type { ExpressionEditorDialogProps } from "./types";

// Shared expandable dialog for large expression editors. The dialog draft
// is isolated: parent updates are ignored until the user presses Save.
export const ExpressionEditorDialog: React.FC<ExpressionEditorDialogProps> = ({
  open,
  onOpenChange,
  ...sessionProps
}) => (
  <Dialog open={open} onOpenChange={onOpenChange}>
    {open ? <ExpressionEditorDialogSession onOpenChange={onOpenChange} {...sessionProps} /> : null}
  </Dialog>
);

type ExpressionEditorDialogSessionProps = Omit<ExpressionEditorDialogProps, "open">;

const ExpressionEditorDialogSession: React.FC<ExpressionEditorDialogSessionProps> = ({
  onOpenChange,
  title,
  initialValue,
  onSave,
  testId,
  children,
  headerActions,
}) => {
  const [draft, setDraft] = React.useState(initialValue);

  const handleSave = () => {
    onSave(draft);
    onOpenChange(false);
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  // Suggestion portals live on document.body; Radix would otherwise treat
  // clicking a suggestion as an outside interaction and dismiss the dialog.
  const handleInteractOutside = (event: Event) => {
    const target = event.target as Element | null;
    if (target?.closest?.("[data-autocomplete-suggestions]")) {
      event.preventDefault();
    }
  };

  const handleEscapeKeyDown = (event: KeyboardEvent) => {
    const target = event.target as Element | null;
    const autocompleteSuggestionsAreOpen = target?.closest(
      "[data-autocomplete-input][data-autocomplete-suggestions-open]",
    );
    const monacoEditor = target?.closest(".monaco-editor");
    const monacoSuggestionsAreOpen = monacoEditor?.querySelector(".suggest-widget.visible");
    if (autocompleteSuggestionsAreOpen || monacoSuggestionsAreOpen) {
      event.preventDefault();
    }
  };

  return (
    <DialogContent
      size="90vw"
      className="flex flex-col gap-0 overflow-hidden p-0"
      onClick={(e) => e.stopPropagation()}
      onEscapeKeyDown={handleEscapeKeyDown}
      onPointerDownOutside={handleInteractOutside}
      onFocusOutside={handleInteractOutside}
      onInteractOutside={handleInteractOutside}
      data-testid={testId}
    >
      <div className="flex shrink-0 items-center justify-between gap-2 border-b border-gray-200 px-4 py-3 pr-12 dark:border-gray-600">
        <DialogTitle className="truncate">{title}</DialogTitle>
        <DialogDescription className="sr-only">
          Expanded editor for {title}. Save to apply your changes or cancel to discard them.
        </DialogDescription>
        {headerActions ? <div className="flex items-center gap-2">{headerActions({ draft })}</div> : null}
      </div>
      <div className="flex min-h-0 flex-1 flex-col overflow-hidden px-4 py-3">
        {children({ value: draft, onChange: setDraft })}
      </div>
      <DialogFooter className="shrink-0 border-t border-gray-200 px-4 py-3 dark:border-gray-600">
        <Button type="button" variant="outline" onClick={handleCancel} data-testid="expandable-editor-cancel">
          Cancel
        </Button>
        <Button type="button" onClick={handleSave} data-testid="expandable-editor-save">
          Save
        </Button>
      </DialogFooter>
    </DialogContent>
  );
};
