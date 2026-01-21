import { Link } from "react-router-dom";
import { Heading } from "../Heading/heading";
import { Text } from "../Text/text";
import { Avatar } from "../Avatar/avatar";

export interface CanvasCardData {
  id: string;
  name: string;
  description?: string;
  createdAt: string;
  createdBy: {
    name: string;
    avatar?: string;
    initials: string;
  };
  type: "canvas";
}

export interface CanvasCardProps {
  canvas: CanvasCardData;
  organizationId: string;
  variant?: "grid" | "list";
}

export const CanvasCard = ({ canvas, organizationId, variant = "grid" }: CanvasCardProps) => {
  if (variant === "grid") {
    return (
      <div className="max-h-45 bg-white dark:bg-gray-800rounded-md outline outline-slate-950/10 dark:border-gray-800 hover:shadow-md transition-shadow group">
        <div className="p-6 flex flex-col justify-between h-full">
          <div>
            <div className="flex items-start mb-4">
              <div className="flex items-start justify-between space-x-3 flex-1">
                <div className="flex flex-col flex-1 min-w-0">
                  <Link to={`/${organizationId}/canvas/${canvas.id}`} className="block text-left w-full">
                    <Heading
                      level={3}
                      className="!text-base font-medium text-gray-800 transition-colors mb-0 !leading-6 line-clamp-2 max-w-[15vw] truncate"
                    >
                      {canvas.name}
                    </Heading>
                  </Link>
                </div>
              </div>
            </div>

            <div className="mb-4">
              <Text className="text-[13px] text-left text-gray-600 dark:text-gray-400 line-clamp-3 mt-2">
                {canvas.description || ""}
              </Text>
            </div>
          </div>

          <div className="flex justify-between items-center">
            <div className="flex items-center space-x-2">
              <Avatar
                src={canvas.createdBy.avatar}
                initials={canvas.createdBy.initials}
                alt={canvas.createdBy.name}
                className="w-6 h-6 bg-blue-700 dark:bg-blue-900 text-blue-100 dark:text-blue-100"
              />
              <div className="text-gray-500 text-left">
                <p className="text-xs text-gray-600 dark:text-gray-400 leading-none mb-1">
                  Created by <strong>{canvas.createdBy.name}</strong>
                </p>
                <p className="text-xs text-gray-600 dark:text-gray-400 leading-none">Created at {canvas.createdAt}</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white dark:bg-gray-800rounded-lg border border-gray-200 dark:border-gray-800 hover:shadow-sm transition-shadow group">
      <div className="p-4 pl-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4 flex-1">
            <div className="flex-1 min-w-0">
              <div className="flex items-center space-x-3 mb-1">
                <Link to={`/${organizationId}/canvas/${canvas.id}`} className="block text-left">
                  <Heading
                    level={3}
                    className="text-base font-semibold text-gray-800 transition-colors truncate max-w-[40vw]"
                  >
                    {canvas.name}
                  </Heading>
                </Link>
              </div>

              <Text className="text-[13px] text-left text-gray-600 dark:text-gray-400 mb-2 line-clamp-1 !mb-0">
                {canvas.description || ""}
              </Text>
            </div>
          </div>

          <div className="flex items-center space-x-2 flex-shrink-0">
            <div className="flex items-center space-x-2">
              <div className="text-gray-500 text-right">
                <p className="text-xs text-gray-600 dark:text-gray-400 leading-none mb-1">
                  Created by <strong>{canvas.createdBy.name}</strong>
                </p>
                <p className="text-xs text-gray-600 dark:text-gray-400 leading-none">Created at {canvas.createdAt}</p>
              </div>
              <Avatar
                src={canvas.createdBy.avatar}
                initials={canvas.createdBy.initials}
                alt={canvas.createdBy.name}
                className="w-6 h-6 bg-blue-700 dark:bg-blue-900 text-blue-100 dark:text-blue-100"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
