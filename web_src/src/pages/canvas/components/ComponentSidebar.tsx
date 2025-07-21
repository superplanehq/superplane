import React, { useState } from 'react';
import { useCanvasStore } from '../store/canvasStore';
import { EventSourceWithEvents } from '../store/types';

interface DragData {
  id?: string;
  name?: string;
  type: string;
}

interface SidebarNodeProps {
  label: string;
  icon?: string;
  onDragStart?: (event: React.DragEvent) => void;
  onClick?: () => void;
}

const SidebarNode = ({ label, icon, onDragStart, onClick }: SidebarNodeProps) => {
  return (
    <div
      className="flex items-center p-2 mb-1 bg-gray-100 hover:bg-gray-200 rounded cursor-pointer"
      draggable
      onDragStart={onDragStart}
      onClick={onClick}
    >
      {icon && (
        <span style={{ fontSize: '1.2rem' }} className="material-symbols-outlined text-gray-600 mr-2 py-2">
          {icon}
        </span>
      )}
      <span className="text-sm text-gray-800 truncate">{label}</span>
      <div className="ml-auto flex gap-1">
        <button className="text-xs text-gray-500 hover:text-gray-700 px-1">
          <span style={{ fontSize: '1rem' }} className="material-symbols-outlined text-xs">add</span>
        </button>
        <button className="text-xs text-gray-500 hover:text-gray-700 px-1">
          <span style={{ fontSize: '1rem' }} className="material-symbols-outlined text-xs">drag_indicator</span>
        </button>
      </div>
    </div>
  );
};

interface NodeGroupProps {
  title: string;
  children: React.ReactNode;
}

const NodeGroup = ({ title, children }: NodeGroupProps) => {
  return (
    <div className="mb-4">
      <div className="flex items-center w-full py-2 text-left font-medium text-gray-700">
        <span className="uppercase tracking-wide text-sm font-semibold">{title}</span>
      </div>
      <div className="mt-2">
        {children}
      </div>
    </div>
  );
};

interface ComponentSidebarProps {
  isOpen: boolean;
  onToggle: () => void;
}

export const ComponentSidebar = ({ isOpen, onToggle }: ComponentSidebarProps) => {
  const { stages, eventSources } = useCanvasStore();
  const [searchTerm, setSearchTerm] = useState('');

  const handleDragStart = (event: React.DragEvent, nodeType: string, data: DragData) => {
    event.dataTransfer.setData('application/reactflow', nodeType);
    event.dataTransfer.setData('application/nodedata', JSON.stringify(data));
    event.dataTransfer.effectAllowed = 'move';
  };

  const filteredStages = stages.filter(stage =>
    stage.metadata?.name?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const filteredEventSources = eventSources.filter((eventSource: EventSourceWithEvents) =>
    eventSource.metadata?.name?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div
      className={`fixed top-[42px] left-0 bg-white transition-all duration-300 ease-linear z-20 ${isOpen ? 'w-80' : 'w-0'
        } overflow-hidden`}
      style={{
        boxShadow: isOpen ? 'rgba(0,0,0,0.07) 2px 0 12px' : 'none',
        height: 'calc(100vh - 48px)'
      }}
    >
      <div className="flex flex-col h-full">
        {/* Header */}
        <div className="flex items-center justify-between p-4">
          <h2 className="text-medium font-semibold text-gray-900">Components</h2>
          <button
            onClick={onToggle}
            className="p-1 hover:bg-gray-100 rounded text-gray-500 hover:text-gray-700"
          >
            <span className="material-symbols-outlined">menu_open</span>
          </button>
        </div>

        {/* Search */}
        <div className="p-4">
          <input
            type="text"
            placeholder="Search…"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {/* Stages Section */}
          <NodeGroup title="Stages">
            {filteredStages.map((stage) => (
              <SidebarNode
                key={stage.metadata?.id}
                label={stage.metadata?.name || 'Unnamed Stage'}
                icon="rocket_launch"
                onDragStart={(e) => handleDragStart(e, 'stage', {
                  id: stage.metadata?.id,
                  name: stage.metadata?.name,
                  type: 'stage'
                })}
              />
            ))}
            {filteredStages.length === 0 && (
              <div className="text-sm text-gray-500 italic">No stages found</div>
            )}
          </NodeGroup>

          {/* Event Sources Section */}
          <NodeGroup title="Event Sources">
            {filteredEventSources.map((eventSource: EventSourceWithEvents) => (
              <SidebarNode
                key={eventSource.metadata?.id}
                label={eventSource.metadata?.name || 'Unnamed Event Source'}
                icon="sensors"
                onDragStart={(e) => handleDragStart(e, 'event_source', {
                  id: eventSource.metadata?.id,
                  name: eventSource.metadata?.name,
                  type: 'event_source'
                })}
              />
            ))}
            {filteredEventSources.length === 0 && (
              <div className="text-sm text-gray-500 italic">No event sources found</div>
            )}
          </NodeGroup>
        </div>
      </div>
    </div>
  );
};