import React from "react";
import { Rabbit } from "lucide-react";

interface EmptyStateProps {
  icon?: React.ComponentType<{ size?: number }>;
  title?: string;
  description?: string;
  className?: string;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  icon: Icon = Rabbit,
  title = "Waiting for the first run...",
  description,
  className = "",
}) => {
  return (
    <div className={`flex flex-col justify-center items-center py-5 gap-3 ${className}`}>
      <div className="flex justify-center items-center w-12 h-12 text-yellow-700 dark:text-yellow-400 bg-orange-100 dark:bg-orange-900/50 p-3 rounded-md">
        <Icon size={16} />
      </div>
      <span className="text-sm text-gray-500 dark:text-gray-400">{title}</span>
      {description && <span className="text-sm text-gray-400 dark:text-gray-500 text-center max-w-xs">{description}</span>}
    </div>
  );
};
