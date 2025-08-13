import React from 'react';
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol';
import { Button } from '../../../components/Button/button';
import { Badge } from '../../../components/Badge/badge';

export interface SidebarItemProps {
  title: string;
  subtitle?: string;
  icon?: React.ReactNode;
  onClickAddNode?: () => void;
  onDragStart?: (e: React.DragEvent) => void;
  className?: string;
  comingSoon?: boolean;
  showSubtitle?: boolean;
  disabled?: boolean;
}

export function SidebarItem({
  title,
  subtitle,
  icon,
  onClickAddNode,
  onDragStart,
  className = '',
  comingSoon = false,
  showSubtitle = false,
  disabled = false
}: SidebarItemProps) {
  return (
    <div
      className={`rounded-md flex items-center pl-2 pr-2 py-3 relative mb-2 group ${comingSoon || disabled
        ? 'bg-gray-50 dark:bg-zinc-800 opacity-60 cursor-not-allowed'
        : `cursor-grab bg-gray-100 dark:bg-zinc-800 hover:bg-gray-200 dark:hover:bg-zinc-700 ${className}`
        }`}
      draggable={!comingSoon && !disabled}
      onDragStart={comingSoon || disabled ? undefined : onDragStart}
    >
      <div className="flex items-center space-x-3 pr-1 flex-1 truncate">
        {icon && (
          <div className={comingSoon ? 'opacity-100' : ''}>
            {icon}
          </div>
        )}
        <div className="flex-1 min-w-0">
          <div className={`text-sm font-medium truncate flex items-center gap-2 justify-between ${comingSoon || disabled ? 'text-gray-500 dark:text-zinc-400' : 'text-gray-900 dark:text-zinc-100'
            }`}>
            {title}
          </div>
          {showSubtitle && subtitle && (
            <div className={`text-xs truncate ${comingSoon || disabled ? 'text-gray-500 dark:text-zinc-400' : 'text-gray-500 dark:text-zinc-400'
              }`}>
              {subtitle}
            </div>
          )}
        </div>
      </div>
      {comingSoon && (
        <Badge color="indigo" className="text-xs">
          Soon
        </Badge>
      )}
      {!comingSoon && (
        <div className="flex items-center">
          <Button
            plain
            onClick={disabled ? undefined : onClickAddNode}
            className="!px-1 !py-0 mr-2 opacity-0 group-hover:opacity-100"
            disabled={disabled}
          >
            <MaterialSymbol name={disabled ? 'block' : 'add'} size="md" />
          </Button>
        </div>
      )}
    </div>
  );
}