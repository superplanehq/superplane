import { Button } from "@/components/ui/button";
import type { AiFollowUpOption } from "@/ui/BuildingBlocksSidebar/agentChat";
import { ChevronRight, MessageSquare } from "lucide-react";

export type AiBuilderOptionChipsProps = {
  options: AiFollowUpOption[];
  onSelect: (value: string) => void;
  onFocusInput: () => void;
  disabled: boolean;
};

export function AiBuilderOptionChips({ options, onSelect, onFocusInput, disabled }: AiBuilderOptionChipsProps) {
  if (options.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-1 pt-2">
      {options.map((option) => (
        <Button
          key={option.value}
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={() => onSelect(option.value)}
          className="h-auto w-full justify-start whitespace-normal text-left py-2 px-3 text-xs font-normal gap-2"
        >
          <ChevronRight className="h-3 w-3 shrink-0 text-muted-foreground" />
          {option.label}
        </Button>
      ))}
      <Button
        variant="ghost"
        size="sm"
        disabled={disabled}
        onClick={onFocusInput}
        className="h-auto w-full justify-start py-2 px-3 text-xs font-normal text-muted-foreground gap-2"
      >
        <MessageSquare className="h-3 w-3 shrink-0" />
        Other...
      </Button>
    </div>
  );
}
