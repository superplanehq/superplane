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
  installInstructions: string;
  installCommands: string;
  prompt: string;
};

export function AgentSetupPromptDialog({
  open,
  onOpenChange,
  installInstructions,
  installCommands,
  prompt,
}: AgentSetupPromptDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className="flex max-h-[90vh] w-full max-w-3xl flex-col gap-0 overflow-hidden p-0"
        data-testid="agent-setup-prompt-dialog"
      >
        <DialogHeader className="border-b border-border px-5 py-4">
          <DialogTitle>Edit with a local agent</DialogTitle>
          <DialogDescription>
            Install the SuperPlane CLI in your terminal. Then copy the prompt into a local coding agent.
          </DialogDescription>
        </DialogHeader>
        <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-5 py-4">
          <PromptSection
            title="Install the SuperPlane CLI"
            helper="Run these commands in your terminal."
            body={installInstructions}
            testId="agent-setup-install-markdown"
          />
          <PromptSection
            title="Prompt"
            helper="This prompt includes canvas IDs and CLI commands to read and update the canvas."
            body={prompt}
            testId="agent-setup-prompt-markdown"
          />
        </div>
        <DialogFooter className="border-t border-border px-5 py-3">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          <CopyButton
            variant="button"
            buttonVariant="outline"
            text={installCommands}
            copiedLabel="Copied"
            data-testid="agent-setup-install-copy"
          >
            Copy install commands
          </CopyButton>
          <CopyButton
            variant="button"
            buttonVariant="default"
            text={prompt}
            copiedLabel="Copied"
            data-testid="agent-setup-prompt-copy"
          >
            Copy prompt
          </CopyButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function PromptSection({
  title,
  helper,
  body,
  testId,
}: {
  title: string;
  helper: string;
  body: string;
  testId: string;
}) {
  return (
    <section>
      <h3 className="text-sm font-medium text-foreground">{title}</h3>
      <p className="mt-1 text-[13px] leading-5 text-muted-foreground">{helper}</p>
      <MarkdownContent className="mt-3" content={body} variant="workspace" data-testid={testId} />
    </section>
  );
}
