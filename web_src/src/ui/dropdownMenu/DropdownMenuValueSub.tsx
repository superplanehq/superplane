import { Check } from "lucide-react";

import {
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from "./index";

const MENU_ITEM_CLASSNAME = "cursor-pointer text-[13px]";
const SUB_TRIGGER_CLASSNAME = `${MENU_ITEM_CLASSNAME} py-1 [&_svg]:size-3.5 [&>svg:last-child]:ml-0`;

export function DropdownMenuValueSub({
  label,
  value,
  displayedValue,
  options,
  onValueChange,
  testId,
  disabled,
}: {
  label: string;
  value: string;
  displayedValue?: string;
  options: Array<{ value: string; label: string; testId?: string }>;
  onValueChange: (next: string) => void;
  testId?: string;
  disabled?: boolean;
}) {
  const currentLabel = displayedValue ?? options.find((option) => option.value === value)?.label ?? value;

  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger className={SUB_TRIGGER_CLASSNAME} data-testid={testId} disabled={disabled}>
        {label}
        <span className="ml-auto min-w-0 truncate text-[11px] text-muted-foreground" title={currentLabel}>
          {currentLabel}
        </span>
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent className="max-w-[calc(100vw-1rem)] max-h-[var(--radix-dropdown-menu-content-available-height)] overflow-x-hidden overflow-y-auto">
          {options.map((option) => (
            <DropdownMenuItem
              key={option.value || option.label}
              className={MENU_ITEM_CLASSNAME}
              data-testid={option.testId}
              onSelect={(event) => {
                event.preventDefault();
                onValueChange(option.value);
              }}
            >
              <span className="min-w-0 flex-1 break-words">{option.label}</span>
              {value === option.value ? <Check className="size-3.5" aria-hidden /> : null}
            </DropdownMenuItem>
          ))}
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}
