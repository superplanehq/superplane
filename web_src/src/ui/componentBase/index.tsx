import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { getDraftDiffOutlineClassName, type DraftDiffStatus } from "@/lib/draftDiff";
import { withEventSectionDarkBackground } from "@/lib/eventSectionBackground";
import { withEventStatusBadgeClasses } from "@/lib/eventStatusBadge";
import { eventSectionMetadataTextClassName } from "@/lib/nodeCanvasSections";
import { calcRelativeTimeFromDiff, cn, resolveIcon } from "@/lib/utils";
import { CircleAlert, Rabbit } from "lucide-react";
import React from "react";
import { ComponentHeader } from "../componentHeader";
import { EmptyState } from "../emptyState";
import type { MetadataItem } from "../metadataList";
import { MetadataList } from "../metadataList";
import { SelectionWrapper } from "../selectionWrapper";
import type { ComponentActionsProps } from "../types/componentActions";
import { PayloadTooltip } from "./PayloadTooltip";
import { SpecsTooltip } from "./SpecsTooltip";
import { Timestamp } from "@/components/Timestamp";

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

  const LastEventBackground = withEventSectionDarkBackground(stateStyle.backgroundColor);
  const LastEventStateColor = withEventStatusBadgeClasses(stateStyle.badgeColor);
  const durationText = liveDuration !== null ? calcRelativeTimeFromDiff(liveDuration) : "";

  return (
    <div
      key={index}
      className={
        `px-2 pt-2 relative ${lastSection ? "rounded-b-md" : ""} ${LastEventBackground}` +
        (index < totalSections - 1 ? " border-b border-slate-950/20 dark:border-gray-600/70" : "") +
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
            title={typeof section.eventSubtitle === "string" ? section.eventSubtitle : undefined}
            className={cn(
              "text-[13px] font-medium truncate flex-shrink-0 max-w-[65%]",
              eventSectionMetadataTextClassName,
            )}
          >
            {section.showAutomaticTime && durationText ? durationText : section.eventSubtitle}
          </span>
        ) : (
          <span
            className={cn(
              "text-[13px] font-medium truncate flex-shrink-0 max-w-[65%]",
              eventSectionMetadataTextClassName,
            )}
          >
            <Timestamp date={section.receivedAt} display="relative" relativeStyle="abbreviated" />
          </span>
        )}
      </div>
      <div className="flex justify-left items-center mt-1 gap-2">
        {section.eventId && (
          <span className={cn("text-[13px] font-mono", eventSectionMetadataTextClassName)}>
            #{section.eventId?.slice(0, 4)}
          </span>
        )}
        <span className="text-sm text-gray-700 font-inter truncate text-md min-w-0 font-medium truncate dark:text-white/70">
          {section.eventTitle}
        </span>
      </div>
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
  value?: unknown;
  // Content type for value tooltips (json, xml, or text)
  contentType?: "json" | "xml" | "text";
}

export type EmptyStatePurpose = "runtime" | "setup" | "fallback";

export type EventState = "success" | "failed" | "neutral" | "queued" | "running" | string;

export interface EventStateStyle {
  icon: string;
  textColor: string;
  backgroundColor: string;
  badgeColor: string;
  label?: string; // Optional display label, defaults to state name if not provided
}

export type EventStateMap = Record<EventState, EventStateStyle>;

// eslint-disable-next-line react-refresh/only-export-components
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
  cancelling: {
    icon: "refresh-cw",
    textColor: "text-gray-800",
    backgroundColor: "bg-amber-100",
    badgeColor: "bg-amber-500",
    label: "Cancelling",
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
}

