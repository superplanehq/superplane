import { CopyButton } from "@/ui/CopyButton";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { MarkdownContent } from "@/pages/app/Markdown";

type AgentSetupPromptDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  prompt: string;
};

export function AgentSetupPromptDialog({ open, onOpenChange, prompt }: AgentSetupPromptDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="flex max-h-[90vh] max-w-2xl flex-col gap-0 overflow-hidden p-0"
        data-testid="agent-setup-prompt-dialog"
      >
        <DialogHeader className="border-b border-border px-5 py-4">
          <DialogTitle>Set up a desktop agent</DialogTitle>
          <DialogDescription>
            Copy this prompt into a desktop coding agent. The prompt shows how to install the SuperPlane CLI and update
            this canvas.
          </DialogDescription>
        </DialogHeader>
        <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
          <MarkdownContent content={prompt} data-testid="agent-setup-prompt-markdown" />
        </div>
        <DialogFooter className="border-t border-border px-5 py-3">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          <CopyButton variant="button" text={prompt} copiedLabel="Copied" data-testid="agent-setup-prompt-copy">
            Copy prompt
          </CopyButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
