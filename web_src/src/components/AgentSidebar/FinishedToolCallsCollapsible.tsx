import type { AiBuilderMessage } from "./agentChat";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight } from "lucide-react";

type FinishedToolCallsCollapsibleProps = {
  tools: AiBuilderMessage[];
};

export function FinishedToolCallsCollapsible({ tools }: FinishedToolCallsCollapsibleProps) {
  const label = "Thinking process";

  return (
    <div className="w-full -mt-1">
      <Collapsible defaultOpen={false} className="w-full px-2">
        <CollapsibleTrigger asChild>
          <button
            type="button"
            className="flex w-full max-w-full items-center gap-1 py-1 px-0 text-left text-xs font-normal text-gray-500 hover:text-gray-600 focus-visible:rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/30 data-[state=open]:[&>svg]:rotate-90"
            aria-label={`${label}. Expand for more detail.`}
          >
            <span className="min-w-0 shrink">{label}</span>
            <ChevronRight
              className="h-3.5 w-3.5 mt-px shrink-0 text-gray-400 transition-transform"
              aria-hidden={true}
            />
          </button>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <ul className="list-none space-y-0.5 py-0.5 pl-0">
            {tools.map((tool) => (
              <li key={tool.id} className="text-xs leading-relaxed text-gray-500 whitespace-pre-wrap break-words">
                {tool.content}
              </li>
            ))}
          </ul>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
}
