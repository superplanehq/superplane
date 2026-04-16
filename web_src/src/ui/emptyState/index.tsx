import React from "react";
import { Rabbit } from "lucide-react";

interface EmptyStateProps {
  icon?: React.ComponentType<{ size?: number }>;
  title?: string;
  description?: string;
  className?: string;
  compact?: boolean;
  tone?: "accent" | "neutral";
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  icon: Icon = Rabbit,
  title = "Waiting for the first run...",
  description,
  className = "",
  compact = false,
  tone = "accent",
}) => {
  const iconContainerClassName = tone === "neutral" ? "text-slate-500 bg-slate-100" : "text-yellow-700 bg-orange-100";
  const titleClassName = tone === "neutral" ? "text-slate-500" : "text-gray-500";
  const descriptionClassName = tone === "neutral" ? "text-slate-400" : "text-gray-400";

  if (compact) {
    return (
      <div className={`flex items-center gap-2.5 px-2 py-3 ${className}`}>
        <div className={`flex justify-center items-center w-8 h-8 p-1.5 rounded-md shrink-0 ${iconContainerClassName}`}>
          <Icon size={14} />
        </div>
        <div className="flex flex-col min-w-0">
          <span className={`text-sm ${titleClassName}`}>{title}</span>
          {description && <span className={`text-xs truncate ${descriptionClassName}`}>{description}</span>}
        </div>
      </div>
    );
  }

  return (
    <div className={`flex flex-col justify-center items-center py-5 gap-3 ${className}`}>
      <div className={`flex justify-center items-center w-12 h-12 p-3 rounded-md ${iconContainerClassName}`}>
        <Icon size={16} />
      </div>
      <span className={`text-sm ${titleClassName}`}>{title}</span>
      {description && <span className={`text-sm text-center max-w-xs ${descriptionClassName}`}>{description}</span>}
    </div>
  );
};
