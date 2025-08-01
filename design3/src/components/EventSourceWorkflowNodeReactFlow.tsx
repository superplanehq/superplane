import React, { useState, useCallback, useRef } from 'react';
import { Handle, Position } from '@xyflow/react';
import clsx from 'clsx';
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol';
import { Button } from './lib/Button/button';
import { Input } from './lib/Input/input';
import { Field, Label } from './lib/Fieldset/fieldset';

export interface EventSourceWorkflowNodeReactFlowData {
  id: string;
  title: string;
  cluster?: string;
  events: Array<{
    id: string;
    url: string;
    type?: string;
    enabled?: boolean;
  }>;
  icon?: string;
  selected?: boolean;
  isEditMode?: boolean;
}

interface EventSourceWorkflowNodeReactFlowProps {
  data: EventSourceWorkflowNodeReactFlowData;
  selected?: boolean;
}

export function EventSourceWorkflowNodeReactFlow({ 
  data, 
  selected = false 
}: EventSourceWorkflowNodeReactFlowProps) {
  const [isEditMode, setIsEditMode] = useState(data.isEditMode || false);
  const [editData, setEditData] = useState({
    title: data.title,
    cluster: data.cluster || '',
    events: [...data.events]
  });

  const handleEditToggle = useCallback(() => {
    setIsEditMode(!isEditMode);
  }, [isEditMode]);

  const handleSave = useCallback(() => {
    // Here you would typically save the data to your backend or state management
    setIsEditMode(false);
  }, []);

  const handleCancel = useCallback(() => {
    setEditData({
      title: data.title,
      cluster: data.cluster || '',
      events: [...data.events]
    });
    setIsEditMode(false);
  }, [data]);

  const handleAddEvent = useCallback(() => {
    setEditData(prev => ({
      ...prev,
      events: [...prev.events, {
        id: `event-${prev.events.length + 1}`,
        url: '',
        type: 'webhook',
        enabled: true
      }]
    }));
  }, []);

  const handleRemoveEvent = useCallback((eventId: string) => {
    setEditData(prev => ({
      ...prev,
      events: prev.events.filter(event => event.id !== eventId)
    }));
  }, []);

  const handleEventChange = useCallback((eventId: string, field: string, value: string) => {
    setEditData(prev => ({
      ...prev,
      events: prev.events.map(event =>
        event.id === eventId ? { ...event, [field]: value } : event
      )
    }));
  }, []);

  const truncateUrl = (url: string, maxLength: number = 40) => {
    if (url.length <= maxLength) return url;
    return url.substring(0, maxLength) + '...';
  };

  // Preview Mode
  if (!isEditMode) {
    return (
      <div 
        className={clsx(
          'bg-white dark:bg-zinc-800 rounded-lg border-2 relative transition-all duration-200 hover:shadow-lg min-w-[320px]',
          selected ? 'border-blue-600 dark:border-zinc-200 ring-2 ring-blue-200 dark:ring-white' : 'border-gray-200 dark:border-zinc-700'
        )}
        style={{ width: 320, boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)' }}
        role="article"
      >
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-zinc-700">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
              <MaterialSymbol 
                name={data.icon || 'sync'} 
                size="md" 
                className="text-blue-600 dark:text-blue-400" 
              />
            </div>
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
              {data.title}
            </h3>
          </div>
          <Button
            plain
            onClick={handleEditToggle}
            className="p-1 hover:bg-gray-100 dark:hover:bg-zinc-700 rounded"
          >
            <MaterialSymbol 
              name="edit" 
              size="sm" 
              className="text-gray-500 dark:text-zinc-400" 
            />
          </Button>
        </div>

        {/* Cluster Section */}
        {data.cluster && (
          <div className="px-4 py-3 border-b border-gray-200 dark:border-zinc-700">
            <div className="text-blue-600 dark:text-blue-400 font-medium">
              {data.cluster}
            </div>
          </div>
        )}

        {/* Events Section */}
        <div className="p-4">
          <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-3">
            EVENTS
          </div>
          <div className="space-y-2">
            {data.events.map((event) => (
              <div
                key={event.id}
                className="flex items-center gap-2 p-2 bg-gray-50 dark:bg-zinc-800 rounded-lg"
              >
                <MaterialSymbol 
                  name="bolt" 
                  size="sm" 
                  className="text-green-600 dark:text-green-400 flex-shrink-0" 
                />
                <span className="text-sm text-gray-800 dark:text-zinc-200 truncate font-mono">
                  {truncateUrl(event.url)}
                </span>
              </div>
            ))}
            {data.events.length === 0 && (
              <div className="text-sm text-gray-500 dark:text-zinc-400 italic">
                No events configured
              </div>
            )}
          </div>
        </div>

        {/* React Flow Handles */}
        <Handle 
          type="target" 
          position={Position.Left} 
          className="w-3 h-3 bg-gray-400 border-2 border-white" 
        />
        <Handle 
          type="source" 
          position={Position.Right} 
          className="w-3 h-3 bg-blue-600 border-2 border-white" 
        />
      </div>
    );
  }

  // Edit Mode
  return (
    <div 
      className={clsx(
        'bg-white dark:bg-zinc-800 rounded-lg border-2 relative transition-all duration-200 hover:shadow-lg min-w-[320px]',
        selected ? 'border-blue-600 dark:border-zinc-200 ring-2 ring-blue-200 dark:ring-white' : 'border-gray-200 dark:border-zinc-700'
      )}
      style={{ width: 380, boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)' }}
      role="article"
    >
      {/* Edit Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-zinc-700 bg-blue-50 dark:bg-blue-900/20">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-blue-100 dark:bg-blue-900 flex items-center justify-center">
            <MaterialSymbol 
              name="edit" 
              size="md" 
              className="text-blue-600 dark:text-blue-400" 
            />
          </div>
          <span className="text-sm font-medium text-blue-800 dark:text-blue-200">
            Edit Event Source
          </span>
        </div>
        <div className="flex items-center gap-2">
          <Button
            onClick={handleSave}
            className="px-3 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-700"
          >
            Save
          </Button>
          <Button
            plain
            onClick={handleCancel}
            className="px-3 py-1 text-xs text-gray-600 dark:text-zinc-400 hover:text-gray-800 dark:hover:text-zinc-200"
          >
            Cancel
          </Button>
        </div>
      </div>

      {/* Edit Form */}
      <div className="p-4 space-y-4">
        {/* Title Field */}
        <Field>
          <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
            Title
          </Label>
          <Input
            value={editData.title}
            onChange={(e) => setEditData(prev => ({ ...prev, title: e.target.value }))}
            placeholder="Enter event source title"
            className="w-full"
          />
        </Field>

        {/* Cluster Field */}
        <Field>
          <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
            Cluster
          </Label>
          <Input
            value={editData.cluster}
            onChange={(e) => setEditData(prev => ({ ...prev, cluster: e.target.value }))}
            placeholder="Enter cluster name"
            className="w-full"
          />
        </Field>

        {/* Events Section */}
        <div>
          <div className="flex items-center justify-between mb-3">
            <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
              Events
            </Label>
            <Button
              onClick={handleAddEvent}
              className="px-2 py-1 text-xs bg-green-600 text-white rounded hover:bg-green-700"
            >
              <MaterialSymbol name="add" size="sm" className="mr-1" />
              Add Event
            </Button>
          </div>
          
          <div className="space-y-2 max-h-40 overflow-y-auto">
            {editData.events.map((event, index) => (
              <div key={event.id} className="flex items-center gap-2">
                <div className="flex-1">
                  <Input
                    value={event.url}
                    onChange={(e) => handleEventChange(event.id, 'url', e.target.value)}
                    placeholder="https://hooks.example.com/webhook"
                    className="w-full text-xs font-mono"
                  />
                </div>
                <Button
                  plain
                  onClick={() => handleRemoveEvent(event.id)}
                  className="p-1 text-red-600 hover:bg-red-50 dark:hover:bg-red-900/20 rounded"
                >
                  <MaterialSymbol name="delete" size="sm" />
                </Button>
              </div>
            ))}
            {editData.events.length === 0 && (
              <div className="text-sm text-gray-500 dark:text-zinc-400 italic p-3 border border-dashed border-gray-300 dark:border-zinc-600 rounded">
                No events configured. Click "Add Event" to add webhook URLs.
              </div>
            )}
          </div>
        </div>
      </div>

      {/* React Flow Handles */}
      <Handle 
        type="target" 
        position={Position.Left} 
        className="w-3 h-3 bg-gray-400 border-2 border-white" 
      />
      <Handle 
        type="source" 
        position={Position.Right} 
        className="w-3 h-3 bg-blue-600 border-2 border-white" 
      />
    </div>
  );
}