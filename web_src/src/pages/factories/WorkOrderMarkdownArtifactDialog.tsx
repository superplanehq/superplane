import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { MarkdownContent } from "@/pages/app/Markdown";

interface WorkOrderMarkdownArtifactDialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  body: string;
}

export function WorkOrderMarkdownArtifactDialog({ open, onClose, title, body }: WorkOrderMarkdownArtifactDialogProps) {
  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent
        size="large"
        className="max-h-[calc(100vh-2rem)] w-[calc(100%-2rem)] max-w-2xl overflow-y-auto text-left"
      >
        <DialogTitle>{title}</DialogTitle>
        <div className="mt-2">
          {body.trim() ? (
            <MarkdownContent content={body} />
          ) : (
            <p className="text-sm text-gray-500 dark:text-gray-400">This note is empty.</p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
