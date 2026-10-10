import type { FactoriesFactoryLine, FactoriesFactoryPrFeedbackHandler } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { logoDarkInvertClass } from "@/lib/logoDarkMode";
import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { Check, MoreHorizontal, Radio, Search, X } from "lucide-react";
import { useState, type KeyboardEvent, type ReactNode, type RefObject } from "react";

import type { WorkOrderListState } from "../lib/useWorkOrderListState";
import type { WorkOrderFilterOption } from "../lib/workOrderFilterOptions";
import type { LineBoardColumnColorView } from "../lib/lineBoardColumnColorViewPreference";
import { LineBoardViewMenu } from "../pages/LineBoardViewMenu";
import type { ConfiguredLineIntakeSource } from "../pages/lineIntakeModel";
import { lineIntakeListenTitle } from "../pages/lineIntakeModel";
import { prFeedbackListenTitle, prFeedbackVCSIcon } from "../pages/prFeedbackSettingsModel";
import { humanizeLineName } from "../lib/humanizeLineName";
import { FilterChips } from "../workOrders/header/FilterChips";
import { FilterMenu } from "../workOrders/header/FilterMenu";
import { MENU_ITEM_CLASSNAME, MENU_LABEL_CLASSNAME } from "../workOrders/header/menuStyles";
import { WorkOrderClosedStatusDialog } from "../workOrders/WorkOrderClosedStatusDialog";
import { MOBILE_BOARD_COPY } from "./mobileCopy";

/**
 * Phone title bar for the line board. The credits chip stays on the left;
 * filter, search, view settings, and the board menu sit on the right. Search
 * opens as a full-width row so the input has room on a narrow screen.
 */
