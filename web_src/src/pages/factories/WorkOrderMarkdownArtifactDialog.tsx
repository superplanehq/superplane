import { X } from "lucide-react";
import { useEffect } from "react";
import { Dialog, DialogBody, DialogTitle } from "@/components/Dialog/dialog";
import { MarkdownContent } from "@/pages/app/Markdown";

interface WorkOrderMarkdownArtifactDialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  body: string;
}

export function WorkOrderMarkdownArtifactDialog({ open, onClose, title, body }: WorkOrderMarkdownArtifactDialogProps) {
  // The shared Dialog primitive only dismisses on backdrop click, which
  // leaves keyboard users stranded. Wire Escape + an explicit X here so
  // we don't have to touch the primitive (used by other dialogs).
  useEffect(() => {
    if (!open) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open, onClose]);

  return (
    <Dialog open={open} onClose={onClose} size="2xl" className="text-left">
      <button
        type="button"
        aria-label="Close"
        onClick={onClose}
        className="absolute top-4 right-4 text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
      >
        <X className="h-5 w-5" />
      </button>
      <DialogTitle>{title}</DialogTitle>
      <DialogBody>
        {body.trim() ? (
          <MarkdownContent content={body} />
        ) : (
          <p className="text-sm text-gray-500 dark:text-gray-400">This note is empty.</p>
        )}
      </DialogBody>
    </Dialog>
  );
}
