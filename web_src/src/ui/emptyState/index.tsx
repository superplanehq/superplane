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
    <div className={`flex flex-col justify-center items-center py-10 gap-3 ${className}`}>
      <div className="flex justify-center align-center w-12 h-12 text-indigo-700 bg-blue-100 p-3 rounded-lg">
        <Icon size={27} />
      </div>
      <span className="text-lg text-gray-500">{title}</span>
      {description && <span className="text-sm text-gray-400 text-center max-w-xs">{description}</span>}
    </div>
  );
};
