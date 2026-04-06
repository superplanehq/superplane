import { aiBuilderNodeDisplayName, type AiBuilderMentionNode } from "@/lib/aiBuilderNodeMentions";
import type { MentionMenuPlacement } from "@/lib/aiBuilderMentionTypeahead";
import { cn } from "@/lib/utils";
import { createPortal } from "react-dom";

export type AiBuilderMentionListPortalProps = {
  open: boolean;
  placement: MentionMenuPlacement | null;
  nodes: AiBuilderMentionNode[];
  selectedIndex: number;
  onHoverIndex: (index: number) => void;
  onPick: (node: AiBuilderMentionNode) => void;
};

export function AiBuilderMentionListPortal({
  open,
  placement,
  nodes,
  selectedIndex,
  onHoverIndex,
  onPick,
}: AiBuilderMentionListPortalProps) {
  if (!open || !placement || nodes.length === 0) {
    return null;
  }

  return createPortal(
    <div
      className="fixed z-[300] overflow-y-auto rounded-md border border-slate-200 bg-popover py-1 shadow-lg"
      role="listbox"
      style={{
        left: placement.left,
        width: placement.width,
        bottom: placement.bottom,
        maxHeight: placement.maxHeight,
      }}
    >
      {nodes.map((node, index) => {
        const label = aiBuilderNodeDisplayName(node);
        return (
          <button
            key={node.id}
            type="button"
            role="option"
            aria-selected={index === selectedIndex}
            className={cn(
              "block w-full px-2 py-1.5 text-left text-sm text-slate-800",
              index === selectedIndex ? "bg-slate-100" : "hover:bg-slate-50",
            )}
            onMouseDown={(ev) => ev.preventDefault()}
            onMouseEnter={() => onHoverIndex(index)}
            onClick={() => onPick(node)}
          >
            {label}
          </button>
        );
      })}
    </div>,
    document.body,
  );
}
