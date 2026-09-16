import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Check, Settings } from "lucide-react";

import type { ColumnAutomationView } from "../lib/columnAutomationViewPreference";
import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomations";
import { MENU_ITEM_CLASSNAME, MENU_LABEL_CLASSNAME } from "../workOrders/header/menuStyles";

const VIEW_OPTIONS: Array<{ id: ColumnAutomationView; label: string; testId: string }> = [
  { id: "names", label: COLUMN_AUTOMATIONS_COPY.viewNames, testId: "lines-board-view-names" },
  { id: "icons", label: COLUMN_AUTOMATIONS_COPY.viewIcons, testId: "lines-board-view-icons" },
];

/**
 * Header cog. View options choose sentence rows or the original title icons.
 */
export function LineBoardViewMenu({
  view,
  onViewChange,
}: {
  view: ColumnAutomationView;
  onViewChange: (view: ColumnAutomationView) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={COLUMN_AUTOMATIONS_COPY.viewMenuLabel}
          className="size-8 shrink-0 text-muted-foreground"
          data-testid="lines-board-view-menu"
        >
          <Settings className="size-3.5" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-52">
        <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>{COLUMN_AUTOMATIONS_COPY.viewOptions}</DropdownMenuLabel>
        {VIEW_OPTIONS.map((option) => (
          <DropdownMenuItem
            key={option.id}
            className={MENU_ITEM_CLASSNAME}
            onSelect={() => onViewChange(option.id)}
            data-testid={option.testId}
          >
            <span className="flex-1">{option.label}</span>
            {view === option.id ? <Check className="size-3.5" aria-hidden /> : null}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
