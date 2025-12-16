import React from "react";
import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import { CollapsedComponent } from "../collapsedComponent";
import { ComponentHeader } from "../componentHeader";
import { SpecsTooltip } from "./SpecsTooltip";
import { JsonTooltip } from "./JsonTooltip";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { MetadataItem, MetadataList } from "../metadataList";
import { EmptyState } from "../emptyState";

interface EventSectionDisplayProps {
  section: EventSection;
  index: number;
  totalSections: number;
  className?: string;
  stateMap?: EventStateMap;
}

const EventSectionDisplay: React.FC<EventSectionDisplayProps> = ({
  section,
  index,
  totalSections,
  className,
  stateMap = DEFAULT_EVENT_STATE_MAP,
}) => {
  // Live timer for running executions
  const [liveDuration, setLiveDuration] = React.useState<number | null>(null);

  React.useEffect(() => {
    if (section.eventState === "running" && section.receivedAt) {
      const receivedAt = section.receivedAt;

      // Calculate initial duration
      setLiveDuration(Date.now() - receivedAt.getTime());

      // Update every second
      const interval = setInterval(() => {
        setLiveDuration(Date.now() - receivedAt.getTime());
      }, 1000);

      return () => clearInterval(interval);
    } else {
      setLiveDuration(null);
    }
  }, [section.eventState, section.receivedAt]);

  const now = new Date();
  const diff = section.receivedAt ? now.getTime() - section.receivedAt.getTime() : 0;
  const timeAgo = section.receivedAt ? calcRelativeTimeFromDiff(diff) : "";
  const durationText = liveDuration !== null ? calcRelativeTimeFromDiff(liveDuration) : "";

  const currentState = section.eventState || "neutral";
  const stateStyle = stateMap[currentState];

  const LastEventIcon = resolveIcon(stateStyle.icon);
  const LastEventColor = stateStyle.textColor;
  const LastEventBackground = stateStyle.backgroundColor;
  const LastEventIconColor = stateStyle.iconColor;
  const iconSize = stateStyle.iconSize;
  const iconClassName = stateStyle.iconClassName;

  // Determine what to show in the top-right corner
  let topRightText = "";
  if (section.subtitle) {
    topRightText = section.subtitle;
  } else if (section.showAutomaticTime) {
    topRightText = durationText && section.eventState === "running" ? `Running for: ${durationText}` : timeAgo;
  } else {
    topRightText = timeAgo;
  }

  return (
    <div
      key={index}
      className={"px-4 pt-2 relative" + (index < totalSections - 1 ? " border-b" : "") + ` ${className}`}
    >
      <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
        <span className="uppercase text-xs font-semibold tracking-wide">{section.title}</span>
        {topRightText && <span className="text-sm">{topRightText}</span>}
      </div>
      <div
        className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md ${LastEventBackground} ${LastEventColor}`}
      >
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <div className={`w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center ${LastEventIconColor}`}>
            <LastEventIcon size={iconSize} className={iconClassName} />
          </div>
          <span className="truncate text-sm min-w-0">{section.eventTitle}</span>
        </div>
        {section.eventSubtitle && (
          <span className="text-sm truncate flex-shrink-0 max-w-[40%] text-gray-500">{section.eventSubtitle}</span>
        )}
      </div>
      {section.handleComponent}
    </div>
  );
};

export interface SpecBadge {
  label: string;
  bgColor: string;
  textColor: string;
}

export interface ComponentBaseSpecValue {
  badges: SpecBadge[];
}

export interface ComponentBaseSpec {
  title: string;
  tooltipTitle?: string;
  iconSlug?: string;

  //
  // Either use:
  // - values for badge-based specs (like headers), or
  // - value for JSON specs (like payload)
  //
  values?: ComponentBaseSpecValue[];
  value?: any;
}

export type EventState = "success" | "failed" | "neutral" | "next-in-queue" | "running" | string;

export interface EventStateStyle {
  icon: string;
  textColor: string;
  backgroundColor: string;
  iconColor: string;
  iconSize: number;
  iconClassName: string;
}

export type EventStateMap = Record<EventState, EventStateStyle>;

export const DEFAULT_EVENT_STATE_MAP: EventStateMap = {
  success: {
    icon: "circle-check",
    textColor: "text-green-700",
    backgroundColor: "bg-green-200",
    iconColor: "text-green-600 ",
    iconSize: 16,
    iconClassName: "",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-red-700",
    backgroundColor: "bg-red-200",
    iconColor: "text-red-600 ",
    iconSize: 16,
    iconClassName: "",
  },
  neutral: {
    icon: "circle",
    textColor: "text-gray-500",
    backgroundColor: "bg-gray-100",
    iconColor: "text-white bg-gray-400",
    iconSize: 12,
    iconClassName: "",
  },
  "next-in-queue": {
    icon: "circle-dashed",
    textColor: "text-gray-500",
    backgroundColor: "bg-gray-100",
    iconColor: "text-gray-500",
    iconSize: 16,
    iconClassName: "",
  },
  running: {
    icon: "refresh-cw",
    textColor: "text-blue-800",
    backgroundColor: "bg-sky-100",
    iconColor: "text-blue-800",
    iconSize: 16,
    iconClassName: "animate-spin",
  },
};

export interface EventSection {
  title: string;
  subtitle?: string;
  showAutomaticTime?: boolean;
  receivedAt?: Date;
  eventState?: EventState;
  eventTitle?: string;
  eventSubtitle?: string | React.ReactNode;
  handleComponent?: React.ReactNode;
}

export interface ComponentBaseProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  headerColor: string;
  title: string;
  description?: string;
  specs?: ComponentBaseSpec[];
  hideCount?: boolean;
  hideMetadataList?: boolean;
  collapsed?: boolean;
  collapsedBackground?: string;
  eventSections?: EventSection[];
  selected?: boolean;
  metadata?: MetadataItem[];
  customField?: React.ReactNode;
  eventStateMap?: EventStateMap;
  hideActionsButton?: boolean;
  includeEmptyState?: boolean;
  emptyStateProps?: {
    icon?: React.ComponentType<{ size?: number }>;
    title?: string;
    description?: string;
  };
}

export const ComponentBase: React.FC<ComponentBaseProps> = ({
  iconSrc,
  iconSlug,
  iconColor,
  iconBackground,
  headerColor,
  title,
  description,
  specs,
  collapsed = false,
  collapsedBackground,
  eventSections,
  selected = false,
  onToggleCollapse,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onEdit,
  onConfigure,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
  hideCount,
  hideMetadataList,
  metadata,
  customField,
  eventStateMap,
  hideActionsButton,
  includeEmptyState = false,
  emptyStateProps,
}) => {
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
          shape="circle"
          onDoubleClick={onToggleCollapse}
          onRun={onRun}
          runDisabled={runDisabled}
          runDisabledTooltip={runDisabledTooltip}
          onEdit={onEdit}
          onDuplicate={onDuplicate}
          onConfigure={onConfigure}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
          hideActionsButton={hideActionsButton}
        >
          <div className="flex flex-col items-center gap-1">
            {metadata?.map((item, index) => (
              <div key={`metadata-${index}`} className="flex items-center gap-1 text-xs text-gray-500">
                {React.createElement(resolveIcon(item.icon), { size: 12 })}
                <span className="truncate max-w-[150px]">{item.label}</span>
              </div>
            ))}
            {specs
              ?.filter((spec) => spec.values)
              .map((spec, index) => (
                <div key={`spec-${index}`} className="flex items-center gap-1 text-xs text-gray-500">
                  {React.createElement(resolveIcon(spec.iconSlug || "list-filter"), { size: 12 })}
                  <span>
                    {!hideCount ? spec.values!.length : ""}{" "}
                    {spec.title + (spec.values!.length > 1 && !hideCount ? "s" : "")}
                  </span>
                </div>
              ))}
          </div>
        </CollapsedComponent>
      </SelectionWrapper>
    );
  }

  return (
    <SelectionWrapper selected={selected}>
      <div className="flex flex-col outline outline-black-15 rounded-md w-[23rem] bg-white">
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
          onEdit={onEdit}
          onConfigure={onConfigure}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
          hideActionsButton={hideActionsButton}
        />

        {!hideMetadataList && metadata && metadata.length > 0 && <MetadataList items={metadata} />}

        {specs && specs.length > 0 && (
          <div className="px-2 py-2 border-b text-gray-500 flex flex-col gap-2">
            {specs.map((spec, index) => (
              <div key={index} className="flex items-center gap-2 text-md text-gray-500">
                {React.createElement(resolveIcon(spec.iconSlug || "list-filter"), { size: 18 })}
                {spec.values ? (
                  <SpecsTooltip
                    specTitle={spec.tooltipTitle || spec.title}
                    specValues={spec.values}
                    hideCount={hideCount}
                  >
                    <span className="text-sm bg-gray-500 px-2 py-1 rounded-md text-white font-mono font-medium cursor-help">
                      {hideCount ? "" : spec.values.length}{" "}
                      {spec.title + (spec.values.length > 1 && !hideCount ? "s" : "")}
                    </span>
                  </SpecsTooltip>
                ) : spec.value !== undefined ? (
                  <JsonTooltip title={spec.tooltipTitle || spec.title} value={spec.value}>
                    <span className="text-sm bg-gray-500 px-2 py-1 rounded-md text-white font-mono font-medium cursor-help">
                      {spec.title}
                    </span>
                  </JsonTooltip>
                ) : null}
              </div>
            ))}
          </div>
        )}

        {eventSections?.map((section, index) => (
          <EventSectionDisplay
            className={customField ? "pb-0" : "pb-6"}
            key={index}
            section={section}
            index={index}
            totalSections={eventSections.length}
            stateMap={eventStateMap}
          />
        ))}

        {includeEmptyState && <EmptyState {...emptyStateProps} />}

        {customField || null}
      </div>
    </SelectionWrapper>
  );
};
