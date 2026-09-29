import { Check, ListFilter } from "lucide-react";
import type { ReactNode } from "react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuPortal,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

import {
  defaultLineColumnView,
  isDefaultLineColumnView,
  LINE_COLUMN_AGE_WINDOWS,
  LINE_COLUMN_SORT_KEYS,
  toggleLineColumnSource,
  withLineColumnAge,
  withLineColumnDirection,
  withLineColumnMinimum,
  withLineColumnNoScore,
  withLineColumnSort,
  withLineColumnSources,
  type LineColumnAgeWindow,
  type LineColumnSortDirection,
  type LineColumnSortKey,
  type LineColumnViewChoice,
} from "../lib/lineColumnView";
import type { WorkOrderFilterOption } from "../lib/workOrderFilterOptions";
import { MENU_ITEM_CLASSNAME, MENU_LABEL_CLASSNAME } from "../workOrders/header/menuStyles";

const SORT_LABEL: Record<LineColumnSortKey, string> = {
  updated: "Newest update",
  confidence: "Confidence",
  source: "Source",
  age: "Issue age",
};

const DIRECTION_LABEL: Record<LineColumnSortKey, Record<LineColumnSortDirection, string>> = {
  updated: { forward: "Newest first", reverse: "Oldest first" },
  confidence: { forward: "Highest first", reverse: "Lowest first" },
  source: { forward: "A to Z", reverse: "Z to A" },
  age: { forward: "Oldest first", reverse: "Newest first" },
};

const AGE_LABEL: Record<LineColumnAgeWindow, string> = {
  any: "Any age",
  "7": "Last 7 days",
  "30": "Last 30 days",
  "90": "Last 90 days",
  older: "Older than 90 days",
};

const CONFIDENCE_MINIMUMS = [1, 2, 3, 4, 5] as const;

