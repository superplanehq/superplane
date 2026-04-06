import { Textarea } from "@/components/ui/textarea";
import { useAiBuilderMentionTypeahead } from "@/hooks/useAiBuilderMentionTypeahead";
import { AiBuilderMentionListPortal } from "@/ui/BuildingBlocksSidebar/AiBuilderMentionListPortal";
import type { AiBuilderMentionNode } from "@/lib/aiBuilderNodeMentions";
import { cn } from "@/lib/utils";
import { ArrowUp } from "lucide-react";
import type { ChangeEvent, FormEvent, RefObject } from "react";

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

export type AiBuilderChatInputProps = {
  aiInputRef: RefObject<HTMLTextAreaElement | null>;
  aiInput: string;
  onAiInputChange: (value: string) => void;
  onSendPrompt: () => void;
  disabled: boolean;
  canvasId?: string;
  canvasNodes?: AiBuilderMentionNode[];
  isGeneratingResponse: boolean;
  maxAiInputHeight: number;
  expanded?: boolean;
};

export function AiBuilderChatInput({
  aiInputRef,
  aiInput,
  onAiInputChange,
  onSendPrompt,
  disabled,
  canvasId,
  canvasNodes,
  isGeneratingResponse,
  maxAiInputHeight,
  expanded = false,
}: AiBuilderChatInputProps) {
  const mention = useAiBuilderMentionTypeahead({
    aiInput,
    aiInputRef,
    canvasNodes,
    onAiInputChange,
    onSendPrompt,
  });

  const isDisabled = disabled || isGeneratingResponse || !canvasId || !aiInput.trim();

  const changeHandler = (e: ChangeEvent<HTMLTextAreaElement>) => {
    const v = e.target.value;
    const cursor = e.target.selectionStart ?? v.length;
    mention.handleTextareaChange(v, cursor);
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
        <div ref={mention.mentionAnchorRef} className="relative">
          <AiBuilderMentionListPortal
            open={mention.mentionOpen}
            placement={mention.mentionMenuPlacement}
            nodes={mention.filteredMentionNodes}
            selectedIndex={mention.mentionSelectedIndex}
            onHoverIndex={mention.setMentionSelectedIndex}
            onPick={mention.applyMention}
          />

          <Textarea
            ref={aiInputRef}
            value={aiInput}
            onChange={changeHandler}
            onKeyDown={mention.handleTextareaKeyDown}
            onClick={(e) => mention.syncMentionUi(aiInput, e.currentTarget.selectionStart ?? aiInput.length)}
            onSelect={(e) => mention.syncMentionUi(aiInput, e.currentTarget.selectionStart ?? aiInput.length)}
            placeholder="What would you like to build? (@ to mention a step)"
            disabled={disabled || !canvasId}
            rows={expanded ? 4 : 1}
            className={cn(TEXT_AREA_CLASSNAME, expanded && "min-h-[112px] text-[15px] leading-6")}
            style={{ maxHeight: `${maxAiInputHeight}px` }}
          />
        </div>

        <div className="flex items-center justify-end">
          <button type="submit" className={SUBMIT_BUTTON_CLASSNAME} disabled={isDisabled} aria-label="Send prompt">
            <ArrowUp size={14} />
          </button>
        </div>
      </form>
    </div>
  );
}
