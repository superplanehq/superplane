import { Dialog, DialogBody, DialogTitle } from "@/components/Dialog/dialog";
import { MarkdownContent } from "@/pages/app/Markdown";

interface WorkOrderMarkdownArtifactDialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  body: string;
}

export function WorkOrderMarkdownArtifactDialog({ open, onClose, title, body }: WorkOrderMarkdownArtifactDialogProps) {
  return (
    <Dialog open={open} onClose={onClose} size="2xl" className="text-left">
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
