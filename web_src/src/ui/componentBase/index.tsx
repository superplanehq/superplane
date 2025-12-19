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
import { ChildEvents, type ChildEventsInfo } from "../childEvents";

interface EventSectionDisplayProps {
  section: EventSection;
  index: number;
  totalSections: number;
  className?: string;
  stateMap?: EventStateMap;
  lastSection?: boolean;
}

const EventSectionDisplay: React.FC<EventSectionDisplayProps> = ({
  section,
  index,
  totalSections,
  className,
  stateMap = DEFAULT_EVENT_STATE_MAP,
  lastSection = false,
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

  const currentState = section.eventState || "neutral";
  const stateStyle = stateMap[currentState] || stateMap["neutral"];

  const LastEventBackground = stateStyle.backgroundColor;
  const LastEventStateColor = stateStyle.badgeColor;
  const durationText = liveDuration !== null ? calcRelativeTimeFromDiff(liveDuration) : "";

  return (
    <div
      key={index}
      className={
        `px-3 pt-2 relative ${lastSection ? "rounded-b-md" : ""} ${LastEventBackground}` +
        (index < totalSections - 1 ? " border-b border-slate-400" : "") +
        ` ${className}`
      }
    >
      <div className="flex items-center justify-between gap-2 min-w-0 flex-1">
        <div
          className={`uppercase text-[11px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white ${LastEventStateColor}`}
        >
          <span>{currentState}</span>
        </div>
        {section.eventSubtitle && (
          <span className="text-[13px] font-medium truncate flex-shrink-0 max-w-[40%] text-gray-950/50">
            {section.showAutomaticTime && durationText ? durationText : section.eventSubtitle}
          </span>
        )}
      </div>
      <div className="flex justify-left items-center mt-1 gap-2">
        {section.eventId && <span className="text-[13px] text-gray-950/50 font-mono">#{section.eventId?.slice(0, 4)}</span>}
        <span className="text-sm text-gray-700 font-inter truncate text-md min-w-0 font-medium truncate">
          {section.eventTitle}
        </span>
      </div>
      {section.childEventsInfo && (
        <ChildEvents
          childEventsInfo={section.childEventsInfo}
          onExpandChildEvents={section.onExpandChildEvents}
          onReRunChildEvents={section.onReRunChildEvents}
          showItems={false}
        />
      )}
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
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  value?: any;
}

export type EventState = "success" | "failed" | "neutral" | "queued" | "running" | string;

export interface EventStateStyle {
  icon: string;
  textColor: string;
  backgroundColor: string;
  badgeColor: string;
}

export type EventStateMap = Record<EventState, EventStateStyle>;

export const DEFAULT_EVENT_STATE_MAP: EventStateMap = {
  triggered: {
    icon: "circle",
    textColor: "text-black",
    backgroundColor: "bg-violet-100",
    badgeColor: "bg-violet-400",
  },
  success: {
    icon: "circle-check",
    textColor: "text-black",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-black",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  neutral: {
    icon: "circle",
    textColor: "text-black",
    backgroundColor: "bg-gray-50",
    badgeColor: "bg-gray-400",
  },
  queued: {
    icon: "circle-dashed",
    textColor: "text-black",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
  running: {
    icon: "refresh-cw",
    textColor: "text-black",
    backgroundColor: "bg-sky-100",
    badgeColor: "bg-blue-500",
  },
};

export interface EventSection {
  showAutomaticTime?: boolean;
  receivedAt?: Date;
  eventId?: string;
  eventState?: EventState;
  eventTitle?: string;
  eventSubtitle?: string | React.ReactNode;
  handleComponent?: React.ReactNode;
  childEventsInfo?: ChildEventsInfo;
  onExpandChildEvents?: (childEventsInfo: ChildEventsInfo) => void;
  onReRunChildEvents?: (childEventsInfo: ChildEventsInfo) => void;
}

export interface ComponentBaseProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  headerColor?: string;
  title: string;
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
      <SelectionWrapper selected={selected} fullRounded>
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
      <div className="flex flex-col outline-1 outline-slate-400 rounded-md w-[23rem] bg-white">
        <ComponentHeader
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconBackground={iconBackground}
          iconColor={iconColor}
          headerColor={headerColor || "bg-white"}
          title={title}
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
          <div className="px-2 py-1.5 border-b border-slate-400 text-gray-500 flex flex-col gap-2">
            {specs.map((spec, index) => (
              <div key={index} className="flex items-center gap-1 text-md text-gray-500">
                {React.createElement(resolveIcon(spec.iconSlug || "list-filter"), { size: 18 })}
                {spec.values ? (
                  <SpecsTooltip
                    specTitle={spec.tooltipTitle || spec.title}
                    specValues={spec.values}
                    hideCount={hideCount}
                  >
                    <span className="text-[13px] underline underline-offset-3 decoration-dotted decoration-1 decoration-gray-500 px-2 rounded-md font-inter font-medium cursor-help">
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
            className={"pb-3" + (!!includeEmptyState || !!customField ? " border-b border-slate-400" : "")}
            key={index}
            section={section}
            index={index}
            totalSections={eventSections.length}
            stateMap={eventStateMap}
            lastSection={index === eventSections.length - 1 && !includeEmptyState && !customField}
          />
        ))}

        {includeEmptyState && <EmptyState {...emptyStateProps} />}

        {customField || null}
      </div>
    </SelectionWrapper>
  );
};
