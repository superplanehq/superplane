import { Square, LayoutGrid } from "lucide-react";

namespace ViewToggle {
  export interface Props {
    isCollapsed: boolean;
    onToggle: () => void;
    className?: string;
  }
}

function ViewToggle({ isCollapsed, onToggle, className = "" }: ViewToggle.Props) {
  return (
    <div className={`flex bg-white border-2 border-gray-200 rounded-md ${className}`}>
      <button
        onClick={onToggle}
        className={`flex items-center justify-center w-8 h-8 transition-colors border-r-2 border-gray-200 ${
          isCollapsed ? "" : "opacity-50"
        }`}
        title="Collapsed view"
      >
        <Square size={10} strokeWidth={4} />
      </button>
      <button
        onClick={onToggle}
        className={`flex items-center justify-center w-8 h-8 transition-colors ${!isCollapsed ? "" : "opacity-50"}`}
        title="Expanded view"
      >
        <LayoutGrid size={16} />
      </button>
    </div>
  );
}

export { ViewToggle };