export function ColumnViewMenu({
  title,
  testId,
  choice,
  sourceOptions,
  onChange,
}: {
  title: string;
  testId: string;
  choice: LineColumnViewChoice;
  sourceOptions: readonly WorkOrderFilterOption[];
  onChange: (choice: LineColumnViewChoice) => void;
}) {
  const active = !isDefaultLineColumnView(choice);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`Sort and filter ${title}`}
          data-testid={testId}
          data-active={active ? "" : undefined}
          className="relative flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <ListFilter className="size-3.5" aria-hidden />
          {active ? (
            <span className="absolute top-0.5 right-0.5 size-1.5 rounded-full bg-foreground" aria-hidden />
          ) : null}
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56" data-testid={`${testId}-content`}>
        <ColumnViewSortSection testId={testId} choice={choice} onChange={onChange} />
        <DropdownMenuSeparator />
        <ColumnViewFilterSection testId={testId} choice={choice} sourceOptions={sourceOptions} onChange={onChange} />
        {active ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className={MENU_ITEM_CLASSNAME}
              data-testid={`${testId}-clear`}
              onSelect={() => onChange(defaultLineColumnView())}
            >
              Clear
            </DropdownMenuItem>
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function ColumnViewSortSection({
  testId,
  choice,
  onChange,
}: {
  testId: string;
  choice: LineColumnViewChoice;
  onChange: (choice: LineColumnViewChoice) => void;
}) {
  const directionLabels = DIRECTION_LABEL[choice.sortKey];

  return (
    <>
      <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>Sort</DropdownMenuLabel>
      {LINE_COLUMN_SORT_KEYS.map((sortKey) => (
        <DropdownMenuItem
          key={sortKey}
          className={MENU_ITEM_CLASSNAME}
          data-testid={`${testId}-sort-${sortKey}`}
          onSelect={() => onChange(withLineColumnSort(choice, sortKey))}
        >
          <span className="flex-1">{SORT_LABEL[sortKey]}</span>
          {choice.sortKey === sortKey ? <Check className="size-3.5" aria-hidden /> : null}
        </DropdownMenuItem>
      ))}
      <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>Direction</DropdownMenuLabel>
      {(["forward", "reverse"] as const).map((direction) => (
        <DropdownMenuItem
          key={direction}
          className={MENU_ITEM_CLASSNAME}
          data-testid={`${testId}-direction-${direction}`}
          onSelect={() => onChange(withLineColumnDirection(choice, direction))}
        >
          <span className="flex-1">{directionLabels[direction]}</span>
          {choice.direction === direction ? <Check className="size-3.5" aria-hidden /> : null}
        </DropdownMenuItem>
      ))}
    </>
  );
}

function ColumnViewFilterSection({
  testId,
  choice,
  sourceOptions,
  onChange,
}: {
  testId: string;
  choice: LineColumnViewChoice;
  sourceOptions: readonly WorkOrderFilterOption[];
  onChange: (choice: LineColumnViewChoice) => void;
}) {
  return (
    <>
      <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>Filter</DropdownMenuLabel>
      <FilterSubmenu label="Source" testId={`${testId}-source`}>
        <FilterChoice
          testId={`${testId}-source-any`}
          label="Any source"
          selected={choice.sourceIds.length === 0}
          onSelect={() => onChange(withLineColumnSources(choice, []))}
        />
        {sourceOptions.map((option) => (
          <FilterChoice
            key={option.value}
            testId={`${testId}-source-${option.value}`}
            label={option.label}
            selected={choice.sourceIds.includes(option.value)}
            onSelect={() => onChange(toggleLineColumnSource(choice, option.value))}
          />
        ))}
      </FilterSubmenu>
      <FilterSubmenu label="Confidence" testId={`${testId}-confidence`}>
        <FilterChoice
          testId={`${testId}-confidence-any`}
          label="Any score"
          selected={!choice.noScore && choice.minimumConfidence == null}
          onSelect={() => onChange(withLineColumnNoScore(withLineColumnMinimum(choice, undefined), false))}
        />
        {CONFIDENCE_MINIMUMS.map((minimum) => (
          <FilterChoice
            key={minimum}
            testId={`${testId}-confidence-${minimum}`}
            label={`${minimum} or higher`}
            selected={!choice.noScore && choice.minimumConfidence === minimum}
            onSelect={() => onChange(withLineColumnMinimum(choice, minimum))}
          />
        ))}
        <FilterChoice
          testId={`${testId}-confidence-none`}
          label="No score"
          selected={choice.noScore}
          onSelect={() => onChange(withLineColumnNoScore(choice, true))}
        />
      </FilterSubmenu>
      <FilterSubmenu label="Age" testId={`${testId}-age`}>
        {LINE_COLUMN_AGE_WINDOWS.map((window) => (
          <FilterChoice
            key={window}
            testId={`${testId}-age-${window}`}
            label={AGE_LABEL[window]}
            selected={choice.ageWindow === window}
            onSelect={() => onChange(withLineColumnAge(choice, window))}
          />
        ))}
      </FilterSubmenu>
    </>
  );
}

function FilterSubmenu({ label, testId, children }: { label: string; testId: string; children: ReactNode }) {
  return (
    <DropdownMenuSub>
      <DropdownMenuSubTrigger className={MENU_ITEM_CLASSNAME} data-testid={testId}>
        <span className="flex-1">{label}</span>
      </DropdownMenuSubTrigger>
      <DropdownMenuPortal>
        <DropdownMenuSubContent className="w-52">{children}</DropdownMenuSubContent>
      </DropdownMenuPortal>
    </DropdownMenuSub>
  );
}

function FilterChoice({
  testId,
  label,
  selected,
  onSelect,
}: {
  testId: string;
  label: string;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <DropdownMenuItem
      className={MENU_ITEM_CLASSNAME}
      data-testid={testId}
      onSelect={(event) => {
        event.preventDefault();
        onSelect();
      }}
    >
      <span className="flex-1 truncate">{label}</span>
      {selected ? <Check className="size-3.5" aria-hidden /> : null}
    </DropdownMenuItem>
  );
}