export function MobileBoardHeader({
  state,
  searchRef,
  creditKicker,
  sourceOptions,
  assigneeOptions,
  showPullRequestMerge,
  colorView,
  onColorViewChange,
  intakes,
  prFeedbackHandlers,
  onOpenIntake,
  onOpenPRFeedback,
  lines,
  lineId,
  onSelectLine,
  organizationId,
  factoryId,
  factoryKey,
  canManageClosedStatus,
  vcsProvider,
}: {
  state: WorkOrderListState;
  searchRef: RefObject<HTMLInputElement | null>;
  creditKicker?: ReactNode;
  sourceOptions: WorkOrderFilterOption[];
  assigneeOptions: WorkOrderFilterOption[];
  showPullRequestMerge: boolean;
  colorView: LineBoardColumnColorView;
  onColorViewChange: (view: LineBoardColumnColorView) => void;
  intakes: ConfiguredLineIntakeSource[];
  prFeedbackHandlers: FactoriesFactoryPrFeedbackHandler[];
  onOpenIntake: (intake: ConfiguredLineIntakeSource) => void;
  onOpenPRFeedback: (handlerId: string) => void;
  lines: FactoriesFactoryLine[];
  lineId: string;
  onSelectLine: (lineId: string) => void;
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  canManageClosedStatus: boolean;
  vcsProvider?: string;
}) {
  const [closedStatusDialogOpen, setClosedStatusDialogOpen] = useState(false);

  return (
    <header className="shrink-0 border-b border-border bg-background px-3 pt-[env(safe-area-inset-top)]">
      <div className="flex h-12 items-center gap-1" data-testid="mobile-board-header">
        <MobileLineSwitcher lines={lines} lineId={lineId} onSelect={onSelectLine} />
        <div className="flex min-w-0 flex-1 items-center overflow-hidden">{creditKicker}</div>
        <div className="flex shrink-0 items-center gap-0.5">
          <FilterMenu
            state={state}
            sourceOptions={sourceOptions}
            assigneeOptions={assigneeOptions}
            showPullRequestMerge={showPullRequestMerge}
            onOpenStatusDialog={() => setClosedStatusDialogOpen(true)}
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            onClick={state.searchOpen ? state.closeSearch : state.openSearch}
            aria-label={MOBILE_BOARD_COPY.searchPlaceholder}
            aria-pressed={state.searchOpen}
            className={cn("size-8 shrink-0 text-muted-foreground", state.searchOpen && "bg-accent text-foreground")}
            data-testid="mobile-board-search-toggle"
          >
            <Search className="size-3.5" aria-hidden />
          </Button>
          <LineBoardViewMenu colorView={colorView} onColorViewChange={onColorViewChange} />
          <MobileBoardMenu
            intakes={intakes}
            prFeedbackHandlers={prFeedbackHandlers}
            onOpenIntake={onOpenIntake}
            onOpenPRFeedback={onOpenPRFeedback}
            vcsProvider={vcsProvider}
          />
        </div>
      </div>
      {state.searchOpen ? <MobileSearchRow state={state} searchRef={searchRef} /> : null}
      {state.filterCount - state.filters.lineIds.length > 0 ? (
        <div className="pb-2">
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
          onOpenChange={(open) => {
            if (!open) {
              setClosedStatusDialogOpen(false);
            }
          }}
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
          className="h-8 max-w-[42%] shrink px-2 text-[13px] font-medium"
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

/** Intakes and pull request listeners live here instead of on the columns. */
function MobileBoardMenu({
  intakes,
  prFeedbackHandlers,
  onOpenIntake,
  onOpenPRFeedback,
  vcsProvider,
}: {
  intakes: ConfiguredLineIntakeSource[];
  prFeedbackHandlers: FactoriesFactoryPrFeedbackHandler[];
  onOpenIntake: (intake: ConfiguredLineIntakeSource) => void;
  onOpenPRFeedback: (handlerId: string) => void;
  vcsProvider?: string;
}) {
  const handlers = prFeedbackHandlers.filter((handler): handler is typeof handler & { id: string } =>
    Boolean(handler.id),
  );
  const verifyIcon = prFeedbackVCSIcon(vcsProvider);
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          aria-label={MOBILE_BOARD_COPY.moreMenu}
          className="size-8 shrink-0 text-muted-foreground"
          data-testid="mobile-board-menu"
        >
          <MoreHorizontal className="size-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>{MOBILE_BOARD_COPY.intakes}</DropdownMenuLabel>
        {intakes.length === 0 ? (
          <div className="px-2 py-1.5 text-[12px] text-muted-foreground">{MOBILE_BOARD_COPY.noIntakes}</div>
        ) : (
          intakes.map((intake) => (
            <DropdownMenuItem
              key={intake.intakeId}
              className={MENU_ITEM_CLASSNAME}
              data-testid={`mobile-board-intake-${intake.intakeId}`}
              onSelect={() => onOpenIntake(intake)}
            >
              <ListenerRow
                title={lineIntakeListenTitle(intake.source, intake.paused)}
                iconSrc={intake.source.iconSrc}
                iconAlt={intake.source.iconAlt}
                healthy={intake.healthy}
              />
            </DropdownMenuItem>
          ))
        )}
        {handlers.length > 0 ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuLabel className={MENU_LABEL_CLASSNAME}>{MOBILE_BOARD_COPY.listeners}</DropdownMenuLabel>
            {handlers.map((handler) => (
              <DropdownMenuItem
                key={handler.id}
                className={MENU_ITEM_CLASSNAME}
                data-testid={`mobile-board-listener-${handler.id}`}
                onSelect={() => onOpenPRFeedback(handler.id)}
              >
                <ListenerRow
                  title={prFeedbackListenTitle(handler.source)}
                  iconSrc={verifyIcon.iconSrc}
                  iconAlt={verifyIcon.iconAlt}
                  healthy={handler.healthy !== false}
                />
              </DropdownMenuItem>
            ))}
          </>
        ) : null}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function ListenerRow({
  title,
  iconSrc,
  iconAlt,
  healthy,
}: {
  title: string;
  iconSrc: string;
  iconAlt: string;
  healthy: boolean;
}) {
  return (
    <span className="flex min-w-0 flex-1 items-center gap-2">
      <Radio
        className={cn("size-3.5 shrink-0", healthy ? "text-emerald-600 dark:text-emerald-400" : "text-amber-600")}
        aria-hidden
      />
      <img
        src={iconSrc}
        alt=""
        className={cn(
          "size-3.5 shrink-0 object-contain",
          iconAlt === "GitHub" && "dark:brightness-0 dark:invert",
          logoDarkInvertClass(iconSrc),
        )}
      />
      <span className="min-w-0 flex-1 truncate">{title}</span>
      {healthy ? null : (
        <span className="shrink-0 text-[11px] font-medium text-amber-700 dark:text-amber-400">
          {MOBILE_BOARD_COPY.needsRepair}
        </span>
      )}
    </span>
  );
}
