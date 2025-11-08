import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import React from "react";
import { ComponentHeader } from "../componentHeader";
import { CollapsedComponent } from "../collapsedComponent";
import { MetadataList, type MetadataItem } from "../metadataList";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";

type LastEventState = "processed" | "discarded";

interface TriggerLastEventData {
  title: string;
  subtitle?: string;
  receivedAt: Date;
  state: LastEventState;
}

export interface TriggerProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  headerColor: string;
  title: string;
  description?: string;
  metadata: MetadataItem[];
  lastEventData?: TriggerLastEventData;
  zeroStateText?: string;
  collapsedBackground?: string;
  collapsed?: boolean;
  selected?: boolean;
}

export const Trigger: React.FC<TriggerProps> = ({
  iconSrc,
  iconSlug,
  iconColor,
  iconBackground,
  headerColor,
  title,
  description,
  metadata,
  lastEventData,
  zeroStateText = "No events yet",
  collapsed = false,
  collapsedBackground,
  selected = false,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onEdit,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {
  const timeAgo = React.useMemo(() => {
    if (!lastEventData) return null;
    const now = new Date();
    const diff = now.getTime() - lastEventData.receivedAt.getTime();
    return calcRelativeTimeFromDiff(diff);
  }, [lastEventData]);

  const LastEventIcon = React.useMemo(() => {
    if (!lastEventData) return null;
    if (lastEventData.state === "processed") {
      return resolveIcon("check");
    } else {
      return resolveIcon("x");
    }
  }, [lastEventData]);

  const LastEventColor = React.useMemo(() => {
    if (!lastEventData) return "text-gray-700";
    if (lastEventData.state === "processed") {
      return "text-green-700";
    } else {
      return "text-red-700";
    }
  }, [lastEventData]);

  const LastEventBackground = React.useMemo(() => {
    if (!lastEventData) return "bg-gray-200";
    if (lastEventData.state === "processed") {
      return "bg-green-200";
    } else {
      return "bg-red-200";
    }
  }, [lastEventData]);

  if (collapsed) {
    return (
      <SelectionWrapper selected={selected} fullRounded>
        <CollapsedComponent
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconColor={iconColor}
          iconBackground={iconBackground}
          title={title}
          collapsedBackground={collapsedBackground}
          shape="circle"
          onRun={onRun}
          runDisabled={runDisabled}
          runDisabledTooltip={runDisabledTooltip}
          onEdit={onEdit}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        >
          <div className="flex flex-col items-center gap-1">
            <MetadataList items={metadata} className="flex flex-col gap-1 text-gray-500" iconSize={12} />
          </div>
        </CollapsedComponent>
      </SelectionWrapper>
    );
  }

  return (
    <SelectionWrapper selected={selected}>
      <div className="flex flex-col border-1 border-border rounded-md w-[23rem] bg-white">
        <ComponentHeader
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconBackground={iconBackground}
          iconColor={iconColor}
          headerColor={headerColor}
          title={title}
          description={description}
          onRun={onRun}
          runDisabled={runDisabled}
          runDisabledTooltip={runDisabledTooltip}
          onEdit={onEdit}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        />
        <MetadataList items={metadata} />
        <div className="px-4 pt-3 pb-6">
          {lastEventData ? (
            <>
              <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
                <span className="uppercase text-xs font-semibold tracking-wide">Last Event</span>
                <span className="text-sm">{timeAgo}</span>
              </div>
              <div
                className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md ${LastEventBackground} ${LastEventColor}`}
              >
                <div className="flex items-center gap-2 min-w-0 flex-1">
                  <div
                    className={`w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center ${lastEventData.state === "processed" ? "bg-green-600" : "bg-red-600"}`}
                  >
                    {LastEventIcon && <LastEventIcon size={12} className="text-white" />}
                  </div>
                  <span className="truncate text-sm min-w-0">{lastEventData.title}</span>
                </div>
                {lastEventData.subtitle && (
                  <span className="text-sm truncate flex-shrink-0 max-w-[40%]">{lastEventData.subtitle}</span>
                )}
              </div>
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
