import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import { CollapsedComponent } from "../collapsedComponent";
import { ComponentHeader } from "../componentHeader";
import { ListFilter } from "lucide-react";
import { SpecsTooltip } from "./SpecsTooltip";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { MetadataItem, MetadataList } from "../metadataList";

export interface SpecBadge {
  label: string;
  bgColor: string;
  textColor: string;
}

export interface ComponentBaseSpecValue {
  badges: SpecBadge[];
}

export interface EventSection {
  title: string;
  receivedAt?: Date;
  eventState?: "success" | "failed" | "neutral" | "running";
  eventTitle?: string;
  eventSubtitle?: string;
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
  spec?: {
    title: string;
    tooltipTitle?: string;
    values: ComponentBaseSpecValue[];
  }
  hideCount?: boolean;
  collapsed?: boolean;
  collapsedBackground?: string;
  eventSections?: EventSection[];
  selected?: boolean;
  metadata?: MetadataItem[];
}

export const ComponentBase: React.FC<ComponentBaseProps> = ({ iconSrc, iconSlug, iconColor, iconBackground, headerColor, title, description, spec, collapsed = false, collapsedBackground, eventSections, selected = false, onRun, runDisabled, runDisabledTooltip, onEdit, onConfigure, onDuplicate, onDeactivate, onToggleView, onDelete, isCompactView, hideCount, metadata }) => {
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
            {spec?.title && spec?.values?.length > 0 && <div className="flex items-center gap-1 text-xs text-gray-500">
              <ListFilter size={12} />
              <span>{!hideCount ? spec.values.length : ''} {spec.title + (spec.values.length > 1 && !hideCount ? "s" : "")}</span>
            </div>}
          </div>
        </CollapsedComponent>
      </SelectionWrapper>
    )
  }

  return (
    <SelectionWrapper selected={selected}>
      <div className="flex flex-col border-2 border-border rounded-md w-[23rem] bg-white" >
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
          onConfigure={onConfigure}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        />

        {spec && spec.title && spec.values?.length > 0 &&
          <div className="px-2 py-2 border-b text-gray-500 flex flex-col gap-2">
            {spec?.title && spec?.values?.length > 0 && <div className="flex items-center gap-3 text-md text-gray-500">
              <ListFilter size={18} />
              <SpecsTooltip specTitle={spec.tooltipTitle || spec.title} specValues={spec.values} hideCount={hideCount}>
                <span className="text-sm bg-gray-500 px-2 py-1 rounded-md text-white font-mono font-medium">{hideCount ? '' : spec.values.length} {spec.title + (spec.values.length > 1 && !hideCount ? "s" : "")}</span>
              </SpecsTooltip>
            </div>}
          </div>}

        {metadata && metadata.length > 0 && <MetadataList items={metadata} />}

        {eventSections?.map((section, index) => {
          const now = new Date()
          const diff = section.receivedAt ? now.getTime() - section.receivedAt.getTime() : 0
          const timeAgo = section.receivedAt ? calcRelativeTimeFromDiff(diff) : ""

          const LastEventIcon = section.eventState === "success"
            ? resolveIcon("check")
            : section.eventState === "neutral"
              ? resolveIcon("circle")
              : section.eventState === "running"
                ? resolveIcon("refresh-cw")
                : resolveIcon("x")
          const LastEventColor = section.eventState === "success"
            ? "text-green-700"
            : section.eventState === "neutral"
              ? "text-gray-500"
              : section.eventState === "running"
                ? "text-blue-800"
                : "text-red-700"
          const LastEventBackground = section.eventState === "success"
            ? "bg-green-200"
            : section.eventState === "neutral"
              ? "bg-gray-100"
              : section.eventState === "running"
                ? "bg-sky-100"
                : "bg-red-200"
          const LastEventIconColor = section.eventState === "success"
            ? "text-green-600 bg-green-600"
            : section.eventState === "neutral"
              ? "text-gray-400 bg-gray-400"
              : section.eventState === "running"
                ? "text-blue-800"
                : "text-red-600 bg-red-600"

          return (
            <div key={index} className={"px-4 pt-2 pb-6 relative" + (index < eventSections.length - 1 ? " border-b" : "")}>
              <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
                <span className="uppercase text-sm font-medium">{section.title}</span>
                <span className="text-sm">{timeAgo}</span>
              </div>
              <div className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md ${LastEventBackground} ${LastEventColor}`}>
                <div className="flex items-center gap-2 min-w-0 flex-1">
                  <div className={`w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center ${LastEventIconColor}`}>
                    <LastEventIcon size={section.eventState === "running" ? 16 : 12} className={section.eventState === "running" ? "animate-spin" : "text-white"} />
                  </div>
                  <span className="truncate text-sm min-w-0">{section.eventTitle}</span>
                </div>
                {section.eventSubtitle && (
                  <span className="text-sm truncate flex-shrink-0 max-w-[40%] text-gray-500">{section.eventSubtitle}</span>
                )}
              </div>
              {section.handleComponent}
            </div>
          )
        })}
      </div>
    </SelectionWrapper>
  )
}
