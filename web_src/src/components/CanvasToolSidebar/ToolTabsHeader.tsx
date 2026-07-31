import type { ComponentType, SVGProps } from "react";
import { cn } from "@/lib/utils";

type TabIcon = ComponentType<SVGProps<SVGSVGElement>>;

export function ToolTabsHeader({
  tabs,
  activeTab,
  onSelectTab,
}: {
  tabs: ReadonlyArray<{ value: string; label: string; icon?: TabIcon }>;
  activeTab: string;
  onSelectTab: (value: string) => void;
}) {
  const selectedTab = tabs.some(({ value }) => value === activeTab) ? activeTab : tabs[0]?.value;

  return (
    <div
      className="flex h-10 min-h-10 shrink-0 flex-row items-stretch border-b border-edge-default px-4"
      role="tablist"
      aria-label="Canvas tools"
    >
      {tabs.map(({ value, label, icon: Icon }) => (
        <button
          key={value}
          type="button"
          role="tab"
          aria-selected={selectedTab === value}
          onClick={() => onSelectTab(value)}
          className={cn(
            "mr-4 mb-[-1px] flex items-center gap-1.5 border-b text-[13px] font-medium transition-colors",
            selectedTab === value
              ? "border-action-primary text-content-primary"
              : "border-transparent text-content-secondary hover:text-content-primary",
          )}
        >
          {Icon ? (
            <Icon
              className={cn("size-4 shrink-0", selectedTab === value ? "text-action-primary" : "text-content-muted")}
              aria-hidden
            />
          ) : null}
          {label}
        </button>
      ))}
    </div>
  );
}
