import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { CornerDownRight, Paperclip, SendHorizontal, X } from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";

interface SteeringComposerProps {
  context: string | null;
  disabled?: boolean;
  onClearContext: () => void;
  onSend: (message: string) => void;
}

export function SteeringComposer({ context, disabled = false, onClearContext, onSend }: SteeringComposerProps) {
  const [message, setMessage] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (context) {
      textareaRef.current?.focus();
    }
  }, [context]);

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const direction = message.trim();
    if (!direction || disabled) return;

    onSend(direction);
    setMessage("");
  };

  return (
    <form
      onSubmit={submit}
      className="sticky bottom-0 z-10 border-t border-slate-200 bg-white/95 px-4 py-3 backdrop-blur-sm sm:px-6 dark:border-gray-700 dark:bg-gray-900/95"
    >
      {context ? (
        <div className="mb-2 flex items-center justify-between gap-3 text-xs text-sky-800 dark:text-sky-300">
          <span className="flex min-w-0 items-center gap-1.5">
            <CornerDownRight className="size-3.5 shrink-0" />
            <span className="truncate">In response to {context}</span>
          </span>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            aria-label="Clear steering context"
            onClick={onClearContext}
          >
            <X />
          </Button>
        </div>
      ) : null}

      <div className="rounded-lg border border-slate-300 bg-white shadow-sm focus-within:border-sky-500 focus-within:ring-2 focus-within:ring-sky-100 dark:border-gray-600 dark:bg-gray-950 dark:focus-within:border-sky-400 dark:focus-within:ring-sky-950">
        <Textarea
          ref={textareaRef}
          aria-label="Direct this work"
          value={message}
          disabled={disabled}
          onChange={(event) => setMessage(event.target.value)}
          placeholder={
            disabled ? "This work item has stopped" : "Ask a question, change direction, or add a constraint..."
          }
          className="min-h-20 resize-none border-0 bg-transparent px-3 py-2.5 text-sm shadow-none focus-visible:ring-0 dark:bg-transparent"
        />
        <div className="flex min-h-10 items-center justify-between gap-3 border-t border-slate-100 px-2 py-1.5 dark:border-gray-800">
          <div className="flex min-w-0 items-center gap-1">
            <Tooltip>
              <TooltipTrigger asChild>
                <Button type="button" variant="ghost" size="icon-xs" aria-label="Attach context" disabled={disabled}>
                  <Paperclip />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Attach context</TooltipContent>
            </Tooltip>
            <span className="truncate text-xs text-slate-500 dark:text-gray-400">Direction joins the work record</span>
          </div>
          <Button type="submit" size="sm" disabled={!message.trim() || disabled} aria-label="Send direction">
            <SendHorizontal />
            Send
          </Button>
        </div>
      </div>
    </form>
  );
}
