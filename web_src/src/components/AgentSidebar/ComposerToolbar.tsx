import { memo, useRef } from "react";
import { ArrowUp, ImagePlus, Loader2, Square } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { AgentMode } from "./agentMode";
import { ALLOWED_IMAGE_TYPES } from "./useImageAttachments";
import { ModeToggle } from "./ModeToggle";

interface ComposerToolbarProps {
  agentMode: AgentMode;
  onModeSwitch: (mode: AgentMode) => void;
  modeDisabled?: boolean;
  sending: boolean;
  stopping?: boolean;
  statusLabel: string;
  canSend: boolean;
  canAttach: boolean;
  onStop: () => void;
  onSend: () => void;
  onAddFiles: (files: FileList | File[]) => void;
}

function AttachImageButton({
  canAttach,
  onAddFiles,
}: {
  canAttach: boolean;
  onAddFiles: (files: FileList | File[]) => void;
}) {
  const fileInputRef = useRef<HTMLInputElement>(null);

  return (
    <>
      <input
        ref={fileInputRef}
        type="file"
        accept={ALLOWED_IMAGE_TYPES.join(",")}
        multiple
        className="hidden"
        data-testid="agent-image-input"
        onChange={(event) => {
          if (event.target.files && event.target.files.length > 0) {
            onAddFiles(event.target.files);
          }
          event.target.value = "";
        }}
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-7 shrink-0 rounded-full text-slate-600 hover:bg-slate-100 hover:text-slate-900"
        onClick={() => fileInputRef.current?.click()}
        disabled={!canAttach}
        aria-label="Attach image"
        title="Attach image"
        data-testid="agent-attach-image-button"
      >
        <ImagePlus className="size-3.5" aria-hidden />
      </Button>
    </>
  );
}

export const ComposerToolbar = memo(function ComposerToolbar({
  agentMode,
  onModeSwitch,
  modeDisabled,
  sending,
  stopping,
  statusLabel,
  canSend,
  canAttach,
  onStop,
  onSend,
  onAddFiles,
}: ComposerToolbarProps) {
  return (
    <div className="flex items-center justify-between gap-2 px-2 pb-2">
      <div className="flex min-w-0 items-center gap-1">
        <AttachImageButton canAttach={canAttach} onAddFiles={onAddFiles} />
        <ModeToggle mode={agentMode} onSwitch={onModeSwitch} disabled={modeDisabled} streaming={sending} />
      </div>
      <div className="flex min-w-0 shrink-0 items-center gap-2">
        <span className="truncate text-xs text-muted-foreground">{statusLabel}</span>
        {sending && (
          <Button
            type="button"
            variant="outline"
            size="icon"
            className="size-7 shrink-0 rounded-full border-slate-300 bg-white text-slate-700 hover:bg-slate-100"
            onClick={onStop}
            disabled={stopping}
            aria-label={stopping ? "Stopping" : "Stop"}
            title={stopping ? "Stopping..." : "Stop"}
            data-testid="agent-stop-button"
          >
            {stopping ? (
              <Loader2 className="size-3.5 animate-spin" aria-hidden />
            ) : (
              <Square className="size-3 fill-current" aria-hidden />
            )}
          </Button>
        )}
        <Button
          type="button"
          variant="default"
          size="icon"
          className="size-7 shrink-0 rounded-full"
          onClick={onSend}
          disabled={!canSend}
          aria-label="Send message"
          data-testid="agent-send-message-button"
        >
          <ArrowUp className="size-3.5" aria-hidden />
        </Button>
      </div>
    </div>
  );
});
