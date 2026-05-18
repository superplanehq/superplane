import { X } from "lucide-react";
import { cn } from "@/lib/utils";

export function ToolTabsHeader({
  tabs,
  activeTab,
  onSelectTab,
  onClose,
}: {
  tabs: ReadonlyArray<{ value: string; label: string }>;
  activeTab: string;
  onSelectTab: (value: string) => void;
  onClose: () => void;
}) {
  return (
    <div className="flex h-10 min-h-10 shrink-0 items-center border-b border-slate-950/15 px-4">
      <div className="flex min-w-0 flex-1 flex-row items-stretch" role="tablist" aria-label="Canvas tools">
        {tabs.map(({ value, label }) => (
          <button
            key={value}
            type="button"
            role="tab"
            aria-selected={activeTab === value}
            onClick={() => onSelectTab(value)}
            className={cn(
              "mr-4 mb-[-1px] flex items-center border-b text-[13px] font-medium transition-colors",
              activeTab === value
                ? "border-gray-700 text-gray-800 dark:border-blue-600 dark:text-blue-400"
                : "border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300",
            )}
          >
            {label}
          </button>
        ))}
      </div>
      <button
        type="button"
        onClick={onClose}
        aria-label="Close sidebar"
        className="ml-2 inline-flex size-6 shrink-0 items-center justify-center rounded-md text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-700"
      >
        <X className="size-4" />
      </button>
    </div>
  );
}
