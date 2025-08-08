import React from 'react';
import { MaterialSymbol } from '../MaterialSymbol/material-symbol';
import { Button } from '../Button/button';
import { Badge } from '../Badge/badge';

export interface SidebarItemProps {
  title: string;
  subtitle?: string;
  icon?: React.ReactNode;
  onClickAddNode?: () => void;
  onDragStart?: (e: React.DragEvent) => void;
  className?: string;
  comingSoon?: boolean;
  showSubtitle?: boolean;
}

export function SidebarItem({ 
  title, 
  subtitle, 
  icon,
  onClickAddNode,
  onDragStart,
  className = '',
  comingSoon = false,
  showSubtitle = false
}: SidebarItemProps) {
  return (
    <div 
      className={`rounded-md flex items-center pl-2 pr-2 py-3 relative mb-2 group ${
        comingSoon 
          ? 'bg-gray-50 opacity-60 cursor-not-allowed' 
          : `cursor-grab bg-gray-100 hover:bg-gray-200 ${className}`
      }`}
      draggable={!comingSoon}
      onDragStart={comingSoon ? undefined : onDragStart}
    >
      <div className="flex items-center space-x-3 pr-1 flex-1 truncate">
        {icon && (
          <div className={comingSoon ? 'opacity-100' : ''}>
            {icon}
          </div>
        )}
        <div className="flex-1 min-w-0">
          <div className={`text-sm font-medium truncate flex items-center gap-2 justify-between ${
            comingSoon ? 'text-gray-900' : 'text-gray-900'
          }`}>
            {title}
            
          </div>
          {showSubtitle && subtitle && (
            <div className={`text-xs truncate ${
              comingSoon ? 'text-gray-500' : 'text-gray-500'
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
            onClick={onClickAddNode} 
            className="!px-1 !py-0 mr-2 opacity-0 group-hover:opacity-100"
          >
            <MaterialSymbol name="add" size="md" />
          </Button>
        
        <MaterialSymbol 
          name="drag_indicator" 
          size="md" 
          className={comingSoon ? 'opacity-30' : ''}
        />
      </div>
      )}
    </div>
  );
}