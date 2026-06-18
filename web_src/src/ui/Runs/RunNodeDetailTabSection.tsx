import JsonView from "@uiw/react-json-view";
import React from "react";
import { cn, resolveIcon } from "@/lib/utils";
import { RunNodeDetailDetailsView } from "./RunNodeDetailDetailsView";
import { RUN_NODE_ICON_SIZE } from "./RunNodeIcon";
import type { RunNodeDetailTabData, RunNodeDetailTabKey } from "./runNodeDetailModel";

export interface RunNodeDetailTabSectionProps {
  activeTab: RunNodeDetailTabKey;
  tabData: RunNodeDetailTabData | null;
  hasDetailsSection: boolean;
  hasPayload: boolean;
  hasConfig: boolean;
  headerEventBadge: { badgeColor: string; label: string } | null;
  createdAt?: string;
  onSelectTab: (tab: RunNodeDetailTabKey) => void;
}

export function RunNodeDetailTabSection({
  activeTab,
  tabData,
  hasDetailsSection,
  hasPayload,
  hasConfig,
  headerEventBadge,
  createdAt,
  onSelectTab,
}: RunNodeDetailTabSectionProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
      <div className="relative z-10 flex h-9 shrink-0 items-stretch overflow-visible border-b border-slate-200 px-2">
        {hasDetailsSection ? (
          <TabButton
            active={activeTab === "details"}
            icon="info"
            label="Details"
            onClick={() => onSelectTab("details")}
          />
        ) : null}
        {hasPayload ? (
          <TabButton
            active={activeTab === "payload"}
            icon="code"
            label="Payload"
            onClick={() => onSelectTab("payload")}
          />
        ) : null}
        {hasConfig ? (
          <TabButton
            active={activeTab === "configuration"}
            icon="settings"
            label="Config"
            onClick={() => onSelectTab("configuration")}
          />
        ) : null}
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3">
        {activeTab === "details" && hasDetailsSection ? (
          <RunNodeDetailDetailsView
            details={tabData?.details ?? {}}
            statusBadge={headerEventBadge}
            relativeTime={createdAt}
          />
        ) : null}
        {activeTab === "payload" && hasPayload ? (
          <JsonView value={tabData?.payload as object} collapsed={2} style={{ fontSize: 12 }} />
        ) : null}
        {activeTab === "configuration" && hasConfig ? (
          <JsonView value={tabData?.configuration as object} collapsed={2} style={{ fontSize: 12 }} />
        ) : null}
      </div>
    </div>
  );
}

function TabButton({
  active,
  icon,
  label,
  onClick,
}: {
  active: boolean;
  icon: string;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "mb-[-1px] flex items-center gap-1 self-stretch border-b px-2.5 text-[13px] font-medium",
        active ? "border-gray-800 text-gray-800" : "border-transparent text-gray-500 hover:text-gray-800",
      )}
    >
      {React.createElement(resolveIcon(icon), { size: RUN_NODE_ICON_SIZE, className: "h-3.5 w-3.5 shrink-0" })}
      {label}
    </button>
  );
}
