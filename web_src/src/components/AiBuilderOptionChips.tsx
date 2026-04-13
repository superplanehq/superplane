import { Button } from "@/components/ui/button";
import type { AiFollowUpOption } from "@/ui/BuildingBlocksSidebar/agentChat";
import { MessageSquare } from "lucide-react";

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
    <div className="flex flex-wrap gap-1.5 pt-2">
      {options.map((option) => (
        <Button
          key={option.value}
          variant="outline"
          size="sm"
          disabled={disabled}
          onClick={() => onSelect(option.value)}
          className="h-auto whitespace-normal text-left py-1 px-2.5 text-xs font-normal"
        >
          {option.label}
        </Button>
      ))}
      <Button
        variant="ghost"
        size="sm"
        disabled={disabled}
        onClick={onFocusInput}
        className="h-auto py-1 px-2.5 text-xs font-normal text-muted-foreground"
      >
        <MessageSquare className="h-3 w-3 mr-1" />
        Other...
      </Button>
    </div>
  );
}
