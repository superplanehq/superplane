import { cn } from "@/lib/utils";
import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "react-router";

import { PermissionDeniedPage } from "@/components/PermissionDeniedPage";
import { PublicFactoriesSidebar } from "../layout/FactoriesSidebar";
import { WorkspacePageHeader } from "../layout/WorkspacePageHeader";
import { columnAutomationHeaderRowCount } from "../lib/columnAutomationHeadline";
import type { buildAssigneeFilterOptions, buildSourceFilterOptions } from "../lib/workOrderFilterOptions";
import { humanizeLineName } from "../lib/humanizeLineName";
import { useLineBoardColumnColorViewPreference } from "../lib/lineBoardColumnColorViewPreference";
import { useFactoriesThemeClass } from "../lib/useFactoriesThemeClass";
import { useWorkOrderListState } from "../lib/useWorkOrderListState";
import { FilterChips } from "../workOrders/header/FilterChips";
import { FilterMenu } from "../workOrders/header/FilterMenu";
import { WorkOrderKanbanBoard, workOrderKanbanLaneSizeClassName } from "../workOrders/WorkOrderBoardChrome";
import { SearchField } from "../workOrders/header/SearchField";
import {
  factoryKanbanPageClassName,
  factorySectionHeaderClassName,
  factoryWorkOrdersBodyClassName,
} from "./factoryPageLayoutStyles";
import { PublicBoardColumnLane } from "./PublicFactoryBoardLane";
import {
  boardFilterOptions,
  columnAutomations,
  loadStateForErrorStatus,
  parseBoardEvent,
  publicBoardSocketPath,
  publicBoardUrl,
  visibleBoardColumns,
  type BoardLoad,
  type PublicBoard,
  type PublicBoardColumn,
} from "./publicBoardModel";

export function PublicFactoryBoardPage({
  signedIn,
  accountName,
  accountAvatarUrl,
}: {
  signedIn: boolean;
  accountName?: string;
  accountAvatarUrl?: string | null;
}) {
  useFactoriesThemeClass();
  const {
    organizationId = "",
    factoryKey = "",
    lineId = "",
  } = useParams<{
    organizationId: string;
    factoryKey: string;
    lineId: string;
  }>();
  const [load, setLoad] = useState<BoardLoad>({ status: "loading" });
  const listState = useWorkOrderListState(`${organizationId}:${factoryKey}:${lineId}`);
  const signedInRef = useRef(signedIn);
  signedInRef.current = signedIn;
  const reloadRef = useRef<() => Promise<void>>(async () => undefined);

  const loadBoard = async () => {
    let response: Response;
    try {
      response = await fetch(publicBoardUrl(organizationId, factoryKey, lineId), { credentials: "same-origin" });
    } catch {
      return;
    }
    const failed = loadStateForErrorStatus(response.status, signedInRef.current);
    if (failed) {
      setLoad(failed);
      return;
    }
    const board = (await response.json()) as PublicBoard;
    setLoad({ status: "ready", board });
  };
  reloadRef.current = loadBoard;

  useEffect(() => {
    let cancelled = false;
    const run = async () => {
      const response = await fetch(publicBoardUrl(organizationId, factoryKey, lineId), { credentials: "same-origin" });
      if (cancelled) {
        return;
      }
      const failed = loadStateForErrorStatus(response.status, signedInRef.current);
      if (failed) {
        setLoad(failed);
        return;
      }
      const board = (await response.json()) as PublicBoard;
      if (!cancelled) {
        setLoad({ status: "ready", board });
      }
    };
    void run();
    return () => {
      cancelled = true;
    };
  }, [organizationId, factoryKey, lineId]);

  usePublicBoardSocket(load.status === "ready", organizationId, factoryKey, lineId, reloadRef);

  const board = load.status === "ready" ? load.board : null;
  const columns = useMemo(
    () => visibleBoardColumns(board?.columns ?? [], listState.filters, listState.search),
    [board, listState.filters, listState.search],
  );
  const filterOptions = useMemo(() => boardFilterOptions(board?.columns ?? [], board?.workspaceKey), [board]);
  const automationRowCount = useMemo(
    () => columnAutomationHeaderRowCount((board?.columns ?? []).map((column) => columnAutomations(column))),
    [board],
  );

  if (load.status === "loading") {
    return <PublicBoardStatus title="Loading board" />;
  }
  if (load.status === "denied") {
    return <PermissionDeniedPage description="You do not have permission to open this workspace." />;
  }
  if (load.status === "missing" || !board) {
    return (
      <PublicBoardStatus
        title="This board is not available."
        signInHref={signedIn ? undefined : guestSignInHref()}
      />
    );
  }

  return (
    <PublicBoardView
      organizationId={organizationId}
      factoryKey={factoryKey}
      lineId={lineId}
      account={signedIn ? { name: accountName, avatarUrl: accountAvatarUrl } : null}
      board={board}
      columns={columns}
      automationRowCount={automationRowCount}
      listState={listState}
      sourceOptions={filterOptions.sources}
      assigneeOptions={filterOptions.assignees}
      narrowed={listState.search.trim().length > 0 || listState.filterCount > 0}
    />
  );
}

