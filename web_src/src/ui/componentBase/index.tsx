import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import { AlertTriangle } from "lucide-react";
import React from "react";
import { ChildEvents, type ChildEventsInfo } from "../childEvents";
import { ComponentHeader } from "../componentHeader";
import { EmptyState } from "../emptyState";
import { MetadataItem, MetadataList } from "../metadataList";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { PayloadTooltip } from "./PayloadTooltip";
import { SpecsTooltip } from "./SpecsTooltip";
import { formatTimeAgo } from "@/utils/date";

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
        `px-2 pt-2 relative ${lastSection ? "rounded-b-md" : ""} ${LastEventBackground}` +
        (index < totalSections - 1 ? " border-b border-slate-950/20" : "") +
        ` ${className}`
      }
    >
      <div className="flex items-center justify-between gap-2 min-w-0 flex-1">
        <div
          className={`uppercase text-[11px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white ${LastEventStateColor}`}
        >
          <span>{stateStyle.label || currentState}</span>
        </div>
        {section.eventSubtitle ? (
          <span
            title={String(section.eventSubtitle)}
            className="text-[13px] font-medium truncate flex-shrink-0 max-w-[65%] text-gray-950/50"
          >
            {section.showAutomaticTime && durationText ? durationText : section.eventSubtitle}
          </span>
        ) : (
          <span className="text-[13px] font-medium truncate flex-shrink-0 max-w-[65%] text-gray-950/50">
            {formatTimeAgo(new Date(section.receivedAt!))}
          </span>
        )}
      </div>
      <div className="flex justify-left items-center mt-1 gap-2">
        {section.eventId && (
          <span className="text-[13px] text-gray-950/50 font-mono">#{section.eventId?.slice(0, 4)}</span>
        )}
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
  // - value for JSON/text/XML specs (like payload)
  //
  values?: ComponentBaseSpecValue[];
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  value?: any;
  // Content type for value tooltips (json, xml, or text)
  contentType?: "json" | "xml" | "text";
}

export type EventState = "success" | "failed" | "neutral" | "queued" | "running" | string;

export interface EventStateStyle {
  icon: string;
  textColor: string;
  backgroundColor: string;
  badgeColor: string;
  label?: string; // Optional display label, defaults to state name if not provided
}

export type EventStateMap = Record<EventState, EventStateStyle>;

export const DEFAULT_EVENT_STATE_MAP: EventStateMap = {
  triggered: {
    icon: "circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-violet-100",
    badgeColor: "bg-violet-400",
  },
  success: {
    icon: "circle-check",
    textColor: "text-gray-800",
    backgroundColor: "bg-green-100",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-400",
  },
  cancelled: {
    icon: "circle-slash-2",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-100",
    badgeColor: "bg-gray-500",
  },
  error: {
    icon: "triangle-alert",
    textColor: "text-gray-800",
    backgroundColor: "bg-red-100",
    badgeColor: "bg-red-500",
  },
  neutral: {
    icon: "circle",
    textColor: "text-gray-800",
    backgroundColor: "bg-gray-50",
    badgeColor: "bg-gray-400",
  },
  queued: {
    icon: "circle-dashed",
    textColor: "text-gray-800",
    backgroundColor: "bg-orange-100",
    badgeColor: "bg-yellow-600",
  },
  running: {
    icon: "refresh-cw",
    textColor: "text-gray-800",
    backgroundColor: "bg-sky-100",
    badgeColor: "bg-blue-500",
  },
};

