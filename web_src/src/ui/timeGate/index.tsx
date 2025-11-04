import React from "react";
import { ComponentHeader } from "../componentHeader";
import { CollapsedComponent } from "../collapsedComponent";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";

export type TimeGateState = "success" | "failed" | "running";

export interface TimeGateExecutionItem {
  receivedAt?: Date;
  state?: TimeGateState;
}

export interface AwaitingEvent {
  title: string;
  subtitle: string;
}

export interface TimeGateProps extends ComponentActionsProps {
  title?: string;
  mode?: "include" | "exclude";
  timeWindow?: string;
  days?: string;
  lastExecution?: TimeGateExecutionItem;
  awaitingEvent?: AwaitingEvent;
  collapsed?: boolean;
  selected?: boolean;
  collapsedBackground?: string;
  iconBackground?: string;
  iconColor?: string;
  headerColor?: string;
  hideLastRun?: boolean;
}

export const TimeGate: React.FC<TimeGateProps> = ({
  title = "Time Gate",
  mode = "include",
  timeWindow,
  days,
  lastExecution,
  awaitingEvent,
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
  const getStateIcon = React.useCallback((state: TimeGateState) => {
    if (state === "success") return resolveIcon("check");
    if (state === "running") return resolveIcon("refresh-cw");
    return resolveIcon("x");
  }, []);

  const getStateColor = React.useCallback((state: TimeGateState) => {
    if (state === "success") return "text-green-700";
    if (state === "running") return "text-blue-800";
    return "text-red-700";
  }, []);

  const getStateBackground = React.useCallback((state: TimeGateState) => {
    if (state === "success") return "bg-green-200";
    if (state === "running") return "bg-sky-100";
    return "bg-red-200";
  }, []);

  const getStateIconBackground = React.useCallback((state: TimeGateState) => {
    if (state === "success") return "bg-green-600";
    if (state === "running") return "bg-none animate-spin";
    return "bg-red-600";
  }, []);

  const getStateIconColor = React.useCallback((state: TimeGateState) => {
    if (state === "success") return "text-white";
    if (state === "running") return "text-blue-800";
    return "text-white";
  }, []);

  const AwaitingIcon = React.useMemo(() => {
    return resolveIcon("circle-dashed");
  }, []);

  if (collapsed) {
    return (
      <SelectionWrapper selected={selected}>
        <CollapsedComponent
          iconSlug="clock"
          iconColor={iconColor || "text-blue-600"}
          iconBackground={iconBackground || "bg-blue-100"}
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
          <div className="flex flex-col gap-1 text-xs text-gray-500 mt-1">
            <div className="flex items-center gap-2">
              <span className="capitalize font-medium">{mode}</span>
              {timeWindow && <span>{timeWindow}</span>}
            </div>
            {days && <span className="truncate">{days}</span>}
          </div>
        </CollapsedComponent>
      </SelectionWrapper>
    );
  }

  const description = timeWindow
    ? `${mode === "include" ? "Allow" : "Block"} events during ${timeWindow}`
    : "No time window configured";

  return (
    <SelectionWrapper selected={selected}>
      <div className="flex flex-col border-2 border-border rounded-md w-[26rem] bg-white">
        <ComponentHeader
          iconSlug="clock"
          iconBackground={iconBackground || "bg-blue-100"}
          iconColor={iconColor || "text-blue-600"}
          headerColor={headerColor || "bg-blue-50"}
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

        {/* Configuration Section */}
        <div className="px-4 py-3 border-b bg-gray-50">
          <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
            <span className="uppercase text-sm font-medium">Configuration</span>
          </div>

          <div className="flex flex-col gap-2 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-gray-600">Mode:</span>
              <span className={`px-2 py-1 rounded text-xs font-medium ${
                mode === "include"
                  ? "bg-green-100 text-green-800"
                  : "bg-red-100 text-red-800"
              }`}>
                {mode === "include" ? "Include" : "Exclude"}
              </span>
            </div>

            {timeWindow && (
              <div className="flex items-center justify-between">
                <span className="text-gray-600">Time Window:</span>
                <span className="font-mono text-xs bg-gray-100 px-2 py-1 rounded">
                  {timeWindow}
                </span>
              </div>
            )}

            {days && (
              <div className="flex items-start justify-between gap-2">
                <span className="text-gray-600 flex-shrink-0">Days:</span>
                <span className="text-xs text-right">{days}</span>
              </div>
            )}
          </div>
        </div>

        {/* Awaiting Section - Show when waiting for time window */}
        <div className="px-4 py-3">
          {awaitingEvent ? (
            <>
              <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
                <span className="uppercase text-sm font-medium">
                  Waiting for Time Window
                </span>
              </div>

              <div className="flex items-center justify-between gap-3 px-2 py-2 rounded-md bg-orange-200 mb-4">
                <div className="flex items-center gap-2 min-w-0 flex-1 text-amber-800">
                  <div className="w-5 h-5 rounded-full flex items-center justify-center">
                    <AwaitingIcon size={20} className="text-amber-800" />
                  </div>
                  <span className="truncate text-sm">{awaitingEvent.title}</span>
                </div>
                {awaitingEvent.subtitle && (
                  <span className="truncate text-sm flex-shrink-0 text-amber-800">
                    {awaitingEvent.subtitle}
                  </span>
                )}
              </div>
            </>
          ) : (
            <>
              {/* Last Run Section - Only show when not waiting */}
              {!hideLastRun && lastExecution && lastExecution.state && lastExecution.receivedAt && (
                <>
                  <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
                    <span className="uppercase text-sm font-medium">Last Run</span>
                  </div>

                  <div className="flex flex-col gap-2 mb-4">
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
                        <span className="text-sm">
                          {lastExecution.state === "running"
                            ? "Processing..."
                            : lastExecution.state === "success"
                              ? "Event passed through"
                              : "Event blocked"}
                        </span>
                      </div>
                      <span className="text-xs text-gray-500">
                        {calcRelativeTimeFromDiff(
                          new Date().getTime() - lastExecution.receivedAt.getTime()
                        )}
                      </span>
                    </div>
                  </div>
                </>
              )}

              {/* No executions state */}
              {(!lastExecution || !lastExecution.state || !lastExecution.receivedAt) && !hideLastRun && (
                <>
                  <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
                    <span className="uppercase text-sm font-medium">Last Run</span>
                  </div>
                  <div className="flex items-center gap-3 px-2 py-2 rounded-md bg-gray-100 text-gray-500 mb-4">
                    <div className="w-5 h-5 rounded-full flex items-center justify-center bg-gray-400">
                      <div className="w-2 h-2 rounded-full bg-white"></div>
                    </div>
                    <span className="text-sm">No events received yet</span>
                  </div>
                </>
              )}
            </>
          )}
        </div>
      </div>
    </SelectionWrapper>
  );
};