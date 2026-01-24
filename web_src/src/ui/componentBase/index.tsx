import React from "react";
import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import { ComponentHeader } from "../componentHeader";
import { SpecsTooltip } from "./SpecsTooltip";
import { PayloadTooltip } from "./PayloadTooltip";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { MetadataItem, MetadataList } from "../metadataList";
import { EmptyState } from "../emptyState";
import { ChildEvents, type ChildEventsInfo } from "../childEvents";
import { AlertTriangle } from "lucide-react";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";

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
        (index < totalSections - 1 ? " border-b border-slate-950/20 dark:border-gray-600" : "") +
        ` ${className}`
      }
    >
      <div className="flex items-center justify-between gap-2 min-w-0 flex-1">
        <div
          className={`uppercase text-[11px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white ${LastEventStateColor}`}
        >
          <span>{stateStyle.label || currentState}</span>
        </div>
        {section.eventSubtitle && (
          <span
            title={String(section.eventSubtitle)}
            className="text-[13px] font-medium truncate flex-shrink-0 max-w-[65%] text-gray-950/50 dark:text-gray-400"
          >
            {section.showAutomaticTime && durationText ? durationText : section.eventSubtitle}
          </span>
        )}
      </div>
      <div className="flex justify-left items-center mt-1 gap-2">
        {section.eventId && (
          <span className="text-[13px] text-gray-950/50 dark:text-gray-400 font-mono">#{section.eventId?.slice(0, 4)}</span>
        )}
        <span className="text-sm text-gray-700 dark:text-gray-300 font-inter truncate text-md min-w-0 font-medium truncate">
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
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-violet-100 dark:bg-violet-900/50",
    badgeColor: "bg-violet-400",
  },
  success: {
    icon: "circle-check",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-green-100 dark:bg-green-900/50",
    badgeColor: "bg-emerald-500",
  },
  failed: {
    icon: "circle-x",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-red-100 dark:bg-red-900/50",
    badgeColor: "bg-red-400",
  },
  cancelled: {
    icon: "circle-slash-2",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-gray-100 dark:bg-gray-700",
    badgeColor: "bg-gray-500",
  },
  error: {
    icon: "triangle-alert",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-red-100 dark:bg-red-900/50",
    badgeColor: "bg-red-500",
  },
  neutral: {
    icon: "circle",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-gray-50 dark:bg-gray-700",
    badgeColor: "bg-gray-400",
  },
  queued: {
    icon: "circle-dashed",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-orange-100 dark:bg-orange-900/50",
    badgeColor: "bg-yellow-600",
  },
  running: {
    icon: "refresh-cw",
    textColor: "text-gray-800 dark:text-gray-200",
    backgroundColor: "bg-sky-100 dark:bg-sky-900/50",
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
  appName?: string;
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
  includeEmptyState?: boolean;
  emptyStateProps?: {
    icon?: React.ComponentType<{ size?: number }>;
    title?: string;
    description?: string;
  };
  error?: string;
}

export const ComponentBase: React.FC<ComponentBaseProps> = ({
  iconSrc,
  iconSlug,
  iconColor,
  appName,
  title,
  specs,
  collapsed: _collapsed = false,
  collapsedBackground: _collapsedBackground,
  eventSections,
  selected = false,
  onToggleCollapse,
  onRun,
  runDisabled,
  runDisabledTooltip: _runDisabledTooltip,
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
  eventStateMap,
  includeEmptyState = false,
  emptyStateProps,
  error,
}) => {
  const hasError = error && error.trim() !== "";
  const RunIcon = React.useMemo(() => resolveIcon("play"), []);
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
        className={`group relative flex flex-col outline-1 outline-slate-950/20 dark:outline-gray-600 rounded-md w-[23rem] bg-white dark:bg-gray-800 ${hasError ? "!outline-orange-500" : ""}`}
        data-view-mode={isCompactView ? "compact" : "expanded"}
      >
        <div className="absolute -top-8 right-0 z-10 h-8 w-44 opacity-0" />
        <div className="absolute -top-8 right-0 z-10 hidden items-center gap-2 group-hover:flex nodrag">
          {onRun && (
            <button
              type="button"
              data-testid="node-action-run"
              onClick={(event) => {
                event.preventDefault();
                event.stopPropagation();
                onRun();
              }}
              disabled={runDisabled}
              className="flex items-center gap-1 px-1 py-0.5 text-[13px] font-medium text-gray-500 dark:text-gray-400 transition hover:text-gray-800 dark:hover:text-gray-200 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <RunIcon className="h-4 w-4" />
              <span>Run</span>
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
              className="flex items-center justify-center p-1 text-gray-500 dark:text-gray-400 transition hover:text-gray-800 dark:hover:text-gray-200"
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
              className="flex items-center justify-center p-1 text-gray-500 dark:text-gray-400 transition hover:text-gray-800 dark:hover:text-gray-200"
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
              className="flex items-center justify-center p-1 text-gray-500 dark:text-gray-400 transition hover:text-gray-800 dark:hover:text-gray-200"
            >
              <DeleteIcon className="h-4 w-4" />
            </button>
          )}
        </div>
        <ComponentHeader
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconColor={iconColor}
          appName={appName}
          title={title}
          onDoubleClick={onToggleCollapse}
          isCompactView={isCompactView}
          statusBadgeColor={compactStatusBadgeColor}
        />

        {hasError && (
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="absolute -top-6 left-1 bg-orange-500 rounded-t-md h-6 p-1 cursor-pointer">
                  <AlertTriangle size={16} className="text-white" />
                </div>
              </TooltipTrigger>
              <TooltipContent>
                <p className="max-w-xs text-sm">{error}</p>
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        )}

        {isCompactView ? null : (
          <>
            {!hideMetadataList && metadata && metadata.length > 0 && <MetadataList items={metadata} />}

            {specs && specs.length > 0 && (
              <div className="px-2 py-1.5 border-b border-slate-950/20 dark:border-gray-600 text-gray-500 dark:text-gray-400 flex flex-col gap-1.5">
                {specs.map((spec, index) => (
                  <div key={index} className="flex items-center text-md text-gray-500 dark:text-gray-400">
                    <div className="w-4 h-4 mr-2">
                      {React.createElement(resolveIcon(spec.iconSlug || "list-filter"), { size: 16 })}
                    </div>
                    {spec.values ? (
                      <SpecsTooltip
                        specTitle={spec.tooltipTitle || spec.title}
                        specValues={spec.values}
                        hideCount={hideCount}
                      >
                        <span className="text-[13px] underline underline-offset-3 decoration-dotted decoration-1 decoration-gray-500 dark:decoration-gray-400 rounded-md font-inter font-medium cursor-help">
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

            {eventSections?.map((section, index) => (
              <EventSectionDisplay
                className={"pb-3" + (!!includeEmptyState || !!customField ? " border-b border-slate-950/20 dark:border-gray-600" : "")}
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
          </>
        )}
      </div>
    </SelectionWrapper>
  );
};
