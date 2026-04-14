import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { ArrowUp, Square } from "lucide-react";
import type { FormEvent, KeyboardEvent, RefObject } from "react";

export type InputFormProps = {
  aiInputRef: RefObject<HTMLTextAreaElement | null>;
  aiInput: string;
  onAiInputChange: (value: string) => void;
  onSendPrompt: () => void;
  onStopPrompt: () => void;
  disabled: boolean;
  canvasId?: string;
  isGeneratingResponse: boolean;
  maxAiInputHeight: number;
  expanded?: boolean;
};

const TEXT_AREA_CLASSNAME = cn(
  "min-h-[20px] flex-1 resize-none border-0",
  "rounded-sm bg-transparent px-0.5 py-0.5 shadow-none",
  "focus-visible:ring-0 focus-visible:border-transparent",
);

const SUBMIT_BUTTON_CLASSNAME = cn(
  "p-1 rounded-full bg-slate-600 text-white hover:bg-slate-700",
  "cursor-pointer",
  "disabled:opacity-50 disabled:cursor-not-allowed",
  "flex items-center justify-center",
);

const STOP_BUTTON_CLASSNAME = cn(
  "p-1 rounded-full bg-slate-600 text-white hover:bg-slate-700",
  "cursor-pointer",
  "flex items-center justify-center",
);

export function InputForm({
  aiInputRef,
  aiInput,
  onAiInputChange,
  onSendPrompt,
  onStopPrompt,
  disabled,
  canvasId,
  isGeneratingResponse,
  maxAiInputHeight,
  expanded = false,
}: InputFormProps) {
  const isSendDisabled = disabled || isGeneratingResponse || !canvasId || !aiInput.trim();

  const keyDownHandler = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      onSendPrompt();
    }
  };

  const submitHandler = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    onSendPrompt();
  };

  return (
    <div className={cn("m-1.5", expanded && "mb-3")}>
      <form
        onSubmit={submitHandler}
        className={cn("rounded-md border border-slate-300 bg-white p-1.5", expanded && "p-3 shadow-sm")}
      >
        <Textarea
          ref={aiInputRef}
          value={aiInput}
          onChange={(e) => onAiInputChange(e.target.value)}
          onKeyDown={keyDownHandler}
          placeholder="What would you like to build?"
          disabled={disabled || !canvasId}
          rows={expanded ? 4 : 1}
          className={cn(TEXT_AREA_CLASSNAME, expanded && "min-h-[112px] text-[15px] leading-6")}
          style={{ maxHeight: `${maxAiInputHeight}px` }}
        />

        <div className="flex items-center justify-end">
          {isGeneratingResponse ? (
            <button
              type="button"
              className={STOP_BUTTON_CLASSNAME}
              onClick={onStopPrompt}
              aria-label="Stop prompt"
              title="Stop prompt"
            >
              <Square size={14} />
            </button>
          ) : (
            <button
              type="submit"
              className={SUBMIT_BUTTON_CLASSNAME}
              disabled={isSendDisabled}
              aria-label="Send prompt"
            >
              <ArrowUp size={14} />
            </button>
          )}
        </div>
      </form>
    </div>
  );
}
