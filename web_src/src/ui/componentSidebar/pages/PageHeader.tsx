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
  previousPage?: "overview" | "history" | "queue";

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
  previousPage = "overview",
  showSearchAndFilter = false,
  searchQuery = "",
  onSearchChange,
  statusFilter = "all",
  onStatusFilterChange,
  extraStatusOptions = [],
}) => {
  const getBackButtonText = () => {
    if (page === "execution-chain") {
      switch (previousPage) {
        case "history":
          return "Back to Run History";
        case "queue":
          return "Back to Queue";
        default:
          return "All Runs";
      }
    }
    return "Back";
  };

  return (
    <>
      {/* Back to Overview Section */}
      <div className="px-3 py-2 border-b-1 border-border">
        <button
          onClick={onBackToOverview}
          className="flex items-center gap-2 text-sm text-gray-500 hover:text-gray-800 font-medium cursor-pointer"
        >
          <ArrowLeft size={16} />
          {getBackButtonText()}
        </button>
      </div>

      {/* Page Header with Search and Filter */}
    </>
  );
};
