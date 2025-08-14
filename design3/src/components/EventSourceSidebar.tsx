import React, { useState } from 'react';
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol';
import { Button } from './lib/Button/button';
import { ControlledTabs, type Tab } from './lib/Tabs/tabs';
import { Text } from './lib/Text/text';
import { Link } from './lib/Link/link';
import { Badge } from './lib/Badge/badge';
import clsx from 'clsx';
import { EmptyState } from './lib/EmptyState/empty-state';

interface EventData {
  id: string;
  url: string;
  status: 'pending' | 'discarded' | 'forwarded';
  timestamp: string;
  processingTime?: number;
  type?: string;
  payload?: Record<string, any>;
  headers?: Record<string, string>;
  triggeredWorkflows?: Array<{
    id: string;
    name: string;
    status: 'running' | 'completed' | 'failed';
  }>;
}

interface EventSourceSidebarProps {
  nodeId?: string;
  nodeTitle?: string;
  nodeIcon?: string;
  isOpen?: boolean;
  onClose?: () => void;
  className?: string;
  events?: Array<{
    id: string;
    url: string;
    type?: string;
    enabled?: boolean;
    status?: 'pending' | 'discarded' | 'forwarded';
    timestamp?: string;
    processingTime?: number;
  }>;
}

// Mock event data for demonstration
const mockRecentEvents: EventData[] = [
  {
    id: 'event-1',
    url: 'https://github.com/owner/repo/push',
    status: 'pending',
    timestamp: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
    type: 'push',
    payload: {
      branch: 'main',
      commit: 'a1b2c3d',
      author: 'john.doe'
    },
    headers: {
      'x-github-event': 'push',
      'user-agent': 'GitHub-Hookshot'
    }
  },
  {
    id: 'event-2',
    url: 'https://github.com/owner/repo/pull_request',
    status: 'discarded',
    timestamp: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
    type: 'pull_request',
    payload: {
      action: 'opened',
      number: 42,
      branch: 'feature/auth'
    },
    headers: {
      'x-github-event': 'pull_request',
      'user-agent': 'GitHub-Hookshot'
    }
  },
  {
    id: 'event-3',
    url: 'https://github.com/owner/repo/push',
    status: 'forwarded',
    timestamp: new Date(Date.now() - 25 * 60 * 1000).toISOString(),
    processingTime: 245,
    type: 'push',
    payload: {
      branch: 'develop',
      commit: 'x7y8z9w',
      author: 'jane.smith'
    },
    headers: {
      'x-github-event': 'push',
      'user-agent': 'GitHub-Hookshot'
    },
    triggeredWorkflows: [
      {
        id: 'wf-1',
        name: 'CI Pipeline',
        status: 'completed'
      },
      {
        id: 'wf-2', 
        name: 'Deploy to Staging',
        status: 'running'
      }
    ]
  }
];

const mockHistoryEvents: EventData[] = [
  {
    id: 'hist-event-1',
    url: 'https://github.com/owner/repo/push',
    status: 'forwarded',
    timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
    processingTime: 156,
    type: 'push',
    payload: {
      branch: 'main',
      commit: 'f5g6h7i',
      author: 'alice.wilson'
    },
    triggeredWorkflows: [
      {
        id: 'wf-3',
        name: 'Production Deploy',
        status: 'completed'
      }
    ]
  },
  {
    id: 'hist-event-2',
    url: 'https://github.com/owner/repo/release',
    status: 'forwarded',
    timestamp: new Date(Date.now() - 4 * 60 * 60 * 1000).toISOString(),
    processingTime: 89,
    type: 'release',
    payload: {
      tag: 'v2.1.0',
      name: 'Version 2.1.0',
      author: 'release-bot'
    },
    triggeredWorkflows: [
      {
        id: 'wf-4',
        name: 'Release Pipeline',
        status: 'completed'
      },
      {
        id: 'wf-5',
        name: 'Update Documentation',
        status: 'completed'
      }
    ]
  }
];

