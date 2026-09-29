import { Input } from "@/components/ui/input";
import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "react-router";

import { PermissionDeniedPage } from "@/components/PermissionDeniedPage";
import { lineBoardColumnLaneProps } from "./lineBoardColumnColors";
import { WorkOrderBoardLane, WorkOrderKanbanBoard } from "../workOrders/WorkOrderBoardChrome";
import { PublicBoardCard, type PublicBoardCardModel } from "../workOrders/WorkOrderCard";

interface PublicBoardColumn {
  key: string;
  title: string;
  color?: string;
  labels: string[];
  cards: PublicBoardCardModel[];
}

interface PublicBoard {
  workspaceName: string;
  lineName: string;
  showClarity: boolean;
  showConfidence: boolean;
  columns: PublicBoardColumn[];
}

type BoardLoad =
  | { status: "loading" }
  | { status: "ready"; board: PublicBoard }
  | { status: "denied" }
  | { status: "missing" };

export function PublicFactoryBoardPage({ signedIn }: { signedIn: boolean }) {
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
  const [query, setQuery] = useState("");
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
    if (response.status === 404) {
      if (signedInRef.current) {
        setLoad({ status: "denied" });
        return;
      }
      redirectToLogin();
      return;
    }
    if (!response.ok) {
      setLoad({ status: "missing" });
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
      if (response.status === 404) {
        if (signedInRef.current) {
          setLoad({ status: "denied" });
          return;
        }
        redirectToLogin();
        return;
      }
      if (!response.ok) {
        setLoad({ status: "missing" });
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
  const columns = useMemo(() => filterBoardColumns(board?.columns ?? [], query), [board, query]);

  if (load.status === "loading") {
    return <PublicBoardStatus title="Loading board" />;
  }
  if (load.status === "denied") {
    return <PermissionDeniedPage description="You do not have permission to open this workspace." />;
  }
  if (load.status === "missing" || !board) {
    return <PublicBoardStatus title="This board is not available." />;
  }

  return <PublicBoardView board={board} columns={columns} query={query} onQueryChange={setQuery} />;
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
  board,
  columns,
  query,
  onQueryChange,
}: {
  board: PublicBoard;
  columns: PublicBoardColumn[];
  query: string;
  onQueryChange: (next: string) => void;
}) {
  return (
    <div className="flex h-screen min-h-0 flex-col bg-background text-foreground" data-testid="public-factory-board">
      <header className="flex shrink-0 items-center gap-3 border-b border-border px-4 py-3">
        <h1 className="min-w-0 truncate text-[15px] font-medium">{board.workspaceName}</h1>
        <Input
          value={query}
          onChange={(event) => onQueryChange(event.target.value)}
          placeholder="Search tasks"
          aria-label="Search tasks"
          className="ml-auto max-w-xs"
          data-testid="public-board-search"
        />
      </header>
      <div className="flex min-h-0 flex-1 p-4">
        <WorkOrderKanbanBoard testId="public-board-columns">
          {columns.map((column) => (
            <PublicBoardColumnLane key={column.key} column={column} query={query} board={board} />
          ))}
        </WorkOrderKanbanBoard>
      </div>
    </div>
  );
}

function PublicBoardColumnLane({
  column,
  query,
  board,
}: {
  column: PublicBoardColumn;
  query: string;
  board: PublicBoard;
}) {
  return (
    <WorkOrderBoardLane
      title={column.title}
      count={column.cards.length}
      emptyDescription={query.trim() ? "No matching tasks." : "Nothing here."}
      subheader={
        column.labels.length > 0 ? (
          <div className="space-y-1 px-3 py-2">
            {column.labels.map((label) => (
              <p key={label} className="truncate text-[12px] text-muted-foreground">
                {label}
              </p>
            ))}
          </div>
        ) : null
      }
      testId={`public-board-column-${column.key}`}
      {...lineBoardColumnLaneProps(column.color)}
    >
      <div className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto p-2 [scrollbar-width:thin]">
        {column.cards.map((card) => (
          <PublicBoardCard
            key={`${column.key}-${card.title}-${card.createdAt}`}
            card={card}
            showClarity={board.showClarity}
            showConfidence={board.showConfidence}
          />
        ))}
      </div>
    </WorkOrderBoardLane>
  );
}

function PublicBoardStatus({ title }: { title: string }) {
  return (
    <div
      className="flex h-screen items-center justify-center text-[13px] text-muted-foreground"
      data-testid="public-board-status"
    >
      {title}
    </div>
  );
}

function filterBoardColumns(columns: PublicBoardColumn[], query: string): PublicBoardColumn[] {
  const needle = query.trim().toLowerCase();
  if (!needle) {
    return columns;
  }
  return columns.map((column) => ({
    ...column,
    cards: column.cards.filter((card) => card.title.toLowerCase().includes(needle)),
  }));
}

function publicBoardUrl(organizationId: string, factoryKey: string, lineId: string): string {
  return `/api/v1/public/organizations/${encodeURIComponent(organizationId)}/workspaces/${encodeURIComponent(factoryKey)}/lines/${encodeURIComponent(lineId)}/board`;
}

function publicBoardSocketPath(organizationId: string, factoryKey: string, lineId: string): string {
  return `/ws/public/organizations/${encodeURIComponent(organizationId)}/workspaces/${encodeURIComponent(factoryKey)}/lines/${encodeURIComponent(lineId)}`;
}

function parseBoardEvent(data: unknown): string {
  if (typeof data !== "string") {
    return "";
  }
  try {
    const message = JSON.parse(data) as { event?: string };
    return message.event ?? "";
  } catch {
    return "";
  }
}

function redirectToLogin(): void {
  const redirectTarget = `${window.location.pathname}${window.location.search}`;
  window.location.href = `/login?redirect=${encodeURIComponent(redirectTarget)}`;
}
