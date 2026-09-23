import { Mic } from "lucide-react";
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
    <div className="flex min-w-0 items-center gap-2">
      {dictation.interimPhrase ? (
        <span className="truncate text-[12px] text-muted-foreground" aria-live="polite" data-testid="dictate-interim">
          {dictation.interimPhrase}
        </span>
      ) : null}
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className={cn(
          "size-8 rounded-full text-muted-foreground",
          dictation.isListening && "animate-pulse ring-2 ring-ring/40 motion-reduce:animate-none",
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
        <Mic className="size-4" aria-hidden />
      </Button>
    </div>
  );
}
