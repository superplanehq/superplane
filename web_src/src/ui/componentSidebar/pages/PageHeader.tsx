import React from "react";
import { ArrowLeft } from "lucide-react";

interface PageHeaderProps {
  page: "history" | "queue" | "execution-chain";
  onBackToOverview: () => void;
  previousPage?: "overview" | "history" | "queue";
}

export const PageHeader: React.FC<PageHeaderProps> = ({ page, onBackToOverview, previousPage = "overview" }) => {
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
