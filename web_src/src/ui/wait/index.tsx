import React from "react";
import { ComponentHeader } from "../componentHeader";
import { CollapsedComponent } from "../collapsedComponent";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";

export type WaitState = "success" | "failed" | "running";

export interface WaitExecutionItem {
  title: string;
  receivedAt?: Date;
  completedAt?: Date;
  state?: WaitState;
  values?: Record<string, string>;
  expectedDuration?: number; // Expected wait duration in milliseconds
}

export interface WaitProps extends ComponentActionsProps {
  title?: string;
  duration?: {
    value: number;
    unit: "seconds" | "minutes" | "hours";
  };
  lastExecution?: WaitExecutionItem;
  nextInQueue?: { title: string; subtitle?: string };
  collapsed?: boolean;
  selected?: boolean;
  collapsedBackground?: string;
  iconBackground?: string;
  iconColor?: string;
  headerColor?: string;
  hideLastRun?: boolean;
}

const formatDuration = (value: number, unit: string): string => {
  const unitLabels: Record<string, string> = {
    seconds: value === 1 ? "second" : "seconds",
    minutes: value === 1 ? "minute" : "minutes",
    hours: value === 1 ? "hour" : "hours",
  };
  return `${value} ${unitLabels[unit] || unit}`;
};

export const Wait: React.FC<WaitProps> = ({
  title = "Wait",
  duration,
  lastExecution,
  nextInQueue,
  collapsed = false,
  selected = false,
  collapsedBackground,
  iconBackground,
  iconColor,
  headerColor,
  hideLastRun = false,
  onToggleCollapse,
  onRun,
  onEdit,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {
  const getStateIcon = React.useCallback((state: WaitState) => {
    if (state === "success") return resolveIcon("check");
    if (state === "running") return resolveIcon("refresh-cw");
    return resolveIcon("x");
  }, []);

  const getStateColor = React.useCallback((state: WaitState) => {
    if (state === "success") return "text-green-700";
    if (state === "running") return "text-blue-800";
    return "text-red-700";
  }, []);

  const getStateBackground = React.useCallback((state: WaitState) => {
    if (state === "success") return "bg-green-200";
    if (state === "running") return "bg-sky-100";
    return "bg-red-200";
  }, []);

  const getStateIconBackground = React.useCallback((state: WaitState) => {
    if (state === "success") return "bg-green-600";
    if (state === "running") return "bg-none animate-spin";
    return "bg-red-600";
  }, []);

  const getStateIconColor = React.useCallback((state: WaitState) => {
    if (state === "success") return "text-white";
    if (state === "running") return "text-blue-800";
    return "text-white";
  }, []);

  // Live countdown timer for running waits
  const [timeLeft, setTimeLeft] = React.useState<number | null>(null);

  React.useEffect(() => {
    if (lastExecution?.state === "running" && lastExecution.receivedAt && lastExecution.expectedDuration) {
      const receivedAt = lastExecution.receivedAt;
      const expectedDuration = lastExecution.expectedDuration;

      // Calculate initial time left
      const elapsed = Date.now() - receivedAt.getTime();
      setTimeLeft(Math.max(0, expectedDuration - elapsed));

      // Update every second
      const interval = setInterval(() => {
        const elapsed = Date.now() - receivedAt.getTime();
        const remaining = Math.max(0, expectedDuration - elapsed);
        setTimeLeft(remaining);
      }, 1000);

      return () => clearInterval(interval);
    } else {
      setTimeLeft(null);
    }
  }, [lastExecution?.state, lastExecution?.receivedAt, lastExecution?.expectedDuration]);

  // Format timestamp for "Done at"
  const formatTimestamp = (date: Date): string => {
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  };

  if (collapsed) {
    return (
      <SelectionWrapper selected={selected}>
        <CollapsedComponent
          iconSlug="alarm-clock"
          iconColor={iconColor || "text-yellow-600"}
          iconBackground={iconBackground || "bg-yellow-100"}
          title={title}
          collapsedBackground={collapsedBackground}
          shape="rounded"
          onDoubleClick={onToggleCollapse}
          onRun={onRun}
          onEdit={onEdit}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        >
          {duration && (
            <div className="flex items-center gap-2 text-xs text-gray-500 mt-1">
              <span>{formatDuration(duration.value, duration.unit)}</span>
            </div>
          )}
        </CollapsedComponent>
      </SelectionWrapper>
    );
  }

  const description = duration
    ? `Waiting for ${formatDuration(duration.value, duration.unit)}...`
    : "No duration configured";

  return (
    <SelectionWrapper selected={selected}>
      <div className="flex flex-col border-1 border-border rounded-md w-[26rem] bg-white overflow-hidden">
        <ComponentHeader
          iconSlug="alarm-clock"
          iconBackground={iconBackground || "bg-yellow-100"}
          iconColor={iconColor || "text-yellow-600"}
          headerColor={headerColor || "bg-yellow-50"}
          title={title}
          description={description}
          onDoubleClick={onToggleCollapse}
          onRun={onRun}
          onEdit={onEdit}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        />

        {/* Last Run Section */}
        {!hideLastRun && (
          <div className="px-4 py-3 border-b">
            <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
              <span className="uppercase text-xs font-semibold tracking-wide">Last Run</span>
            </div>

            {lastExecution && lastExecution.state && lastExecution.receivedAt ? (
              <div className="flex flex-col gap-2">
                <div
                  className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md ${getStateBackground(lastExecution.state)} ${getStateColor(lastExecution.state)}`}
                >
                  <div className="flex items-center gap-2 min-w-0 flex-1">
                    <div
                      className={`w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center ${getStateIconBackground(lastExecution.state)}`}
                    >
                      {React.createElement(getStateIcon(lastExecution.state), {
                        size: lastExecution.state === "running" ? 16 : 12,
                        className: getStateIconColor(lastExecution.state),
                      })}
                    </div>
                    <span className="text-sm font-medium truncate">{lastExecution.title}</span>
                  </div>
                  <span className="text-xs text-gray-500">
                    {lastExecution.state === "running" && timeLeft !== null
                      ? `Time left: ${calcRelativeTimeFromDiff(timeLeft)}`
                      : lastExecution.completedAt
                        ? `Done at: ${formatTimestamp(lastExecution.completedAt)}`
                        : ""}
                  </span>
                </div>
              </div>
            ) : (
              <div className="flex items-center gap-3 px-2 py-2 rounded-md bg-gray-100 text-gray-500">
                <div className="w-5 h-5 rounded-full flex items-center justify-center bg-gray-400">
                  <div className="w-2 h-2 rounded-full bg-white"></div>
                </div>
                <span className="text-sm">No executions received yet</span>
              </div>
            )}
          </div>
        )}

        {nextInQueue && (
          <div className="px-4 pt-3 pb-6">
            <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
              <span className="uppercase text-xs font-semibold tracking-wide">Next In Queue</span>
            </div>
            <div className="flex items-center justify-between gap-3 px-2 py-2 rounded-md bg-gray-100 min-w-0">
              <div className="flex items-center gap-2 text-gray-500 min-w-0 flex-1">
                <div className="w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center">
                  {React.createElement(resolveIcon("circle-dashed"), { size: 20, className: "text-gray-500" })}
                </div>
                <span className="truncate text-sm">{nextInQueue.title}</span>
              </div>
              {nextInQueue.subtitle && (
                <span className="text-sm truncate text-gray-500 flex-shrink-0 max-w-[40%]">{nextInQueue.subtitle}</span>
              )}
            </div>
          </div>
        )}
      </div>
    </SelectionWrapper>
  );
};
