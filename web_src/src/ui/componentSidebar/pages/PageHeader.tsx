import React from "react";
import { ArrowLeft, Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ChildEventsState } from "../../composite";

const DEFAULT_STATUS_OPTIONS: { value: ChildEventsState; label: string }[] = [
  { value: "processed", label: "Processed" },
  { value: "discarded", label: "Failed" },
  { value: "running", label: "Running" },
];

interface PageHeaderProps {
  page: "history" | "queue" | "execution-chain";
  onBackToOverview: () => void;

  // Search and filter props (only for history/queue)
  showSearchAndFilter?: boolean;
  searchQuery?: string;
  onSearchChange?: (value: string) => void;
  statusFilter?: string;
  onStatusFilterChange?: (value: string) => void;
  extraStatusOptions?: string[];
}

export const PageHeader: React.FC<PageHeaderProps> = ({
  page,
  onBackToOverview,
  showSearchAndFilter = false,
  searchQuery = "",
  onSearchChange,
  statusFilter = "all",
  onStatusFilterChange,
  extraStatusOptions = [],
}) => {
  const getPageTitle = () => {
    switch (page) {
      case "history":
        return "Full History";
      case "queue":
        return "Queue";
      case "execution-chain":
        return "Execution Chain";
      default:
        return "";
    }
  };

  return (
    <>
      {/* Back to Overview Section */}
      <div className="px-3 py-2 border-b-1 border-border">
        <button
          onClick={onBackToOverview}
          className="flex items-center gap-2 text-sm text-gray-500 hover:text-gray-800 cursor-pointer"
        >
          <ArrowLeft size={16} />
          Back to Overview
        </button>
      </div>

      {/* Page Header with Search and Filter */}
      <div className="px-3 py-3">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-xs font-semibold uppercase text-gray-500">
            {getPageTitle()}
          </h2>
        </div>
        {showSearchAndFilter && (
          <div className="flex gap-2">
            {/* Search Input */}
            <div className="relative flex-1">
              <Search size={16} className="absolute left-2 top-1/2 transform -translate-y-1/2 text-gray-500" />
              <Input
                type="text"
                placeholder="Search events..."
                value={searchQuery}
                onChange={(e) => onSearchChange?.(e.target.value)}
                className="pl-8 h-9 text-sm"
              />
            </div>
            {/* Status Filter */}
            <Select
              value={statusFilter}
              onValueChange={(value) => onStatusFilterChange?.(value)}
            >
              <SelectTrigger className="w-[160px] h-9 text-sm text-gray-500">
                <SelectValue placeholder="All Statuses" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all" className="text-gray-500">
                  All Statuses
                </SelectItem>
                {DEFAULT_STATUS_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value} className="text-gray-500">
                    {option.label}
                  </SelectItem>
                ))}
                {extraStatusOptions.map((status) => (
                  <SelectItem key={status} value={status} className="text-gray-500">
                    {status.charAt(0).toUpperCase() + status.slice(1)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
      </div>
    </>
  );
};