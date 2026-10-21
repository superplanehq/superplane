import { ChevronDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuTrigger,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
} from "@/ui/dropdownMenu";

export interface WorkOrderRefinementModelSelection {
  defaultModel: string;
  options: Array<{ value: string; label: string }>;
  isLoading?: boolean;
}

export function WorkOrderRefinementModelSelect({
  selection,
  model,
  disabled,
  onChange,
}: {
  selection: WorkOrderRefinementModelSelection;
  model: string;
  disabled: boolean;
  onChange: (model: string) => void;
}) {
  const label = selection.options.find((option) => option.value === model)?.label || model || "Default";
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-8 min-w-0 shrink gap-1 px-2 text-xs font-normal text-muted-foreground"
          aria-label={`Model: ${label}`}
          disabled={disabled || selection.isLoading || selection.options.length === 0}
        >
          <span className="truncate">{label}</span>
          <ChevronDown className="size-3 shrink-0 opacity-60" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" side="top">
        <DropdownMenuRadioGroup value={model} onValueChange={onChange}>
          {selection.options.map((option) => (
            <DropdownMenuRadioItem key={option.value} value={option.value}>
              {option.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