export interface ComponentBaseProps extends ComponentActionsProps {
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  title: string;
  showHeader?: boolean;
  specs?: ComponentBaseSpec[];
  hideCount?: boolean;
  hideMetadataList?: boolean;
  collapsed?: boolean;
  collapsedBackground?: string;
  eventSections?: EventSection[];
  selected?: boolean;
  metadata?: MetadataItem[];
  /** Custom content rendered on the node */
  customField?: React.ReactNode | (() => React.ReactNode);
  /** Where to render customField: "before" (before events) or "after" (after events, default) */
  customFieldPosition?: "before" | "after";
  /** Whether the custom field should only be shown in live mode */
  customFieldVisibility?: "always" | "live-only";
  eventStateMap?: EventStateMap;
  includeEmptyState?: boolean;
  emptyStateProps?: {
    icon?: React.ComponentType<{ size?: number }>;
    title?: string;
    description?: string;
    purpose?: EmptyStatePurpose;
    tone?: "accent" | "neutral";
  };
  error?: string;
  warning?: string;
  canvasMode?: "live" | "edit";
  /**
   * When true, only the header (icon + title) is shown for expanded nodes; body is replaced with a neutral slate block.
   * Used for contextual dimming (e.g. runs view non-participant nodes).
   */
  dimBodyBelowHeader?: boolean;
  draftDiffStatus?: DraftDiffStatus;
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
  onDuplicate,
  onToggleView,
  onDelete,
  isCompactView,
  hideCount,
  hideMetadataList,
  metadata,
  customField,
  customFieldPosition = "after",
  customFieldVisibility = "always",
  eventStateMap,
  includeEmptyState = false,
  emptyStateProps,
  error,
  warning,
  canvasMode = "live",
  dimBodyBelowHeader = false,
  draftDiffStatus,
}) => {
  const safeMetadata = Array.isArray(metadata) ? metadata : undefined;
  const safeSpecs = Array.isArray(specs) ? specs : undefined;
  const safeEventSections = Array.isArray(eventSections) ? eventSections : undefined;
  const safeError = typeof error === "string" ? error : "";
  const safeWarning = typeof warning === "string" ? warning : "";
  const safeCustomFieldPosition = customFieldPosition === "before" ? "before" : "after";
  const safeCustomFieldVisibility = customFieldVisibility === "live-only" ? "live-only" : "always";
  const safeCustomField = React.useMemo(() => {
    if (typeof customField === "function") {
      return () => {
        try {
          return customField() ?? null;
        } catch (renderError) {
          console.error("[ComponentBase] customField threw during render:", renderError);
          return null;
        }
      };
    }

    return customField ?? null;
  }, [customField]);
  const hasError = safeError.trim() !== "";
  const hasWarning = safeWarning.trim() !== "";
  const hasBadge = hasError || hasWarning;
  const emptyStatePurpose =
    emptyStateProps?.purpose || (includeEmptyState ? (hasError ? "setup" : "runtime") : undefined);
  const resolvedEmptyStateProps = React.useMemo(() => {
    if (canvasMode !== "edit" || emptyStatePurpose !== "runtime") {
      return emptyStateProps;
    }

    return {
      ...emptyStateProps,
      icon: Rabbit,
      title: "Ready to run...",
      description: undefined,
      purpose: "runtime" as const,
      tone: "neutral" as const,
    };
  }, [canvasMode, emptyStateProps, emptyStatePurpose]);
  const DuplicateIcon = React.useMemo(() => resolveIcon("copy"), []);
  const DeleteIcon = React.useMemo(() => resolveIcon("trash-2"), []);
  const ToggleViewIcon = React.useMemo(
    () => resolveIcon(isCompactView ? "chevrons-up-down" : "chevrons-down-up"),
    [isCompactView],
  );
  const resolvedEventStateMap = eventStateMap ?? DEFAULT_EVENT_STATE_MAP;
  const compactEventState = safeEventSections?.[0]?.eventState || "neutral";
  const compactStatusBadgeColor =
    safeEventSections && safeEventSections.length > 0
      ? (resolvedEventStateMap[compactEventState] || resolvedEventStateMap.neutral).badgeColor
      : undefined;
  const renderedCustomField =
    safeCustomFieldVisibility === "live-only" && canvasMode === "edit"
      ? null
      : typeof safeCustomField === "function"
        ? safeCustomField()
        : safeCustomField || null;
  const shouldClipCustomFieldToBottom =
    !!renderedCustomField &&
    (safeCustomFieldPosition === "after" || (!safeEventSections?.length && !includeEmptyState));
  const customFieldNode = renderedCustomField ? (
    shouldClipCustomFieldToBottom ? (
      <div className="overflow-hidden rounded-b-md">{renderedCustomField}</div>
    ) : (
      renderedCustomField
    )
  ) : null;
  return (
    <SelectionWrapper selected={selected}>
      <div
        className={cn(
          "group relative flex flex-col rounded-md w-[23rem]",
          getDraftDiffOutlineClassName(draftDiffStatus),
          !draftDiffStatus && hasError && "!outline-orange-500 dark:!outline-orange-400/50",
          dimBodyBelowHeader ? "bg-slate-200 dark:bg-gray-800" : "bg-white dark:bg-gray-800",
        )}
        data-view-mode={isCompactView ? "compact" : "expanded"}
      >
        <div className="absolute -top-8 right-0 z-10 h-8 w-44 opacity-0" />
        {showHeader ? (
          <div className="absolute -top-8 right-0 z-10 hidden items-center gap-2 group-hover:flex nodrag">
            {onDuplicate && (
              <button
                type="button"
                data-testid="node-action-duplicate"
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  onDuplicate();
                }}
                className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
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
                className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
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
                className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
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
          isCompactView={isCompactView}
          statusBadgeColor={compactStatusBadgeColor}
          mergeWithMutedBodyBelow={dimBodyBelowHeader}
        />

        {dimBodyBelowHeader ? (
          !isCompactView ? (
            <div className="min-h-28 w-full shrink-0 bg-slate-200 rounded-b-md dark:bg-gray-800" aria-hidden />
          ) : null
        ) : (
          <>
            {hasBadge && (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <div
                      data-testid="node-warning-badge"
                      className="absolute -top-8 left-0 flex h-6 w-6 cursor-pointer items-center justify-center rounded-full bg-orange-500 dark:bg-orange-400/25 dark:ring-1 dark:ring-orange-400/50"
                    >
                      <CircleAlert className="h-4 w-4 text-white dark:text-orange-300" />
                    </div>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p className="max-w-xs text-sm">{hasError ? safeError : safeWarning}</p>
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            )}

            {isCompactView ? null : (
              <>
                {!hideMetadataList && safeMetadata && safeMetadata.length > 0 && <MetadataList items={safeMetadata} />}

                {safeSpecs && safeSpecs.length > 0 && (
                  <div className="px-2 py-1.5 border-b border-slate-950/20 dark:border-gray-600/70 text-gray-500 flex flex-col gap-1.5">
                    {safeSpecs.map((spec, index) => (
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
                            <span className="text-[13px] bg-gray-500 px-2 py-0.5 rounded-md text-white font-mono font-medium cursor-help">
                              {spec.title}
                            </span>
                          </PayloadTooltip>
                        ) : null}
                      </div>
                    ))}
                  </div>
                )}

                {safeCustomFieldPosition === "before" && customFieldNode}

                {safeEventSections?.map((section, index) => (
                  <EventSectionDisplay
                    className={
                      "pb-3" +
                      (!!includeEmptyState || (!!renderedCustomField && safeCustomFieldPosition === "after")
                        ? " border-b border-slate-950/20"
                        : "")
                    }
                    key={index}
                    section={section}
                    index={index}
                    totalSections={safeEventSections.length}
                    stateMap={eventStateMap}
                    lastSection={
                      index === safeEventSections.length - 1 &&
                      !includeEmptyState &&
                      !(renderedCustomField && safeCustomFieldPosition === "after")
                    }
                  />
                ))}

                {includeEmptyState && <EmptyState compact {...resolvedEmptyStateProps} />}

                {safeCustomFieldPosition === "after" && customFieldNode}
              </>
            )}
          </>
        )}
      </div>
    </SelectionWrapper>
  );
};
