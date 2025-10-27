import { calcRelativeTimeFromDiff, resolveIcon } from "@/lib/utils";
import { CollapsedComponent } from "../collapsedComponent";
import { ComponentHeader } from "../componentHeader";
import { ListFilter } from "lucide-react";
import { SpecsTooltip } from "./SpecsTooltip";

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
  eventState?: "success" | "failed";
  eventTitle?: string;
  handleComponent?: React.ReactNode;
}

export interface ComponentBaseProps {
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
  collapsed?: boolean;
  collapsedBackground?: string;
  eventSections?: EventSection[];
}

export const ComponentBase: React.FC<ComponentBaseProps> = ({ iconSrc, iconSlug, iconColor, iconBackground, headerColor, title, description, spec, collapsed = false, collapsedBackground, eventSections }) => {
  if (collapsed) {
    return (
      <CollapsedComponent
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        iconColor={iconColor}
        iconBackground={iconBackground}
        title={title}
        collapsedBackground={collapsedBackground}
        shape="circle"
      >
        <div className="flex flex-col items-center gap-1">
          {spec?.title && spec?.values?.length > 0 && <div className="flex items-center gap-1 text-xs text-gray-500">
            <ListFilter size={12} />
            <span>{spec.values.length} {spec.title + (spec.values.length > 1 ? "s" : "")}</span>
          </div>}
        </div>
      </CollapsedComponent>
    )
  }

  return (
    <div className="flex flex-col border-2 border-border rounded-md w-[23rem] bg-white" >
      <ComponentHeader
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        iconBackground={iconBackground}
        iconColor={iconColor}
        headerColor={headerColor}
        title={title}
        description={description}
      />

      {spec && spec.title && spec.values?.length > 0 &&
        <div className="px-2 py-2 border-b text-gray-500 flex flex-col gap-2">
          {spec?.title && spec?.values?.length > 0 && <div className="flex items-center gap-3 text-md text-gray-500">
            <ListFilter size={18} />
            <SpecsTooltip specTitle={spec.tooltipTitle || spec.title} specValues={spec.values}>
              <span className="text-sm bg-gray-500 px-2 py-1 rounded-md text-white font-mono font-medium cursor-help">{spec.values.length} {spec.title + (spec.values.length > 1 ? "s" : "")}</span>
            </SpecsTooltip>
          </div>}
        </div>}

      {eventSections?.map((section, index) => {
        const now = new Date()
        const diff = section.receivedAt ? now.getTime() - section.receivedAt.getTime() : 0
        const timeAgo = section.receivedAt ? calcRelativeTimeFromDiff(diff) : ""

        const LastEventIcon = section.eventState === "success" ? resolveIcon("check") : resolveIcon("x")
        const LastEventColor = section.eventState === "success" ? "text-green-700" : "text-red-700"
        const LastEventBackground = section.eventState === "success" ? "bg-green-200" : "bg-red-200"
        const LastEventIconColor = section.eventState === "success" ? "text-green-600 bg-green-600" : "text-red-600 bg-red-600"

        return (
          <div key={index} className={"px-4 pt-2 pb-6 relative" + (index < eventSections.length - 1 ? " border-b" : "")}>
            <div className="flex items-center justify-between gap-3 text-gray-500 mb-2">
              <span className="uppercase text-sm font-medium">{section.title}</span>
              <span className="text-sm">{timeAgo}</span>
            </div>
            <div className={`flex items-center justify-between gap-3 px-2 py-2 rounded-md ${LastEventBackground} ${LastEventColor}`}>
              <div className="flex items-center gap-2 min-w-0 flex-1">
                <div className={`w-5 h-5 flex-shrink-0 rounded-full flex items-center justify-center ${LastEventIconColor}`}>
                  <LastEventIcon size={12} className="text-white" />
                </div>
                <span className="truncate text-sm">{section.eventTitle}</span>
              </div>
            </div>
            {section.handleComponent}
          </div>
        )
      })}
    </div>
  )
}