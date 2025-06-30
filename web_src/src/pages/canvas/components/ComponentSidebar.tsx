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
      className="flex items-center p-2 mb-1 bg-gray-50 hover:bg-gray-100 rounded cursor-pointer border border-gray-200"
      draggable
      onDragStart={onDragStart}
      onClick={onClick}
    >
      {icon && (
        <span className="material-symbols-outlined text-gray-600 mr-2 text-sm">
          {icon}
        </span>
      )}
      <span className="text-sm text-gray-800 truncate">{label}</span>
      <div className="ml-auto flex gap-1">
        <button className="text-xs text-gray-500 hover:text-gray-700 px-1">
          <span className="material-symbols-outlined text-xs">add</span>
        </button>
        <button className="text-xs text-gray-500 hover:text-gray-700 px-1">
          <span className="material-symbols-outlined text-xs">drag_indicator</span>
        </button>
      </div>
    </div>
  );
};

interface NodeGroupProps {
  title: string;
  icon?: string;
  children: React.ReactNode;
}

const NodeGroup = ({ title, icon, children }: NodeGroupProps) => {
  return (
    <div className="mb-4">
      <div className="flex items-center w-full p-2 text-left text-sm font-medium text-gray-700">
        {icon && (
          <span className="material-symbols-outlined text-gray-600 mr-2 text-sm">
            {icon}
          </span>
        )}
        <span className="uppercase tracking-wide text-xs font-semibold">{title}</span>
      </div>
      <div className="ml-6 mt-2">
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
  const { stages, event_sources } = useCanvasStore();
  const [searchTerm, setSearchTerm] = useState('');

  const handleDragStart = (event: React.DragEvent, nodeType: string, data: DragData) => {
    event.dataTransfer.setData('application/reactflow', nodeType);
    event.dataTransfer.setData('application/nodedata', JSON.stringify(data));
    event.dataTransfer.effectAllowed = 'move';
  };

  const filteredStages = stages.filter(stage => 
    stage.metadata?.name?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const filteredEventSources = event_sources.filter((eventSource: EventSourceWithEvents) => 
    eventSource.metadata?.name?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <div
      className={`fixed top-12 left-0 h-screen bg-white border-r border-gray-200 transition-all duration-300 ease-linear z-20 ${
        isOpen ? 'w-80' : 'w-0'
      } overflow-hidden`}
      style={{
        boxShadow: isOpen ? 'rgba(0,0,0,0.07) 2px 0 12px' : 'none',
        height: 'calc(100vh - 4rem)'
      }}
    >
      <div className="flex flex-col h-full">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200">
          <h2 className="text-lg font-semibold text-gray-900">Components</h2>
          <button
            onClick={onToggle}
            className="p-1 hover:bg-gray-100 rounded text-gray-500 hover:text-gray-700"
          >
            <span className="material-symbols-outlined">menu_open</span>
          </button>
        </div>

        {/* Search */}
        <div className="p-4 border-b border-gray-200">
          <input
            type="text"
            placeholder="Search…"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {/* Stages Section */}
          <NodeGroup title="Stages" icon="rocket_launch">
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
          <NodeGroup title="Event Sources" icon="sensors">
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