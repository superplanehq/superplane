import { Mic, Square } from "lucide-react";
import { useEffect } from "react";

import { Button } from "@/components/ui/button";
import type { UseSpeechDictationResult } from "@/hooks/useSpeechDictation";
import { showErrorToast } from "@/lib/toast";
import { cn } from "@/lib/utils";

export type DictateCopy = {
  dictate: string;
  stopDictation: string;
  microphoneDenied: string;
};

export interface DictateButtonProps {
  dictation: UseSpeechDictationResult;
  copy: DictateCopy;
  disabled?: boolean;
}

export function DictateButton({ dictation, copy, disabled = false }: DictateButtonProps) {
  useEffect(() => {
    if (dictation.permissionError) {
      showErrorToast(copy.microphoneDenied);
    }
  }, [copy.microphoneDenied, dictation.permissionError]);

  if (!dictation.isSupported) {
    return null;
  }

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className={cn(
        "size-8 rounded-full text-muted-foreground",
        dictation.isListening &&
          "text-destructive bg-destructive/15 ring-2 ring-destructive animate-pulse hover:bg-destructive/15 hover:text-destructive dark:hover:bg-destructive/15 dark:hover:text-destructive motion-reduce:animate-none",
      )}
      disabled={disabled}
      aria-label={dictation.isListening ? copy.stopDictation : copy.dictate}
      aria-pressed={dictation.isListening}
      data-testid="dictate-button"
      onClick={() => {
        if (dictation.isListening) {
          dictation.stop();
          return;
        }
        dictation.start();
      }}
    >
      {dictation.isListening ? (
        <Square className="size-3 fill-current" aria-hidden data-testid="dictate-stop-icon" />
      ) : (
        <Mic className="size-4" aria-hidden data-testid="dictate-mic-icon" />
      )}
    </Button>
  );
}
