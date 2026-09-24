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
  options,
  onValueChange,
  testId,
}: {
  label: string;
  value: string;
  options: Array<{ value: string; label: string; testId?: string }>;
  onValueChange: (next: string) => void;
  testId?: string;
}) {
  const currentLabel = options.find((option) => option.value === value)?.label ?? value;

  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger className={SUB_TRIGGER_CLASSNAME} data-testid={testId}>
        {label}
        <span className="ml-auto text-[11px] text-muted-foreground">{currentLabel}</span>
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent>
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
              <span className="flex-1">{option.label}</span>
              {value === option.value ? <Check className="size-3.5" aria-hidden /> : null}
            </DropdownMenuItem>
          ))}
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}