export function EventSourceSidebar({
  nodeId,
  nodeTitle = 'Event Source',
  nodeIcon = 'semaphore',
  isOpen = false,
  onClose,
  className,
  events = []
}: EventSourceSidebarProps) {
  console.log('EventSourceSidebar render:', { isOpen, nodeTitle, events: events?.length }); // Debug log
  
  // Convert events to EventData format if needed
  const convertedEvents = events.length > 0 
    ? events.map((event, index) => ({
        id: event.id,
        url: event.url,
        status: event.status || 'pending',
        timestamp: event.timestamp || new Date().toISOString(),
        processingTime: event.processingTime,
        type: event.type || 'webhook',
        payload: {
          source: 'event-source-node',
          index: index
        },
        headers: {
          'content-type': 'application/json'
        }
      } as EventData))
    : mockRecentEvents;
  const [activeTab, setActiveTab] = useState<'activity' | 'history' | 'settings'>('activity');
  const [expandedEvents, setExpandedEvents] = useState<Set<string>>(new Set());
  const [expandedHistoryEvents, setExpandedHistoryEvents] = useState<Set<string>>(new Set());

  const tabs: Tab[] = [
    { id: 'activity', label: 'Activity' },
    { id: 'history', label: 'History' },
    { id: 'settings', label: 'Settings' }
  ];

  const getStatusConfig = (status: string) => {
    switch (status) {
      case 'pending':
        return {
          icon: 'schedule',
          color: 'text-yellow-600 dark:text-yellow-400',
          bgColor: 'bg-yellow-50 dark:bg-yellow-900/20',
          borderColor: 'border-yellow-200 dark:border-yellow-800',
          iconColor: 'text-yellow-600 dark:text-yellow-400',
          label: 'Pending'
        };
      case 'discarded':
        return {
          icon: 'block',
          color: 'text-red-600 dark:text-red-400',
          bgColor: 'bg-red-50 dark:bg-red-900/20',
          borderColor: 'border-red-200 dark:border-red-800',
          iconColor: 'text-red-600 dark:text-red-400',
          label: 'Discarded'
        };
      case 'forwarded':
        return {
          icon: 'check_circle',
          color: 'text-green-600 dark:text-green-400',
          bgColor: 'bg-green-50 dark:bg-green-900/20',
          borderColor: 'border-green-200 dark:border-green-800',
          iconColor: 'text-green-600 dark:text-green-400',
          label: 'Forwarded'
        };
      default:
        return {
          icon: 'help',
          color: 'text-gray-500 dark:text-zinc-400',
          bgColor: 'bg-gray-50 dark:bg-zinc-800',
          borderColor: 'border-gray-200 dark:border-zinc-700',
          iconColor: 'text-gray-500 dark:text-zinc-400',
          label: 'Unknown'
        };
    }
  };

  const toggleEventExpansion = (eventId: string) => {
    setExpandedEvents(prev => {
      const newSet = new Set(prev);
      if (newSet.has(eventId)) {
        newSet.delete(eventId);
      } else {
        newSet.add(eventId);
      }
      return newSet;
    });
  };

  const toggleHistoryEventExpansion = (eventId: string) => {
    setExpandedHistoryEvents(prev => {
      const newSet = new Set(prev);
      if (newSet.has(eventId)) {
        newSet.delete(eventId);
      } else {
        newSet.add(eventId);
      }
      return newSet;
    });
  };

  const formatTimestamp = (timestamp: string) => {
    const now = new Date();
    const eventTime = new Date(timestamp);
    const diffMs = now.getTime() - eventTime.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return eventTime.toLocaleDateString();
  };

  const truncateUrl = (url: string, maxLength: number = 30) => {
    if (url.length <= maxLength) return url;
    return url.substring(0, maxLength) + '...';
  };

  const renderEventDetails = (event: EventData) => (
    <div className="mt-3 space-y-3">
      {/* Event Metadata */}
      <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900">
        <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-2 font-bold">
          Event Details
        </div>
        <div className="space-y-2 text-xs">
          <div className="flex items-center justify-between">
            <span className="text-gray-600 dark:text-zinc-400">URL</span>
            <span className="font-mono text-gray-900 dark:text-zinc-200 truncate max-w-48" title={event.url}>
              {event.url}
            </span>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-gray-600 dark:text-zinc-400">Type</span>
            <Badge className="font-mono !text-xs">{event.type}</Badge>
          </div>
          <div className="flex items-center justify-between">
            <span className="text-gray-600 dark:text-zinc-400">Received</span>
            <span className="font-mono text-gray-900 dark:text-zinc-200">
              {formatTimestamp(event.timestamp)}
            </span>
          </div>
          {event.processingTime && (
            <div className="flex items-center justify-between">
              <span className="text-gray-600 dark:text-zinc-400">Processing Time</span>
              <span className="font-mono text-gray-900 dark:text-zinc-200">
                {event.processingTime}ms
              </span>
            </div>
          )}
        </div>
      </div>

      {/* Payload */}
      {event.payload && (
        <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900">
          <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-2 font-bold">
            Payload
          </div>
          <div className="space-y-1">
            {Object.entries(event.payload).map(([key, value]) => (
              <div key={key} className="flex items-center justify-between">
                <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium">{key}</span>
                <Badge className="font-mono !text-xs">{String(value)}</Badge>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Triggered Workflows */}
      {event.triggeredWorkflows && event.triggeredWorkflows.length > 0 && (
        <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900">
          <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-2 font-bold">
            Triggered Workflows
          </div>
          <div className="space-y-2">
            {event.triggeredWorkflows.map((workflow) => (
              <div key={workflow.id} className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <MaterialSymbol 
                    name={workflow.status === 'completed' ? 'check_circle' : workflow.status === 'running' ? 'sync' : 'error'}
                    size="sm"
                    className={
                      workflow.status === 'completed' ? 'text-green-500' :
                      workflow.status === 'running' ? 'text-blue-500 animate-spin' : 'text-red-500'
                    }
                  />
                  <span className="text-xs font-medium text-gray-900 dark:text-zinc-200">
                    {workflow.name}
                  </span>
                </div>
                <Badge 
                  className={`!text-xs ${
                    workflow.status === 'completed' ? '!bg-green-100 !text-green-800' :
                    workflow.status === 'running' ? '!bg-blue-100 !text-blue-800' : '!bg-red-100 !text-red-800'
                  }`}
                >
                  {workflow.status}
                </Badge>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );

  return (
    <div className={clsx(
      'fixed right-0 top-0 h-full w-96 bg-white dark:bg-zinc-900 border-l border-gray-200 dark:border-zinc-700 shadow-xl z-50 flex flex-col transform transition-all duration-300 ease-in-out',
      isOpen ? 'translate-x-0' : 'translate-x-full',
      className
    )}>
      {/* Header */}
      <div className="flex items-center justify-between p-4 pb-2">
        <div className="flex items-center gap-3">
          <div className='rounded-lg bg-zinc-100 dark:bg-zinc-700 p-2'>
            {nodeIcon === 'semaphore' ? (
              <img width={24} src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAMAAABF0y+mAAAAM1BMVEVHcEwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADbQS4qAAAAEXRSTlMAYq64jCpx/8oGF/mjNBDW6uM72ZcAAACJSURBVHgBzdBFAoAwEATBjbv8/7WwTHA50ziFhv6ekEpp80jWIR/uJt1W/LCbwpTV6a7ZcYV3vePq1QwOGu8n1sifJvb7Nm1EgVd8J6x0vWqlkBxU98XmkxlaxwM8jYzjxLwX+Gtr2hWGO1F1m8Ik0VWTtmMU6FR0aLe73g0FP8zSU0YrJQX9vAn47gbljcJgwwAAAABJRU5ErkJggg==" alt="" />
            ) : nodeIcon === 'github' ? (
              <img width={24} src="/images/github-logo.svg" alt="GitHub" />
            ) : (
              <MaterialSymbol name="sensors" size="md" />
            )}
          </div>
          <h3 className="font-semibold text-gray-900 dark:text-white">{nodeTitle}</h3>
        </div>
        <Button plain onClick={onClose}>
          <MaterialSymbol name="close" size="lg" className="text-gray-500 dark:text-zinc-400" />
        </Button>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200 dark:border-zinc-700">
        <ControlledTabs
          tabs={tabs}
          activeTab={activeTab}
          onTabChange={(tabId) => setActiveTab(tabId as 'activity' | 'history' | 'settings')}
          variant="underline"
        />
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto">
        {activeTab === 'activity' && (
          <div className="p-4 space-y-6">
            {/* Recent Events */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <Text className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide">
                  RECENT EVENTS
                </Text>
                <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                  View all
                </Link>
              </div>
              
              <div className="space-y-3">
                {convertedEvents.map((event) => {
                  const statusConfig = getStatusConfig(event.status);
                  const isExpanded = expandedEvents.has(event.id);
                  
                  return (
                    <div key={event.id} className={`border ${statusConfig.borderColor} ${statusConfig.bgColor} cursor-pointer rounded-lg`}>
                      <div 
                        className="p-3"
                        onClick={() => toggleEventExpansion(event.id)}
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2 truncate pr-2">
                            <MaterialSymbol 
                              name={statusConfig.icon} 
                              size="lg" 
                              className={statusConfig.iconColor}
                            />
                            <span className="font-medium truncate text-sm dark:text-white font-mono">
                              {truncateUrl(event.url)}
                            </span>
                          </div>
                          <div className="flex items-center gap-3">
                            {!isExpanded && (
                              <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">
                                {formatTimestamp(event.timestamp)}
                              </span>
                            )}
                            <MaterialSymbol 
                              name={isExpanded ? 'expand_less' : 'expand_more'} 
                              size="lg" 
                              className="text-gray-600 dark:text-zinc-400" 
                            />
                          </div>
                        </div>
                        
                        {isExpanded && renderEventDetails(event)}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        )}
        
        {activeTab === 'history' && (
          <div className="p-4 space-y-6">
            {/* Historical Events */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <Text className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide">
                  EVENT HISTORY ({mockHistoryEvents.length})
                </Text>
              </div>
              
              <div className="space-y-3">
                {mockHistoryEvents.map((event) => {
                  const statusConfig = getStatusConfig(event.status);
                  const isExpanded = expandedHistoryEvents.has(event.id);
                  
                  return (
                    <div key={event.id} className={`border ${statusConfig.borderColor} ${statusConfig.bgColor} cursor-pointer rounded-lg`}>
                      <div 
                        className="p-3"
                        onClick={() => toggleHistoryEventExpansion(event.id)}
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2 truncate pr-2">
                            <MaterialSymbol 
                              name={statusConfig.icon} 
                              size="lg" 
                              className={statusConfig.iconColor}
                            />
                            <span className="font-medium truncate text-sm dark:text-white font-mono">
                              {truncateUrl(event.url)}
                            </span>
                          </div>
                          <div className="flex items-center gap-3">
                            {!isExpanded && (
                              <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">
                                {formatTimestamp(event.timestamp)}
                              </span>
                            )}
                            <MaterialSymbol 
                              name={isExpanded ? 'expand_less' : 'expand_more'} 
                              size="lg" 
                              className="text-gray-600 dark:text-zinc-400" 
                            />
                          </div>
                        </div>
                        
                        {isExpanded && renderEventDetails(event)}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        )}
        
        {activeTab === 'settings' && (
          <div className="p-4">
            <EmptyState 
              size="sm" 
              icon="settings"
              title="Settings"
              body="Event source configuration and settings will be available here."
              className="bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700"
            />
          </div>
        )}
      </div>
    </div>
  );
}