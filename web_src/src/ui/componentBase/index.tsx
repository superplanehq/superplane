import React from "react";
import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import { CollapsedComponent } from "../collapsedComponent";
import { ComponentHeader } from "../componentHeader";
import { SpecsTooltip } from "./SpecsTooltip";
import { JsonTooltip } from "./JsonTooltip";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { MetadataItem, MetadataList } from "../metadataList";

interface EventSectionDisplayProps {
  section: EventSection;
  index: number;
  totalSections: number;
}

const EventSectionDisplay: React.FC<EventSectionDisplayProps> = ({ section, index, totalSections }) => {
  // Live timer for running executions
  const [liveDuration, setLiveDuration] = React.useState<number | null>(null);

  React.useEffect(() => {
    if (section.inProgress && section.receivedAt) {
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
  }, [section.inProgress, section.receivedAt]);

  const now = new Date();
  const diff = section.receivedAt ? now.getTime() - section.receivedAt.getTime() : 0;
  const timeAgo = section.receivedAt ? calcRelativeTimeFromDiff(diff) : "";
  const durationText = liveDuration !== null ? calcRelativeTimeFromDiff(liveDuration) : "";

  const Icon = resolveIcon(section.iconSlug);

  // Determine what to show in the top-right corner
  let topRightText = "";
  if (section.subtitle) {
    topRightText = section.subtitle;
  } else if (section.showAutomaticTime) {
    topRightText = durationText && section.inProgress ? `Running for: ${durationText}` : timeAgo;
  } else {
    topRightText = timeAgo;
  }

  return (
    <div key={index} className={"px-4 pt-2 pb-6 relative" + (index < totalSections - 1 ? " border-b" : "")}>
      <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
        <span className="uppercase text-xs font-semibold tracking-wide">{section.title}</span>
        {topRightText && <span className="text-sm">{topRightText}</span>}
      </div>
      <div
        className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md ${section.backgroundColor} ${section.textColor}`}
      >
        <div className="flex items-center gap-2 min-w-0 flex-1">
          <div className={`w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center ${section.iconColor}`}>
            <Icon size={section.iconSize} className={section.iconClassName} />
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

export interface EventSection {
  title: string;
  subtitle?: string;
  showAutomaticTime?: boolean;
  receivedAt?: Date;
  eventTitle?: string;
  eventSubtitle?: string;
  handleComponent?: React.ReactNode;
  iconSlug: string;
  iconColor: string;
  iconSize: number;
  iconClassName: string;
  backgroundColor: string;
  textColor: string;
  inProgress: boolean;
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
  collapsed?: boolean;
  collapsedBackground?: string;
  eventSections?: EventSection[];
  selected?: boolean;
  metadata?: MetadataItem[];
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
  metadata,
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
        />

        {metadata && metadata.length > 0 && <MetadataList items={metadata} />}

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
          <EventSectionDisplay key={index} section={section} index={index} totalSections={eventSections.length} />
        ))}
      </div>
    </SelectionWrapper>
  );
};