function usePublicBoardSocket(
  boardReady: boolean,
  organizationId: string,
  factoryKey: string,
  lineId: string,
  reloadRef: { current: () => Promise<void> },
) {
  useEffect(() => {
    if (!boardReady) {
      return;
    }
    let closedByPage = false;
    let socket: WebSocket | null = null;
    let retry: number | undefined;
    let delay = 1000;

    const connect = () => {
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      socket = new WebSocket(
        `${protocol}//${window.location.host}${publicBoardSocketPath(organizationId, factoryKey, lineId)}`,
      );
      socket.onopen = () => {
        delay = 1000;
      };
      socket.onmessage = (event) => {
        const message = parseBoardEvent(event.data);
        if (message === "board_changed") {
          void reloadRef.current();
        }
      };
      socket.onclose = () => {
        if (closedByPage) {
          return;
        }
        void reloadRef.current();
        retry = window.setTimeout(connect, delay);
        delay = Math.min(delay * 2, 30000);
      };
    };
    connect();
    return () => {
      closedByPage = true;
      if (retry !== undefined) {
        window.clearTimeout(retry);
      }
      socket?.close();
    };
  }, [boardReady, organizationId, factoryKey, lineId, reloadRef]);
}

function PublicBoardView({
  organizationId,
  factoryKey,
  lineId,
  account,
  board,
  columns,
  automationRowCount,
  listState,
  sourceOptions,
  assigneeOptions,
  narrowed,
}: {
  organizationId: string;
  factoryKey: string;
  lineId: string;
  account: { name?: string; avatarUrl?: string | null } | null;
  board: PublicBoard;
  columns: PublicBoardColumn[];
  automationRowCount: number;
  listState: ReturnType<typeof useWorkOrderListState>;
  sourceOptions: ReturnType<typeof buildSourceFilterOptions>;
  assigneeOptions: ReturnType<typeof buildAssigneeFilterOptions>;
  narrowed: boolean;
}) {
  const searchRef = useRef<HTMLInputElement>(null);
  const { view: colorView } = useLineBoardColumnColorViewPreference();

  return (
    <div className="flex h-screen w-full bg-background text-foreground" data-testid="public-factory-board">
      <PublicFactoriesSidebar
        organizationId={organizationId}
        factoryKey={factoryKey}
        lineId={lineId}
        workspaceName={board.workspaceName}
        account={account}
      />
      <main className="relative flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background">
        <div className={factoryKanbanPageClassName}>
          <WorkspacePageHeader
            className={factorySectionHeaderClassName}
            data-testid="public-board-header"
            title={humanizeLineName(board.lineName)}
            leading={<PublicViewBadge />}
            actions={
              <>
                <FilterMenu state={listState} sourceOptions={sourceOptions} assigneeOptions={assigneeOptions} />
                <SearchField
                  inputRef={searchRef}
                  open={listState.searchOpen}
                  value={listState.search}
                  onOpen={listState.openSearch}
                  onChange={listState.setSearch}
                  onClose={listState.closeSearch}
                />
              </>
            }
            belowRow={<FilterChips state={listState} sourceOptions={sourceOptions} assigneeOptions={assigneeOptions} />}
          />
          <div className={factoryWorkOrdersBodyClassName}>
            <WorkOrderKanbanBoard testId="public-board-columns">
              {columns.map((column, index) => (
                <div
                  key={column.key}
                  className={cn("relative flex min-h-0 self-stretch", workOrderKanbanLaneSizeClassName)}
                >
                  {index > 0 ? (
                    <span
                      className="absolute top-[21px] left-0 z-[1] h-px w-3 -translate-x-full bg-border"
                      aria-hidden
                    />
                  ) : null}
                  <PublicBoardColumnLane
                    column={column}
                    narrowed={narrowed}
                    board={board}
                    automationRowCount={automationRowCount}
                    colorView={colorView}
                  />
                </div>
              ))}
            </WorkOrderKanbanBoard>
          </div>
        </div>
      </main>
    </div>
  );
}

function PublicViewBadge() {
  return (
    <span
      data-testid="public-board-badge"
      className="inline-flex h-8 shrink-0 items-center rounded-full bg-emerald-100 px-2.5 text-[12px] font-medium whitespace-nowrap text-emerald-800 dark:bg-emerald-950 dark:text-emerald-200"
    >
      Public view
    </span>
  );
}

function PublicBoardStatus({ title, signInHref }: { title: string; signInHref?: string }) {
  return (
    <div
      className="flex h-screen flex-col items-center justify-center gap-3 text-[13px] text-muted-foreground"
      data-testid="public-board-status"
    >
      <p>{title}</p>
      {signInHref ? (
        <a href={signInHref} className="text-foreground underline">
          Sign in
        </a>
      ) : null}
    </div>
  );
}

function guestSignInHref(): string {
  const redirect = encodeURIComponent(`${window.location.pathname}${window.location.search}`);
  return `/login?redirect=${redirect}`;
}
