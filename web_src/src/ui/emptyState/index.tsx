import React from "react";
import { Rabbit } from "lucide-react";

interface EmptyStateProps {
  icon?: React.ComponentType<{ size?: number }>;
  title?: string;
  description?: string;
  className?: string;
  compact?: boolean;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  icon: Icon = Rabbit,
  title = "Waiting for the first run...",
  description,
  className = "",
  compact = false,
}) => {
  if (compact) {
    return (
      <div className={`flex items-center gap-2.5 px-2 py-3 ${className}`}>
        <div className="flex justify-center items-center w-8 h-8 text-yellow-700 bg-orange-100 p-1.5 rounded-md shrink-0">
          <Icon size={14} />
        </div>
        <div className="flex flex-col min-w-0">
          <span className="text-sm text-gray-500">{title}</span>
          {description && <span className="text-xs text-gray-400 truncate">{description}</span>}
        </div>
      </div>
    );
  }

  return (
    <div className={`flex flex-col justify-center items-center py-5 gap-3 ${className}`}>
      <div className="flex justify-center items-center w-12 h-12 text-yellow-700 bg-orange-100 p-3 rounded-md">
        <Icon size={16} />
      </div>
      <span className="text-sm text-gray-500">{title}</span>
      {description && <span className="text-sm text-gray-400 text-center max-w-xs">{description}</span>}
    </div>
  );
};
