import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
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
  { id: "vivid", label: COLUMN_AUTOMATIONS_COPY.viewColorVivid, testId: "lines-board-view-vivid-column-colors" },
  { id: "dim", label: COLUMN_AUTOMATIONS_COPY.viewColorSoft, testId: "lines-board-view-dim-column-colors" },
  { id: "borders", label: COLUMN_AUTOMATIONS_COPY.viewColorBorders, testId: "lines-board-view-colored-borders" },
  { id: "off", label: COLUMN_AUTOMATIONS_COPY.viewColorOff, testId: "lines-board-view-no-column-colors" },
];

const SUB_TRIGGER_CLASSNAME = `${MENU_ITEM_CLASSNAME} py-1 [&_svg]:size-3.5 [&>svg:last-child]:ml-0`;

/**
 * Header cog. View options open to the side, like Appearance in the
 * profile menu.
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
        {showAutomationView && view && onViewChange ? (
          <ViewOptionSub
            label={COLUMN_AUTOMATIONS_COPY.viewAutomations}
            testId="lines-board-view-automations"
            value={view}
            options={AUTOMATION_VIEW_OPTIONS}
            onValueChange={onViewChange}
          />
        ) : null}
        <ViewOptionSub
          label={COLUMN_AUTOMATIONS_COPY.viewColumnColors}
          testId="lines-board-view-column-color"
          value={colorView}
          options={COLOR_VIEW_OPTIONS}
          onValueChange={onColorViewChange}
        />
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function ViewOptionSub<T extends string>({
  label,
  testId,
  value,
  options,
  onValueChange,
}: {
  label: string;
  testId: string;
  value: T;
  options: Array<{ id: T; label: string; testId: string }>;
  onValueChange: (next: T) => void;
}) {
  const currentLabel = options.find((option) => option.id === value)?.label ?? options[0]?.label;

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
              key={option.id}
              className={MENU_ITEM_CLASSNAME}
              onSelect={(event) => {
                event.preventDefault();
                onValueChange(option.id);
              }}
              data-testid={option.testId}
            >
              <span className="flex-1">{option.label}</span>
              {value === option.id ? <Check className="size-3.5" aria-hidden /> : null}
            </DropdownMenuItem>
          ))}
        </DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}
