import { CircleDashedIcon } from "lucide-react";
import React from "react";
import { ApprovalItem, type ApprovalItemProps } from "../approvalItem";
import { CollapsedComponent } from "../collapsedComponent";
import { ComponentHeader } from "../componentHeader";
import { ItemGroup } from "../item";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { resolveIcon } from "@/lib/utils";
import { ListFilter } from "lucide-react";
import { SpecsTooltip } from "../componentBase/SpecsTooltip";
import type { ComponentBaseSpecValue } from "../componentBase";

export interface AwaitingEvent {
  title: string;
  subtitle?: string;
}

type LastRunState = "processed" | "discarded" | "running";

interface ApprovalLastRunData {
  title: string;
  subtitle?: string;
  receivedAt: Date;
  state: LastRunState;
}

export interface ApprovalProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconBackground?: string;
  iconColor?: string;
  headerColor: string;
  title: string;
  description?: string;
  approvals: ApprovalItemProps[];
  awaitingEvent?: AwaitingEvent;
  lastRunData?: ApprovalLastRunData;
  collapsedBackground?: string;
  receivedAt?: Date;
  zeroStateText?: string;
  collapsed?: boolean;
  selected?: boolean;
  spec?: {
    title: string;
    tooltipTitle?: string;
    values: ComponentBaseSpecValue[];
  };
}

