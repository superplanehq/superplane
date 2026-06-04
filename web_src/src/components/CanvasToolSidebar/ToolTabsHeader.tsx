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
      className="flex h-10 min-h-10 shrink-0 flex-row items-stretch border-b border-slate-950/15 px-4"
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
              ? "border-gray-700 text-gray-800 dark:border-blue-600 dark:text-blue-400"
              : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300",
          )}
        >
          {Icon ? (
            <Icon
              className={cn(
                "size-4 shrink-0",
                selectedTab === value ? "text-gray-800 dark:text-blue-400" : "text-gray-400 dark:text-gray-400",
              )}
              aria-hidden
            />
          ) : null}
          {label}
        </button>
      ))}
    </div>
  );
}
