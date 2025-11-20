import { Maximize, Minimize } from "lucide-react";

namespace ViewToggle {
  export interface Props {
    isCollapsed: boolean;
    onToggle: () => void;
    className?: string;
  }
}

function ViewToggle({ isCollapsed, onToggle, className = "" }: ViewToggle.Props) {
  return (
    <div className={`flex bg-white border border-border rounded-sm ${className}`}>
      <div className="border-r border-border">
        <button
          onClick={onToggle}
          className={`flex items-center justify-center w-6 h-6 transition-colors ${
            isCollapsed ? "" : "opacity-30"
          }`}
          title="Collapsed view"
        >
          <Minimize size={16} strokeWidth={2} />
        </button>
      </div>
      <div>
        <button
          onClick={onToggle}
          className={`flex items-center justify-center w-6 h-6 transition-colors ${!isCollapsed ? "" : "opacity-30"}`}
          title="Expanded view"
        >
          <Maximize size={16} />
        </button>
      </div>
    </div>
  );
}

export { ViewToggle };
