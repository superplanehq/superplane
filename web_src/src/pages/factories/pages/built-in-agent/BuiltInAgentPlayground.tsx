import { useCallback } from "react";
import { Bot, ChartLine, Home } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

import { factoriesRailControlClassName } from "../../layout/factoriesRail";
import { BuiltInAgentBoardPane } from "./BuiltInAgentBoardPane";
import { BuiltInAgentPanel } from "./BuiltInAgentPanel";
import { BUILT_IN_AGENT_COPY } from "./builtInAgentMocks";
import { useBuiltInAgentPrototype, type BuiltInAgentPrototypeSeed } from "./useBuiltInAgentPrototype";

export function BuiltInAgentPlayground({ seed }: { seed?: BuiltInAgentPrototypeSeed }) {
  const prototype = useBuiltInAgentPrototype(seed);
  const { panelOpen, closePanel, openPanel } = prototype;
  const onToggleAgent = useCallback(() => {
    if (panelOpen) {
      closePanel();
      return;
    }
    openPanel();
  }, [closePanel, openPanel, panelOpen]);

  return (
    <div className="flex h-svh min-h-0 w-full overflow-hidden bg-background">
      <PrototypeRail panelOpen={prototype.panelOpen} onToggleAgent={onToggleAgent} />
      {prototype.panelOpen ? (
        <BuiltInAgentPanel
          tasks={prototype.tasks}
          transcript={prototype.transcript}
          pendingDeleteId={prototype.pendingDeleteId}
          pendingPlan={prototype.pendingPlan}
          hasError={prototype.hasError}
          onClose={prototype.closePanel}
          onCreateTask={prototype.createTask}
          onDeleteTask={prototype.deleteTask}
          onConfirmDelete={prototype.confirmDelete}
          onCancelDelete={prototype.cancelDelete}
          onSendMessage={prototype.sendMessage}
          onAcceptPlan={prototype.acceptPlan}
          onDismissPlan={prototype.dismissPlan}
          onRetryAfterError={prototype.retryAfterError}
        />
      ) : null}
      <BuiltInAgentBoardPane tasks={prototype.tasks} />
    </div>
  );
}

function PrototypeRail({ panelOpen, onToggleAgent }: { panelOpen: boolean; onToggleAgent: () => void }) {
  return (
    <aside
      className="flex h-full w-[var(--workspace-navigation-width)] shrink-0 flex-col items-center border-r border-sidebar-border bg-sidebar text-sidebar-foreground"
      aria-label="Workspace"
    >
      <div className="flex flex-col items-center gap-1 px-1.5 pt-3 pb-1">
        <span
          className={cn(
            factoriesRailControlClassName,
            "bg-sidebar-accent text-[11px] font-medium tracking-[-0.01em] text-foreground",
          )}
          aria-hidden
        >
          {BUILT_IN_AGENT_COPY.workspaceLabel}
        </span>
      </div>
      <nav className="flex flex-col items-center gap-1 px-1.5" aria-label="Workspace">
        <RailButton label={BUILT_IN_AGENT_COPY.boardLabel} Icon={Home} current={!panelOpen} />
        <RailButton label={BUILT_IN_AGENT_COPY.velocityLabel} Icon={ChartLine} />
        <RailButton label={BUILT_IN_AGENT_COPY.openPanel} Icon={Bot} current={panelOpen} onClick={onToggleAgent} />
      </nav>
    </aside>
  );
}

function RailButton({
  label,
  Icon,
  current = false,
  onClick,
}: {
  label: string;
  Icon: LucideIcon;
  current?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      aria-current={current ? "page" : undefined}
      onClick={onClick}
      className={cn(factoriesRailControlClassName, current && "bg-sidebar-accent text-foreground")}
    >
      <Icon className="size-3.5" aria-hidden />
    </button>
  );
}