export interface EventSection {
  showAutomaticTime?: boolean;
  receivedAt?: Date;
  eventId: string;
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
  title: string;
  showHeader?: boolean;
  paused?: boolean;
  specs?: ComponentBaseSpec[];
  hideCount?: boolean;
  hideMetadataList?: boolean;
  collapsed?: boolean;
  collapsedBackground?: string;
  eventSections?: EventSection[];
  selected?: boolean;
  metadata?: MetadataItem[];
  /** Custom content rendered on the node */
  customField?: React.ReactNode | ((onRun?: () => void, nodeId?: string) => React.ReactNode);
  /** Where to render customField: "before" (before events) or "after" (after events, default) */
  customFieldPosition?: "before" | "after";
  eventStateMap?: EventStateMap;
  includeEmptyState?: boolean;
  emptyStateProps?: {
    icon?: React.ComponentType<{ size?: number }>;
    title?: string;
    description?: string;
  };
  error?: string;
  warning?: string;
}

export const ComponentBase: React.FC<ComponentBaseProps> = ({
  iconSrc,
  iconSlug,
  iconColor,
  title,
  showHeader = true,
  specs,
  collapsed: _collapsed = false,
  collapsedBackground: _collapsedBackground,
  eventSections,
  selected = false,
  onToggleCollapse,
  onRun,
  runDisabled,
  runDisabledTooltip: _runDisabledTooltip,
  onTogglePause,
  onEdit: _onEdit,
  onConfigure: _onConfigure,
  onDuplicate,
  onDeactivate: _onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
  hideCount,
  hideMetadataList,
  metadata,
  customField,
  customFieldPosition = "after",
  eventStateMap,
  includeEmptyState = false,
  emptyStateProps,
  error,
  warning,
  paused,
}) => {
  const hasError = error && error.trim() !== "";
  const hasWarning = warning && warning.trim() !== "";
  const hasBadge = hasError || hasWarning;
  const PauseIcon = React.useMemo(() => resolveIcon("pause"), []);
  const ResumeIcon = React.useMemo(() => resolveIcon("step-forward"), []);
  const DuplicateIcon = React.useMemo(() => resolveIcon("copy"), []);
  const DeleteIcon = React.useMemo(() => resolveIcon("trash-2"), []);
  const ToggleViewIcon = React.useMemo(
    () => resolveIcon(isCompactView ? "chevrons-up-down" : "chevrons-down-up"),
    [isCompactView],
  );
  const resolvedEventStateMap = eventStateMap ?? DEFAULT_EVENT_STATE_MAP;
  const compactEventState = eventSections?.[0]?.eventState || "neutral";
  const compactStatusBadgeColor =
    eventSections && eventSections.length > 0
      ? (resolvedEventStateMap[compactEventState] || resolvedEventStateMap.neutral).badgeColor
      : undefined;

  return (
    <SelectionWrapper selected={selected}>
      <div
        className={`group relative flex flex-col outline-1 outline-slate-950/20 rounded-md w-[23rem] bg-white ${hasError ? "!outline-orange-500" : ""}`}
        data-view-mode={isCompactView ? "compact" : "expanded"}
      >
        <div className="absolute -top-8 right-0 z-10 h-8 w-44 opacity-0" />
        {showHeader ? (
          <div className="absolute -top-8 right-0 z-10 hidden items-center gap-2 group-hover:flex nodrag">
            {onTogglePause && !hasError && (
              <button
                type="button"
                data-testid="node-action-pause"
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  onTogglePause();
                }}
                className="flex items-center gap-1 px-1 py-0.5 text-[13px] font-medium text-gray-500 transition hover:text-gray-800"
              >
                {paused ? <ResumeIcon className="h-4 w-4" /> : <PauseIcon className="h-4 w-4" />}
                <span>{paused ? "Resume" : "Pause"}</span>
              </button>
            )}
            {onDuplicate && (
              <button
                type="button"
                data-testid="node-action-duplicate"
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  onDuplicate();
                }}
                className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
              >
                <DuplicateIcon className="h-4 w-4" />
              </button>
            )}
            {onToggleView && (
              <button
                type="button"
                data-testid="node-action-toggle-view"
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  onToggleView();
                }}
                className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
              >
                <ToggleViewIcon className="h-4 w-4" />
              </button>
            )}
            {onDelete && (
              <button
                type="button"
                data-testid="node-action-delete"
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  onDelete();
                }}
                className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
              >
                <DeleteIcon className="h-4 w-4" />
              </button>
            )}
          </div>
        ) : null}
        <ComponentHeader
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconColor={iconColor}
          title={title}
          onDoubleClick={onToggleCollapse}
          isCompactView={isCompactView}
          statusBadgeColor={compactStatusBadgeColor}
        />

        {hasBadge && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <div
                  data-testid="node-warning-badge"
                  className="absolute -top-6 left-1 bg-orange-500 rounded-t-md h-6 p-1 cursor-pointer"
                >
                  <AlertTriangle size={16} className="text-white" />
                </div>
              </TooltipTrigger>
              <TooltipContent>
                <p className="max-w-xs text-sm">{hasError ? error : warning}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}

        {paused && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <div
                  data-testid="node-paused-badge"
                  className={`absolute -top-6 ${hasBadge ? "left-8" : "left-1"} bg-blue-500 rounded-t-md h-6 p-1 cursor-pointer`}
                >
                  <PauseIcon className="h-4 w-4 text-white" />
                </div>
              </TooltipTrigger>
              <TooltipContent>
                <p className="max-w-xs text-sm">Queued items will not be consumed.</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}

        {isCompactView ? null : (
          <>
            {!hideMetadataList && metadata && metadata.length > 0 && <MetadataList items={metadata} />}

            {specs && specs.length > 0 && (
              <div className="px-2 py-1.5 border-b border-slate-950/20 text-gray-500 flex flex-col gap-1.5">
                {specs.map((spec, index) => (
                  <div key={index} className="flex items-center text-md text-gray-500">
                    <div className="w-4 h-4 mr-2">
                      {React.createElement(resolveIcon(spec.iconSlug || "list-filter"), { size: 16 })}
                    </div>
                    {spec.values ? (
                      <SpecsTooltip
                        specTitle={spec.tooltipTitle || spec.title}
                        specValues={spec.values}
                        hideCount={hideCount}
                      >
                        <span className="text-[13px] underline underline-offset-3 decoration-dotted decoration-1 decoration-gray-500 rounded-md font-inter font-medium cursor-help">
                          {hideCount ? "" : spec.values.length}{" "}
                          {spec.title + (spec.values.length > 1 && !hideCount ? "s" : "")}
                        </span>
                      </SpecsTooltip>
                    ) : spec.value !== undefined ? (
                      <PayloadTooltip
                        title={spec.tooltipTitle || spec.title}
                        value={spec.value}
                        contentType={spec.contentType || "json"}
                      >
                        <span className="text-sm bg-gray-500 px-2 py-1 rounded-md text-white font-mono font-medium cursor-help">
                          {spec.title}
                        </span>
                      </PayloadTooltip>
                    ) : null}
                  </div>
                ))}
              </div>
            )}

            {customFieldPosition === "before" &&
              (typeof customField === "function" ? customField(runDisabled ? undefined : onRun) : customField || null)}

            {eventSections?.map((section, index) => (
              <EventSectionDisplay
                className={
                  "pb-3" +
                  (!!includeEmptyState || (!!customField && customFieldPosition === "after")
                    ? " border-b border-slate-950/20"
                    : "")
                }
                key={index}
                section={section}
                index={index}
                totalSections={eventSections.length}
                stateMap={eventStateMap}
                lastSection={
                  index === eventSections.length - 1 &&
                  !includeEmptyState &&
                  !(customField && customFieldPosition === "after")
                }
              />
            ))}

            {includeEmptyState && <EmptyState {...emptyStateProps} />}

            {customFieldPosition === "after" &&
              (typeof customField === "function" ? customField(runDisabled ? undefined : onRun) : customField || null)}
          </>
        )}
      </div>
    </SelectionWrapper>
  );
};
