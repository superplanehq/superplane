import type { FactoriesFactoryLine } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Check, Search, X } from "lucide-react";
import { useState, type KeyboardEvent, type RefObject } from "react";

import type { WorkOrderListState } from "../lib/useWorkOrderListState";
import type { WorkOrderFilterOption } from "../lib/workOrderFilterOptions";
import { humanizeLineName } from "../lib/humanizeLineName";
import { FilterChips } from "../workOrders/header/FilterChips";
import { FilterMenu } from "../workOrders/header/FilterMenu";
import { MENU_ITEM_CLASSNAME, MENU_LABEL_CLASSNAME } from "../workOrders/header/menuStyles";
import { WorkOrderClosedStatusDialog } from "../workOrders/WorkOrderClosedStatusDialog";
import { MOBILE_BOARD_COPY } from "./mobileCopy";
import { MobileWorkspaceSwitcher } from "./MobileWorkspaceSwitcher";

export function MobileBoardHeader({
  state,
  searchRef,
  sourceOptions,
  assigneeOptions,
  showPullRequestMerge,
  lines,
  lineId,
  onSelectLine,
  organizationId,
  factoryId,
  factoryKey,
  canManageClosedStatus,
}: {
  state: WorkOrderListState;
  searchRef: RefObject<HTMLInputElement | null>;
  sourceOptions: WorkOrderFilterOption[];
  assigneeOptions: WorkOrderFilterOption[];
  showPullRequestMerge: boolean;
  lines: FactoriesFactoryLine[];
  lineId: string;
  onSelectLine: (lineId: string) => void;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  canManageClosedStatus: boolean;
}) {
  const [closedStatusDialogOpen, setClosedStatusDialogOpen] = useState(false);
  return (
    <header className="shrink-0 border-b border-border bg-background px-4 pt-[env(safe-area-inset-top)]">
      <div className="flex h-16 items-center justify-between gap-3" data-testid="mobile-board-header">
        <div className="min-w-0 flex-1">
          <MobileWorkspaceSwitcher />
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          onClick={state.searchOpen ? state.closeSearch : state.openSearch}
          aria-label={MOBILE_BOARD_COPY.searchPlaceholder}
          aria-pressed={state.searchOpen}
          className={cn("size-11 shrink-0 text-muted-foreground", state.searchOpen && "bg-accent text-foreground")}
          data-testid="mobile-board-search-toggle"
        >
          <Search className="size-5" aria-hidden />
        </Button>
      </div>
      {state.searchOpen ? <MobileSearchRow state={state} searchRef={searchRef} /> : null}
      <div className="flex min-h-16 items-center gap-2 pb-2">
        <h1 className="min-w-0 flex-1 text-2xl font-semibold tracking-tight">Board</h1>
        <FilterMenu
          state={state}
          sourceOptions={sourceOptions}
          assigneeOptions={assigneeOptions}
          showPullRequestMerge={showPullRequestMerge}
          onOpenStatusDialog={() => setClosedStatusDialogOpen(true)}
          triggerLabel="Filter"
        />
      </div>
      {lines.length > 1 ? (
        <div className="pb-2">
          <MobileLineSwitcher lines={lines} lineId={lineId} onSelect={onSelectLine} />
        </div>
      ) : null}
      {state.filterCount - state.filters.lineIds.length > 0 ? (
        <div className="pb-3">
          <FilterChips
            state={state}
            sourceOptions={sourceOptions}
            assigneeOptions={assigneeOptions}
            showPullRequestMerge={showPullRequestMerge}
          />
        </div>
      ) : null}
      {closedStatusDialogOpen ? (
        <WorkOrderClosedStatusDialog
          open
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          lineId={lineId}
          canManage={canManageClosedStatus}
          onOpenChange={setClosedStatusDialogOpen}
        />
      ) : null}
    </header>
  );
}

/** Chooses a line when the workspace has more than one. */
function MobileLineSwitcher({
  lines,
  lineId,
  onSelect,
}: {
  lines: FactoriesFactoryLine[];
  lineId: string;
  onSelect: (lineId: string) => void;
}) {
  const choices = lines.filter((line): line is FactoriesFactoryLine & { id: string } => Boolean(line.id));
  if (choices.length < 2) {
    return null;
  }
  const current = choices.find((line) => line.id === lineId) ?? choices[0];
  const name = humanizeLineName(current.name);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="h-11 max-w-[55%] shrink px-2 text-sm font-medium"
          aria-label={`${MOBILE_BOARD_COPY.switchLine}: ${name}`}
          data-testid="mobile-board-line-switcher"
        >
          <span className="truncate">{name}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-56">
        <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>{MOBILE_BOARD_COPY.lines}</DropdownMenuLabel>
        {choices.map((line) => (
          <DropdownMenuItem
            key={line.id}
            className={MENU_ITEM_CLASSNAME}
            data-testid={`mobile-board-line-${line.id}`}
            onSelect={() => onSelect(line.id)}
          >
            <span className="min-w-0 flex-1 truncate">{humanizeLineName(line.name)}</span>
            {line.id === lineId ? <Check className="size-3.5 shrink-0" aria-hidden /> : null}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function MobileSearchRow({
  state,
  searchRef,
}: {
  state: WorkOrderListState;
  searchRef: RefObject<HTMLInputElement | null>;
}) {
  const handleKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Escape") {
      event.preventDefault();
      state.closeSearch();
    }
  };
  return (
    <div className="relative pb-2">
      <Search
        className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-[calc(50%+0.25rem)] text-muted-foreground"
        aria-hidden
      />
      <Input
        ref={searchRef}
        autoFocus
        value={state.search}
        onChange={(event) => state.setSearch(event.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={MOBILE_BOARD_COPY.searchPlaceholder}
        aria-label={MOBILE_BOARD_COPY.searchPlaceholder}
        className="!h-9 w-full pr-9 pl-8 text-[14px] shadow-none"
        data-testid="mobile-board-search-input"
      />
      <button
        type="button"
        onClick={state.closeSearch}
        aria-label={MOBILE_BOARD_COPY.closeSearch}
        className="absolute top-1/2 right-1.5 -translate-y-[calc(50%+0.25rem)] rounded p-1 text-muted-foreground hover:text-foreground"
        data-testid="mobile-board-search-close"
      >
        <X className="size-4" aria-hidden />
      </button>
    </div>
  );
}
