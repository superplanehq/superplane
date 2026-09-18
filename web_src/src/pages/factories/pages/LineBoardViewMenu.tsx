import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Check, Settings } from "lucide-react";

import type { ColumnAutomationView } from "../lib/columnAutomationViewPreference";
import { COLUMN_AUTOMATIONS_COPY } from "../lib/columnAutomations";
import type { LineBoardColumnColorView } from "../lib/lineBoardColumnColorViewPreference";
import { MENU_ITEM_CLASSNAME, MENU_LABEL_CLASSNAME } from "../workOrders/header/menuStyles";

const AUTOMATION_VIEW_OPTIONS: Array<{ id: ColumnAutomationView; label: string; testId: string }> = [
  { id: "names", label: COLUMN_AUTOMATIONS_COPY.viewNames, testId: "lines-board-view-names" },
  { id: "icons", label: COLUMN_AUTOMATIONS_COPY.viewIcons, testId: "lines-board-view-icons" },
];

const COLOR_VIEW_OPTIONS: Array<{ id: LineBoardColumnColorView; label: string; testId: string }> = [
  { id: "fill", label: COLUMN_AUTOMATIONS_COPY.viewColumnColors, testId: "lines-board-view-column-colors" },
  { id: "off", label: COLUMN_AUTOMATIONS_COPY.viewNoColumnColors, testId: "lines-board-view-no-column-colors" },
  { id: "borders", label: COLUMN_AUTOMATIONS_COPY.viewColoredBorders, testId: "lines-board-view-colored-borders" },
];

/**
 * Header cog. View options choose automation layout and how column colors
 * appear on the board.
 */
export function LineBoardViewMenu({
  view,
  onViewChange,
  colorView,
  onColorViewChange,
}: {
  view?: ColumnAutomationView;
  onViewChange?: (view: ColumnAutomationView) => void;
  colorView: LineBoardColumnColorView;
  onColorViewChange: (view: LineBoardColumnColorView) => void;
}) {
  const showAutomationView = Boolean(view && onViewChange);

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
        {showAutomationView
          ? AUTOMATION_VIEW_OPTIONS.map((option) => (
              <DropdownMenuItem
                key={option.id}
                className={MENU_ITEM_CLASSNAME}
                onSelect={() => onViewChange?.(option.id)}
                data-testid={option.testId}
              >
                <span className="flex-1">{option.label}</span>
                {view === option.id ? <Check className="size-3.5" aria-hidden /> : null}
              </DropdownMenuItem>
            ))
          : null}
        {showAutomationView ? <DropdownMenuSeparator /> : null}
        {COLOR_VIEW_OPTIONS.map((option) => (
          <DropdownMenuItem
            key={option.id}
            className={MENU_ITEM_CLASSNAME}
            onSelect={() => onColorViewChange(option.id)}
            data-testid={option.testId}
          >
            <span className="flex-1">{option.label}</span>
            {colorView === option.id ? <Check className="size-3.5" aria-hidden /> : null}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
