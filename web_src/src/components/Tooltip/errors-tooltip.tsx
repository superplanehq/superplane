import Tippy from '@tippyjs/react/headless';
import { MaterialSymbol } from '@/components/MaterialSymbol/material-symbol';
import { useState } from 'react';

export interface TooltipError {
  id: string;
  message: string;
  type: ErrorType;
}

export enum ErrorType {
  ERROR = 'ERROR',
  WARNING = 'WARNING',
  INFO = 'INFO'
}

interface ErrorsTooltipProps {
  errors: TooltipError[];
  onAcknowledge: (errorId: string) => void;
  onErrorClick?: (error: TooltipError) => void;
  className?: string;
  isLoading?: boolean;
  title?: string;
}

export function ErrorsTooltip({ errors, onAcknowledge, onErrorClick, title = 'Errors', className = '', isLoading = false }: ErrorsTooltipProps) {
  const [hoveredGroupIndex, setHoveredGroupIndex] = useState<number | null>(null);
  const getIconName = (type: ErrorType) => {
    switch (type) {
      case 'ERROR':
        return 'error';
      case 'WARNING':
        return 'warning';
      case 'INFO':
        return 'info';
      default:
        return 'info';
    }
  };


  const getIconColor = (type: ErrorType) => {
    switch (type) {
      case 'ERROR':
        return 'text-red-600 dark:text-red-500';
      case 'WARNING':
        return 'text-yellow-600 dark:text-yellow-500';
      case 'INFO':
        return 'text-blue-600 dark:text-blue-500';
      default:
        return 'text-blue-600 dark:text-blue-500';
    }
  };

  const groupedErrors = errors.reduce((acc, error) => {
    const key = `${error.type}-${error.message}`;
    if (!acc[key]) {
      acc[key] = {
        type: error.type || 'INFO',
        message: error.message || 'No message',
        count: 0,
        errorIds: [],
        errors: []
      };
    }
    acc[key].count++;
    if (error.id) {
      acc[key].errorIds.push(error.id);
    }
    acc[key].errors.push(error);
    return acc;
  }, {} as Record<string, { type: ErrorType; message: string; count: number; errorIds: string[]; errors: TooltipError[] }>);

  const sortedGroups = Object.values(groupedErrors).sort((a, b) => {
    const typeOrder = { 'ERROR': 0, 'WARNING': 1, 'INFO': 2 };
    return (typeOrder[a.type as keyof typeof typeOrder] || 3) - (typeOrder[b.type as keyof typeof typeOrder] || 3);
  });

  const getTypeLabel = (type: ErrorType, count: number) => {
    const labels: Record<ErrorType, string> = {
      'ERROR': count === 1 ? 'error' : 'errors',
      'WARNING': count === 1 ? 'warning' : 'warnings',
      'INFO': count === 1 ? 'info' : 'infos',
    };
    return labels[type] || 'errors';
  };

  const getBackgroundColor = (type: ErrorType) => {
    switch (type) {
      case 'ERROR':
        return 'bg-red-50 dark:bg-red-900/20';
      case 'WARNING':
        return 'bg-yellow-50 dark:bg-yellow-900/20';
      case 'INFO':
        return 'bg-blue-50 dark:bg-blue-900/20';
      default:
        return 'bg-blue-50 dark:bg-blue-900/20';
    }
  };


  const errorCounts = {
    error: errors.filter(error => error.type === 'ERROR').length,
    warning: errors.filter(error => error.type === 'WARNING').length,
    info: errors.filter(error => error.type === 'INFO').length
  };

  const dominantErrorType = (() => {
    if (errorCounts.error > 0) return 'ERROR';
    if (errorCounts.warning > 0) return 'WARNING';
    return 'INFO';
  })();

  if (errors.length === 0 && !isLoading) {
    return null;
  }

  if (isLoading) {
    return (
      <div
        className={`transition-colors cursor-pointer relative ${className}`}
        role="button"
        tabIndex={0}
      >
        <div className="flex items-center gap-1 px-2 py-1 rounded-md bg-gray-50 dark:bg-gray-800">
          <MaterialSymbol name="sync" size="sm" className="text-gray-500 dark:text-gray-400 animate-spin" fill={1} />
        </div>
      </div>
    );
  }

  return (
    <Tippy
      render={() => (
        <div className="p-3 min-w-[350px] bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg">
          <div className="text-left font-medium text-sm mb-3 text-gray-900 dark:text-gray-100">
            {title}
          </div>
          <div className="space-y-2 flex flex-col gap-6">
            {sortedGroups.map((group, index) => (
              <div
                key={index}
                className={`text-xs p-2 -m-2 rounded transition-colors ${getBackgroundColor(group.type)}`}
                onMouseEnter={() => setHoveredGroupIndex(index)}
                onMouseLeave={() => setHoveredGroupIndex(null)}
              >
                <div className="flex items-center gap-2">
                  <MaterialSymbol name={getIconName(group.type)} size="sm" className={`${getIconColor(group.type)} flex-shrink-0`} fill={1} />
                  <div
                    className="flex-1 min-w-0 text-left cursor-pointer"
                    onClick={(e) => {
                      e.stopPropagation();
                      if (onErrorClick && group.errors.length > 0) {
                        onErrorClick(group.errors[0]);
                      }
                    }}
                  >
                    <span className={`font-medium ${getIconColor(group.type)}`}>
                      {group.count} {getTypeLabel(group.type, group.count)}:
                    </span>
                    <span className="text-gray-600 dark:text-gray-400 ml-1 break-words">
                      {group.message}
                    </span>
                  </div>
                  <div className="w-6 flex justify-center">
                    {hoveredGroupIndex === index && (
                      <button
                        onClick={async (e: React.MouseEvent) => {
                          e.stopPropagation();
                          await Promise.all(group.errorIds.map(id => onAcknowledge(id)));
                        }}
                        className="border-none bg-transparent rounded cursor-pointer transition-all hover:scale-110"
                      >
                        <MaterialSymbol
                          name="close"
                          size="sm"
                          className="m-0 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 flex-shrink-0"
                          fill={1}
                        />
                      </button>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-600">
            <button
              onClick={async (e) => {
                e.stopPropagation()
                const allErrorIds = sortedGroups.flatMap(group => group.errorIds);
                await Promise.all(allErrorIds.map(id => onAcknowledge(id)));
              }}
              className="text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300 text-xs underline"
            >
              Acknowledge all →
            </button>
          </div>
        </div>
      )}
      placement="bottom-end"
      interactive={true}
      delay={200}
    >
      <div
        className={`transition-colors cursor-pointer relative ${className}`}
        role="button"
        tabIndex={0}
      >
        <div className={`flex items-center gap-1 px-2 py-1 rounded-md ${getBackgroundColor(dominantErrorType as ErrorType)}`}>
          {errorCounts.error > 0 && (
            <div className="flex items-center gap-0.5">
              <MaterialSymbol name="error" size="sm" className="text-red-600 dark:text-red-500" fill={1} />
              <span className="text-xs font-medium text-red-600 dark:text-red-500">{errorCounts.error}</span>
            </div>
          )}
          {errorCounts.warning > 0 && (
            <div className="flex items-center gap-0.5">
              <MaterialSymbol name="warning" size="sm" className="text-yellow-600 dark:text-yellow-500" fill={1} />
              <span className="text-xs font-medium text-yellow-600 dark:text-yellow-500">{errorCounts.warning}</span>
            </div>
          )}
          {errorCounts.info > 0 && (
            <div className="flex items-center gap-0.5">
              <MaterialSymbol name="info" size="sm" className="text-blue-600 dark:text-blue-500" fill={1} />
              <span className="text-xs font-medium text-blue-600 dark:text-blue-500">{errorCounts.info}</span>
            </div>
          )}
        </div>
      </div>
    </Tippy>
  );
}