export const Approval: React.FC<ApprovalProps> = ({
  iconSrc,
  iconSlug,
  iconBackground,
  iconColor,
  headerColor,
  title,
  description,
  collapsed = false,
  collapsedBackground,
  receivedAt,
  approvals,
  awaitingEvent,
  lastRunData,
  zeroStateText = "Awaiting events for approval",
  selected = false,
  onToggleCollapse,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  onEdit,
  isCompactView,
  spec,
}) => {
  const calcRelativeTimeFromDiff = (diff: number) => {
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);
    if (days > 0) {
      return `${days}d`;
    } else if (hours > 0) {
      return `${hours}h`;
    } else if (minutes > 0) {
      return `${minutes}m`;
    } else {
      return `${seconds}s`;
    }
  };

  const timeAgo = React.useMemo(() => {
    if (!receivedAt) return null;
    const now = new Date();
    const diff = now.getTime() - receivedAt.getTime();
    return calcRelativeTimeFromDiff(diff);
  }, [receivedAt]);

  const lastRunTimeAgo = React.useMemo(() => {
    if (!lastRunData) return null;
    const now = new Date();
    const diff = now.getTime() - lastRunData.receivedAt.getTime();
    return calcRelativeTimeFromDiff(diff);
  }, [lastRunData]);

  const LastRunIcon = React.useMemo(() => {
    if (!lastRunData) return null;
    if (lastRunData.state === "processed") {
      return resolveIcon("check");
    } else if (lastRunData.state === "discarded") {
      return resolveIcon("x");
    } else {
      return resolveIcon("loader-2");
    }
  }, [lastRunData]);

  const LastRunColor = React.useMemo(() => {
    if (!lastRunData) return "text-gray-700";
    if (lastRunData.state === "processed") {
      return "text-green-700";
    } else if (lastRunData.state === "discarded") {
      return "text-red-700";
    } else {
      return "text-blue-700";
    }
  }, [lastRunData]);

  const LastRunBackground = React.useMemo(() => {
    if (!lastRunData) return "bg-gray-200";
    if (lastRunData.state === "processed") {
      return "bg-green-200";
    } else if (lastRunData.state === "discarded") {
      return "bg-red-200";
    } else {
      return "bg-blue-200";
    }
  }, [lastRunData]);

  if (collapsed) {
    return (
      <SelectionWrapper selected={selected}>
        <CollapsedComponent
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconColor={iconColor}
          iconBackground={iconBackground}
          title={title}
          collapsedBackground={collapsedBackground}
          shape="rounded"
          onDoubleClick={onToggleCollapse}
          onRun={onRun}
          runDisabled={runDisabled}
          runDisabledTooltip={runDisabledTooltip}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          onEdit={onEdit}
          isCompactView={isCompactView}
        >
          <div className="flex flex-col items-center gap-1">
            {spec?.values?.length ? (
              <div className="flex items-center gap-1 text-xs text-gray-500">
                <ListFilter size={12} />
                <span>{spec.values.length} approvals required</span>
              </div>
            ) : null}
          </div>
        </CollapsedComponent>
      </SelectionWrapper>
    );
  }

  return (
    <SelectionWrapper selected={selected}>
      <div className="flex flex-col border-1 border-border rounded-md w-[26rem] bg-white overflow-hidden">
        <ComponentHeader
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconBackground={iconBackground}
          iconColor={iconColor}
          headerColor={headerColor}
          title={title}
          description={description}
          onDoubleClick={onToggleCollapse}
          onRun={onRun}
          runDisabled={runDisabled}
          runDisabledTooltip={runDisabledTooltip}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          onEdit={onEdit}
          isCompactView={isCompactView}
        />

        {spec && spec.values?.length > 0 && (
          <div className="px-2 py-2 border-b text-gray-500 flex flex-col gap-2">
            <div className="flex items-center gap-3 text-md text-gray-500">
              <ListFilter size={18} />
              <SpecsTooltip specTitle={spec.tooltipTitle || spec.title} specValues={spec.values}>
                <span className="text-sm bg-gray-500 px-2 py-1 rounded-md text-white font-mono font-medium">
                  {spec.values.length} approvals required
                </span>
              </SpecsTooltip>
            </div>
          </div>
        )}

        <div className="px-4 py-3">
          {lastRunData && !awaitingEvent && (
            <>
              <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
                <span className="uppercase text-xs font-semibold tracking-wide">Last Run</span>
                <span className="text-sm">{lastRunTimeAgo}</span>
              </div>
              <div
                className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md ${LastRunBackground} ${LastRunColor} mb-4`}
              >
                <div className="flex items-center gap-2 min-w-0 flex-1">
                  <div
                    className={`w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center ${lastRunData.state === "processed" ? "bg-green-600" : lastRunData.state === "discarded" ? "bg-red-600" : "bg-blue-600"}`}
                  >
                    {LastRunIcon && <LastRunIcon size={12} className="text-white" />}
                  </div>
                  <span className="truncate text-sm min-w-0">{lastRunData.title}</span>
                </div>
                {lastRunData.subtitle && (
                  <span className="text-sm truncate flex-shrink-0 max-w-[40%]">{lastRunData.subtitle}</span>
                )}
              </div>
            </>
          )}

          {!lastRunData && !awaitingEvent && (
            <>
              <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
                <span className="uppercase text-xs font-semibold tracking-wide">Last Run</span>
                <span className="text-sm"></span>
              </div>
              <div className="flex items-center justify-between gap-3 px-2 py-2 rounded-md bg-gray-100 text-gray-500 mb-4">
                <div className="flex items-center gap-2 min-w-0 flex-1">
                  <div className="w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center text-gray-400 bg-gray-400">
                    {resolveIcon("circle") &&
                      React.createElement(resolveIcon("circle")!, { size: 12, className: "text-white" })}
                  </div>
                  <span className="truncate text-sm">No events received yet</span>
                </div>
              </div>
            </>
          )}

          {awaitingEvent ? (
            <>
              <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
                <span className="uppercase text-xs font-semibold tracking-wide">Awaiting Approval</span>
                <span className="text-sm">{timeAgo}</span>
              </div>

              <div className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md bg-orange-200 mb-4`}>
                <div className="flex items-center gap-2 w-[80%] text-amber-800">
                  <div className={`w-5 h-5 rounded-full flex items-center justify-center`}>
                    <CircleDashedIcon size={20} className="text-amber-800" />
                  </div>
                  <span className="truncate text-sm">{awaitingEvent.title}</span>
                </div>
                {awaitingEvent.subtitle && (
                  <span className="truncate text-sm no-wrap whitespace-nowrap w-[20%] text-amber-800">
                    {awaitingEvent.subtitle}
                  </span>
                )}
              </div>

              <ItemGroup className="w-full">
                {approvals.map((approval, index) => (
                  <React.Fragment key={`${approval.title}-${index}`}>
                    <ApprovalItem {...approval} />
                  </React.Fragment>
                ))}
              </ItemGroup>
            </>
          ) : (
            <div className="flex items-center justify-center px-2 py-4 rounded-md bg-gray-50 border border-dashed border-gray-300">
              <span className="text-sm text-gray-400">{zeroStateText}</span>
            </div>
          )}
        </div>
      </div>
    </SelectionWrapper>
  );
};
