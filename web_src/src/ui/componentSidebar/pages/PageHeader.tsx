import React from "react";
import { ArrowLeft } from "lucide-react";
import { RUNS_SIDEBAR_ROW_CLASS } from "@/components/CanvasToolSidebar/runsSidebarRowLayout";
import { cn } from "@/lib/utils";

interface PageHeaderProps {
  onBackToOverview: () => void;
  compact?: boolean;
}

export const PageHeader: React.FC<PageHeaderProps> = ({ onBackToOverview, compact = false }) => {
  const backButtonText = "Back";

  if (compact) {
    return (
      <button
        type="button"
        data-testid="compact-page-header-back"
        onClick={onBackToOverview}
        className={cn(
          RUNS_SIDEBAR_ROW_CLASS,
          "w-full shrink-0 text-xs font-medium text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-800 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100",
        )}
      >
        <ArrowLeft className="h-3.5 w-3.5 shrink-0" />
        {backButtonText}
      </button>
    );
  }

  return (
    <div className="px-3 py-2 border-b-1 border-border">
      <button
        onClick={onBackToOverview}
        className="flex items-center gap-2 text-sm text-gray-500 hover:text-gray-800 font-medium cursor-pointer"
      >
        <ArrowLeft size={16} />
        {backButtonText}
      </button>
    </div>
  );
};
