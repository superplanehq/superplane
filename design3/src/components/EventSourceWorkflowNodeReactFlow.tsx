import React, { useState, useCallback, useEffect } from 'react';
import { Handle, Position } from '@xyflow/react';
import clsx from 'clsx';
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol';
import { Button } from './lib/Button/button';
import { Input } from './lib/Input/input';
import { Field, Label } from './lib/Fieldset/fieldset';
import { Link } from './lib/Link/link';
import { BadgeButton } from './lib/Badge/badge';
import { Dialog, DialogTitle, DialogDescription, DialogBody, DialogActions } from './lib/Dialog/dialog';
import { Dropdown, DropdownButton, DropdownMenu, DropdownItem, DropdownLabel } from './lib/Dropdown/dropdown';
import { ControlledTabs, type Tab } from './lib/Tabs/tabs';
import Tippy from '@tippyjs/react';
import { Text } from './lib/Text/text';
import { EmptyState } from './lib/EmptyState/empty-state';
import { EventSourceSidebar } from './EventSourceSidebar';

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
  
  // Local state for events to persist changes in read-only mode  
  // Enhanced events with status indicators for preview mode
  const [displayEvents, setDisplayEvents] = useState(
    data.events.map((event, index) => ({
      ...event,
      status: (index === 0 ? 'pending' : index === 1 ? 'discarded' : 'forwarded') as 'pending' | 'discarded' | 'forwarded',
      timestamp: new Date(Date.now() - (index * 60 * 1000)).toISOString(),
      processingTime: index === 2 ? 245 : undefined
    }))
  );
  
  // Check URL parameter for inline integration mode
  const urlParams = new URLSearchParams(window.location.search);
  const inlineIntegration = urlParams.get('inlineIntegration') === 'true';
  
  // Check URL parameter for event filters version (defaults to 1)
  const eventFiltersVersion = urlParams.get('eventFiltersVersion') === '2' ? 2 : 1;
  
  // Check URL parameter for compact event display (defaults to true)
  const compactEvent = urlParams.get('compactEvent') !== 'false';
  
  // Integration modal state
  const [showIntegrationModal, setShowIntegrationModal] = useState(false);
  const [integrationData, setIntegrationData] = useState({
    name: '',
    orgUrl: '',
    apiToken: {
      secretName: '',
      secretKey: ''
    }
  });
  
  // Inline integration form state
  const [inlineIntegrationData, setInlineIntegrationData] = useState({
    orgUrl: '',
    apiToken: ''
  });
  
  // Secrets management
  const [secrets, setSecrets] = useState<Array<{
    id: string;
    name: string;
    keys: Array<{ key: string; value: string; }>;
  }>>([
    // Mock existing secret
    {
      id: 'secret-1',
      name: 'my semaphore org secrets',
      keys: [{ key: 'API_TOKEN', value: 'hidden' }]
    }
  ]);
  const [newSecret, setNewSecret] = useState({
    name: '',
    key: '',
    value: ''
  });
  
  // Derived: currently selected existing secret (if any)
  const selectedExistingSecret = secrets.find(
    (s) => s.name === integrationData.apiToken.secretName
  );
  
  // API Token tabs state
  const [apiTokenTab, setApiTokenTab] = useState<'existing' | 'new'>('new');
  const [newSecretToken, setNewSecretToken] = useState('');
  const [showGitHubPatInfo, setShowGitHubPatInfo] = useState(false);
  
  // Selected integration state
  const [selectedIntegration, setSelectedIntegration] = useState<string | null>(null);
  
  // Sidebar state for event details
  const [showSidebar, setShowSidebar] = useState(false);
  
  // No state needed for hover-triggered popovers
  
  // Mock filters data
  const appliedFilters = [
    {
      id: 'filter-1',
      type: 'Branch',
      value: 'main',
      operator: 'equals'
    },
    {
      id: 'filter-2', 
      type: 'Event Type',
      value: 'push',
      operator: 'equals'
    }
  ];
  
  // Available integrations (mock data + created ones)
  const [availableIntegrations, setAvailableIntegrations] = useState<Array<{
    id: string;
    name: string;
    orgUrl: string;
    secretName: string;
  }>>([]);
  
  // Define tabs for API Token section
  const apiTokenTabs: Tab[] = [
    {
      id: 'new',
      label: 'Create new secret',
    },
    {
      id: 'existing',
      label: 'Import from existing secret',
    }
  ];


  const handleSave = useCallback(() => {
    // Save as Draft: preserve current editData.events in displayEvents with enhanced properties
    const enhancedEvents = editData.events.map((event, index) => ({
      ...event,
      status: (index === 0 ? 'pending' : index === 1 ? 'discarded' : 'forwarded') as 'pending' | 'discarded' | 'forwarded',
      timestamp: new Date(Date.now() - (index * 60 * 1000)).toISOString(),
      processingTime: index === 2 ? 245 : undefined
    }));
    setDisplayEvents(enhancedEvents);
    
    // Here you would typically save the data to your backend or state management
    setIsEditMode(false);
  }, [editData.events]);

  const handleSaveAndCommit = useCallback(() => {
    // Save & Commit: transition to read-mode without events to show noEvents variant
    setIsEditMode(false);
    
    // Clear events to demonstrate the noEvents variant
    setDisplayEvents([]);
    setEditData(prev => ({
      ...prev,
      events: []
    }));
    
    // Here you would typically commit the changes to your backend or state management
    console.log('Save & Commit: Transitioning to read-mode without events');
  }, []);

  const handleCancel = useCallback(() => {
    setEditData({
      title: data.title,
      cluster: data.cluster || '',
      events: [...data.events]
    });
    setIsEditMode(false);
  }, [data]);


  const handleCreateIntegration = useCallback(() => {
    setShowIntegrationModal(true);
  }, []);


  const handleSaveIntegration = useCallback(() => {
    // Create new integration
    const newIntegration = {
      id: `integration-${Date.now()}`,
      name: integrationData.name,
      orgUrl: integrationData.orgUrl,
      secretName: integrationData.apiToken.secretName
    };
    
    // If creating a new secret, add it to secrets
    if (apiTokenTab === 'new' && newSecretToken) {
      const newSecret = {
        id: `secret-${Date.now()}`,
        name: `${integrationData.name} Secret`,
        keys: [{ key: 'API_TOKEN', value: newSecretToken }]
      };
      setSecrets(prev => [...prev, newSecret]);
      newIntegration.secretName = newSecret.name;
    }
    
    // Add the integration to available integrations and select it
    setAvailableIntegrations(prev => [...prev, newIntegration]);
    setSelectedIntegration(newIntegration.id);
    
    // Close modal and reset form
    setShowIntegrationModal(false);
    setIntegrationData({
      name: '',
      orgUrl: '',
      apiToken: { secretName: '', secretKey: '' }
    });
    setNewSecretToken('');
    setApiTokenTab('new');
  }, [integrationData, apiTokenTab, newSecretToken]);

  const truncateUrl = (url: string, maxLength: number = 40) => {
    if (url.length <= maxLength) return url;
    return url.substring(0, maxLength) + '...';
  };


  // Watch for node selection to open sidebar
  useEffect(() => {
    console.log('Selected prop changed:', selected);
    if (selected && !isEditMode) {
      console.log('Node is selected and not in edit mode, opening sidebar');
      setShowSidebar(true);
    } else if (!selected) {
      console.log('Node is not selected, closing sidebar');
      setShowSidebar(false);
    }
  }, [selected, isEditMode]);

  // Handle sidebar close - also deselect the node
  const handleSidebarClose = useCallback(() => {
    console.log('Closing sidebar');
    setShowSidebar(false);
    // Note: We don't deselect the node here as React Flow manages that
  }, []);

  // Preview Mode
  console.log('EventSourceWorkflowNode render:', { isEditMode, showSidebar, selected }); // Debug log
  
  if (!isEditMode) {
    console.log('Rendering preview mode'); // Debug log
    return (
      <>
        <div 
          className={clsx(
            'bg-white dark:bg-zinc-800 rounded-lg border-2 relative transition-all duration-200 hover:shadow-lg hover:border-blue-300 dark:hover:border-blue-600 min-w-[320px] cursor-pointer',
            selected ? 'border-blue-600 dark:border-zinc-200 ring-2 ring-blue-200 dark:ring-white' : 'border-gray-200 dark:border-zinc-700'
          )}
          style={{ width: 320, boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)' }}
          role="article"
        >
        
        
        {/* Header */}
        <div className="flex flex-col p-4 border-b border-gray-200 dark:border-zinc-700">
          <div className='flex items-start flex-row justify-between w-full'>

            <div className="flex items-center flex-grow-1 gap-3">
              <div className='flex items-center content-center bg-zinc-100 dark:bg-zinc-700 rounded-md w-10 h-10'>
                {data.icon === 'semaphore' ? (
                  <img width={24} height={24} className='m-auto' src='/images/semaphore-logo-sign-black.svg' alt="Semaphore" />
                ) : data.icon === 'github' ? (
                  <img width={24} height={24} className='m-auto' src='/images/github-logo.svg' alt="GitHub" />
                ) : (
                  <img width={24} height={24} className='m-auto' src='https://upload.wikimedia.org/wikipedia/commons/3/39/Kubernetes_logo_without_workmark.svg' alt="Kubernetes" />
                )}
              </div>
           
              <h3 className="text-md font-semibold text-gray-900 dark:text-white truncate">
                {data.title}
              </h3>
              
           
           
          </div>
           {/* Configuration indicator - only show when properly configured (no events but saved) */}
           <span className='hidden'>
           <BadgeButton color='green'>
            <MaterialSymbol name="check" size="sm" />
           </BadgeButton>
          </span>
          </div>
          <div className='flex items-center gap-3 mt-1 text-blue-600 dark:text-blue-300 mt-4'>
              <Tippy
                content={
                  <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 min-w-[250px]">
                    <div className="text-sm font-medium text-zinc-900 dark:text-white mb-3">
                      Project Configuration
                    </div>
                    <div className="space-y-2">
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-zinc-600 dark:text-zinc-400">Project:</span>
                        <span className="text-sm font-mono text-zinc-800 dark:text-zinc-200 bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded">
                          semaphore-project
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-zinc-600 dark:text-zinc-400">Organization:</span>
                        <span className="text-sm text-zinc-700 dark:text-zinc-300">
                          {data.icon === 'semaphore' ? 'Semaphore CI' : data.icon === 'github' ? 'GitHub' : 'External Service'}
                        </span>
                      </div>
                      <div className="flex items-center justify-between">
                        <span className="text-sm text-zinc-600 dark:text-zinc-400">Status:</span>
                        <div className="flex items-center gap-2">
                          <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                          <span className="text-xs text-green-600 dark:text-green-400">Connected</span>
                        </div>
                      </div>
                    </div>
                  </div>
                }
                interactive={true}
                placement="bottom"
                trigger="mouseenter"
                delay={[200, 100]}
                className="z-50"
              >
                <BadgeButton 
                  color='zinc' 
                  href='#' 
                  className='!text-xs'
                >
                  <MaterialSymbol name="assignment" size="md"/> semaphore-project
                </BadgeButton>
              </Tippy>
              
              {/* Event Filters Display - Version 1 (Default) */}
              {eventFiltersVersion === 1 && (
                appliedFilters.length > 0 ? (
                <Tippy
                  content={
                    <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 min-w-[280px]">
                      <div className="space-y-3">
                        {appliedFilters.map((filter) => (
                          <div key={filter.id} className="flex items-center justify-between">
                            <div className="flex items-center gap-2">
                              <span className="text-sm text-zinc-700 dark:text-zinc-300">
                                {filter.type}
                              </span>
                            </div>
                            <div className="flex items-center gap-2 text-xs">
                              <span className="text-zinc-500 dark:text-zinc-400">
                                {filter.operator}
                              </span>
                              <span className="bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-800 dark:text-zinc-200 font-mono">
                                {filter.value}
                              </span>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  }
                  interactive={true}
                  placement="bottom"
                  trigger="mouseenter"
                  delay={[200, 100]}
                  className="z-50"
                >
                  <BadgeButton 
                    color='zinc' 
                    href='#' 
                    className='!text-xs'
                  >
                    <MaterialSymbol name="filter_list" size="md"/>
                    {appliedFilters.length} Event filters
                  </BadgeButton>
                </Tippy>
                ) : (
                  <BadgeButton 
                    color='zinc' 
                    href='#' 
                    className='!text-xs'
                  >
                    <MaterialSymbol name="podcasts" size="md"/>
                    All events
                  </BadgeButton>
                )
              )}

              {/* Event Filters Display - Version 2 (Enhanced Card Layout) */}
              {eventFiltersVersion === 2 && (
                <div className="flex flex-wrap gap-2 max-w-full">
                  {appliedFilters.map((filter) => (
                    <Tippy
                      key={filter.id}
                      content={
                        <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-xl shadow-lg p-4 min-w-[280px]">
                          <div className="flex items-start justify-between mb-3">
                            <div className="flex items-center gap-2">
                              <div className="w-8 h-8 bg-gradient-to-br from-blue-100 to-purple-100 dark:from-blue-900/40 dark:to-purple-900/40 rounded-lg flex items-center justify-center">
                                <MaterialSymbol name="filter_alt" size="sm" className="text-blue-600 dark:text-blue-400" />
                              </div>
                              <div>
                                <div className="text-sm font-semibold text-zinc-900 dark:text-white">
                                  {filter.type} Filter
                                </div>
                                <div className="text-xs text-zinc-500 dark:text-zinc-400">
                                  Active condition
                                </div>
                              </div>
                            </div>
                            <div className="flex items-center gap-1">
                              <button className="p-1 hover:bg-zinc-100 dark:hover:bg-zinc-700 rounded transition-colors">
                                <MaterialSymbol name="edit" size="sm" className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300" />
                              </button>
                              <button className="p-1 hover:bg-red-50 dark:hover:bg-red-900/20 rounded transition-colors">
                                <MaterialSymbol name="close" size="sm" className="text-zinc-400 hover:text-red-600 dark:hover:text-red-400" />
                              </button>
                            </div>
                          </div>
                          
                          <div className="space-y-2 mb-3">
                            <div className="flex items-center justify-between">
                              <span className="text-xs text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">Condition</span>
                              <div className="flex items-center gap-2">
                                <span className="text-xs font-mono bg-zinc-100 dark:bg-zinc-700 px-2 py-1 rounded text-zinc-700 dark:text-zinc-300">
                                  {filter.operator}
                                </span>
                                <span className="text-xs font-mono bg-blue-100 dark:bg-blue-900/40 px-2 py-1 rounded text-blue-700 dark:text-blue-300 font-semibold">
                                  "{filter.value}"
                                </span>
                              </div>
                            </div>
                            <div className="flex items-center justify-between">
                              <span className="text-xs text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">Status</span>
                              <div className="flex items-center gap-1">
                                <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                                <span className="text-xs text-green-600 dark:text-green-400 font-medium">Active</span>
                              </div>
                            </div>
                          </div>
                          
                          <div className="pt-2 border-t border-zinc-200 dark:border-zinc-700">
                            <div className="text-xs text-zinc-500 dark:text-zinc-400">
                              Events matching this filter will trigger the workflow
                            </div>
                          </div>
                        </div>
                      }
                      interactive={true}
                      placement="bottom"
                      trigger="mouseenter"
                      delay={[200, 100]}
                      className="z-50"
                      maxWidth={320}
                    >
                      <div className="group flex items-center gap-2 bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-900/20 dark:to-purple-900/20 hover:from-blue-100 hover:to-purple-100 dark:hover:from-blue-900/30 dark:hover:to-purple-900/30 text-blue-800 dark:text-blue-200 px-3 py-2 rounded-lg text-xs border border-blue-200/50 dark:border-blue-800/50 hover:border-blue-300 dark:hover:border-blue-700 transition-all duration-200 cursor-pointer shadow-sm hover:shadow-md min-w-0">
                        <div className="w-4 h-4 bg-blue-600 dark:bg-blue-400 rounded-full flex items-center justify-center flex-shrink-0">
                          <MaterialSymbol name="filter_alt" size={12} className="text-white dark:text-blue-900" />
                        </div>
                        <div className="flex items-center gap-1 min-w-0">
                          <span className="font-semibold truncate">{filter.type}</span>
                          <span className="text-blue-600 dark:text-blue-400 flex-shrink-0">:</span>
                          <span className="font-mono font-medium truncate max-w-20" title={filter.value}>
                            {filter.value.length > 8 ? `${filter.value.substring(0, 8)}...` : filter.value}
                          </span>
                        </div>
                        <MaterialSymbol 
                          name="info" 
                          size="sm" 
                          className="text-blue-500 dark:text-blue-400 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0" 
                        />
                      </div>
                    </Tippy>
                  ))}
                  
                  {/* Add Filter Button */}
                  <Tippy
                    content={
                      <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-3">
                        <div className="text-xs font-medium text-zinc-900 dark:text-white">
                          Add Event Filter
                        </div>
                        <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                          Filter events based on branches, types, or custom conditions
                        </div>
                      </div>
                    }
                    placement="bottom"
                    trigger="mouseenter"
                    delay={[200, 100]}
                  >
                    <button className="flex items-center gap-1 bg-zinc-100 hover:bg-zinc-200 dark:bg-zinc-800 dark:hover:bg-zinc-700 text-zinc-600 dark:text-zinc-400 px-3 py-2 rounded-lg text-xs border border-dashed border-zinc-300 dark:border-zinc-600 hover:border-zinc-400 dark:hover:border-zinc-500 transition-all duration-200">
                      <MaterialSymbol name="add" size="sm" />
                      <span className="font-medium">Add Filter</span>
                    </button>
                  </Tippy>
                </div>
              )}
              
            </div>
        </div>

        {/* Cluster Section */}
        {data.cluster && (
          <div className="px-4 py-3 pb-0 hidden">
            <div className="text-blue-600 dark:text-blue-400 font-medium">
              {data.cluster}
            </div>
          </div>
        )}

        {/* Events Section */}
        <div className="">
          
          
          {displayEvents.length > 0 ? (
            /* Events list with compact/expanded mode support */
            <div className={compactEvent ? "space-y-1 p-3" : "space-y-2 p-4"}>
              <div className='flex items-center justify-between w-full mb-3'>
                <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide">
                 LATEST EVENTS (3)
                </div>
                <Link href="#" className="text-xs text-zinc-600 dark:text-zinc-400 hover:text-zinc-800 dark:hover:text-zinc-300 transition-colors duration-200">
                View more
              </Link>
              </div>
              {displayEvents.map((event, index) => {
                const getStatusConfig = (status: string) => {
                  switch (status) {
                    case 'pending':
                      return {
                        icon: 'schedule',
                        color: 'text-yellow-600 dark:text-yellow-400',
                        bgColor: 'bg-yellow-50 dark:bg-yellow-900/20',
                        labelColor: 'yellow',
                        borderColor: 'border-yellow-200 dark:border-yellow-800',
                        dotColor: 'bg-yellow-500 animate-pulse',
                        label: 'Pending',
                        shortLabel: 'P',
                        description: 'Processing...'
                      };
                    case 'discarded':
                      return {
                        icon: 'block',
                        color: 'text-zinc-600 dark:text-zinc-400',
                        bgColor: 'bg-zinc-50 dark:bg-zinc-900/20',
                        labelColor: 'zinc',
                        borderColor: 'border-zinc-200 dark:border-zinc-800',
                        dotColor: 'bg-zinc-500',
                        label: 'Discarded',
                        shortLabel: 'D',
                        description: 'Filtered out'
                      };
                    case 'forwarded':
                      return {
                        icon: 'check_circle',
                        color: 'text-green-600 dark:text-green-400',
                        bgColor: 'bg-green-50 dark:bg-green-900/20',
                        labelColor: 'green',
                        borderColor: 'border-green-200 dark:border-green-800',
                        dotColor: 'bg-green-500',
                        label: 'Forwarded',
                        shortLabel: 'C',
                        description: event.processingTime ? `${event.processingTime}ms` : 'Completed'
                      };
                    default:
                      return {
                        icon: 'bolt',
                        color: 'text-zinc-600 dark:text-zinc-400',
                        bgColor: 'bg-zinc-50 dark:bg-zinc-800',
                        labelColor: 'zinc',
                        borderColor: 'border-zinc-200 dark:border-zinc-700',
                        dotColor: 'bg-zinc-500',
                        label: 'Unknown',
                        shortLabel: '?',
                        description: ''
                      };
                  }
                };

                const statusConfig = getStatusConfig(event.status);
                const timeAgo = compactEvent ? `${index * 1}m` : `${index * 1}m ago`;

                if (compactEvent) {
                  // Compact mode - single line with minimal info
                  return (
                    
                      <div className="flex items-center gap-2 p-2 bg-gray-50 dark:bg-zinc-800 rounded-md hover:bg-gray-100 dark:hover:bg-zinc-700 transition-colors duration-150 cursor-pointer">
                        {/* Compact Status Indicator */}
                        <div className="flex items-center gap-2">
                          <div className={`w-2 h-2 ${statusConfig.dotColor} rounded-full flex-shrink-0`}></div>
                          <span className={`text-xs font-medium hidden ${statusConfig.color}`}>
                            {statusConfig.label}
                          </span>
                        </div>
                        
                        {/* Event URL */}
                        <span className={event.status === 'discarded' ? 'text-sm font-mono text-gray-800 dark:text-zinc-200 truncate flex-1 line-through opacity-60' : "text-sm font-mono text-gray-800 dark:text-zinc-200 truncate flex-1"}>
                          {truncateUrl(event.url, 22)}
                        </span>
                        
                        <Tippy
                          key={event.id}
                          content={
                            <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-3 min-w-[260px]">
                              {event.timestamp}
                            </div>
                          }
                          placement="top"
                          trigger="mouseenter"
                          delay={[200, 100]}
                          className="z-50"
                        >    
                        {/* Time */}
                          <span className="text-xs text-zinc-500 dark:text-zinc-400 flex-shrink-0 w-6 text-right">
                            {timeAgo}
                          </span>
                        </Tippy>
                      </div>
                    
                  );
                } else {
                  // Expanded mode - multi-line with detailed status info
                  return (
                    <Tippy
                      key={event.id}
                      content={
                        <div className="bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded-lg shadow-lg p-4 min-w-[280px]">
                          <div className="flex items-start gap-3 mb-3">
                            <div className="flex-1">
                              <div className="text-sm font-semibold text-zinc-900 dark:text-white">
                                Event {statusConfig.label}
                              </div>
                              <div className="text-xs text-zinc-500 dark:text-zinc-400">
                                {timeAgo}
                              </div>
                            </div>
                            <div className={`w-2 h-2 ${statusConfig.dotColor} rounded-full mt-2`}></div>
                          </div>
                          
                          <div className="space-y-2 mb-3">
                            <div className="flex items-center justify-between text-xs">
                              <span className="text-zinc-500 dark:text-zinc-400">URL:</span>
                              <span className="font-mono text-zinc-700 dark:text-zinc-300 truncate max-w-48" title={event.url}>
                                {event.url}
                              </span>
                            </div>
                            <div className="flex items-center justify-between text-xs">
                              <span className="text-zinc-500 dark:text-zinc-400">Status:</span>
                              <span className={`${statusConfig.color} font-medium`}>
                                {statusConfig.label} - {statusConfig.description}
                              </span>
                            </div>
                            {event.type && (
                              <div className="flex items-center justify-between text-xs">
                                <span className="text-zinc-500 dark:text-zinc-400">Type:</span>
                                <span className="text-zinc-700 dark:text-zinc-300 font-mono">
                                  {event.type}
                                </span>
                              </div>
                            )}
                          </div>
                          
                          <div className="pt-2 border-t border-zinc-200 dark:border-zinc-700 text-xs text-zinc-500 dark:text-zinc-400">
                            {event.status === 'pending' && 'Event is being processed and will trigger workflows when complete'}
                            {event.status === 'discarded' && 'Event was filtered out and did not trigger any workflows'}
                            {event.status === 'forwarded' && 'Event successfully triggered workflow execution'}
                          </div>
                        </div>
                      }
                      interactive={true}
                      placement="bottom"
                      trigger="mouseenter"
                      delay={[200, 100]}
                      className="z-50"
                    >
                      <div className={`border bg-zinc-50 dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 rounded-md hover:shadow-sm transition-all duration-200 cursor-pointer`}>
                        {/* Main event row */}
                        <div className="flex items-center justify-between px-3 pt-2">
                          <div className="flex items-center gap-2">
                            <div className={`w-2 h-2 ${statusConfig.dotColor} rounded-full flex-shrink-0`}></div>
                            <span className={`text-xs font-medium ${statusConfig.color}`}>
                              {statusConfig.label}
                            </span>
                            <span className="text-xs text-zinc-500 dark:text-zinc-400">
                              • {statusConfig.description}
                            </span>
                           
                          </div>
                          <span className="text-xs text-zinc-500 dark:text-zinc-400 flex-shrink-0">
                            {timeAgo}
                          </span>
                        </div>
                        <div className="flex items-center gap-3 p-3">
                          
                          
                          {/* Event URL */}
                          <div className="flex-1 min-w-0">
                            <span className={event.status === 'discarded' ? 'text-sm font-mono text-gray-800 dark:text-zinc-200 truncate block line-through opacity-60' : "text-sm font-mono text-gray-800 dark:text-zinc-200 truncate block"}>
                              {truncateUrl(event.url, 35)}
                            </span>
                          </div>
                          
                          {/* Time */}
                          
                        </div>
                        
                        {/* Status information row */}
                        
                      </div>
                    </Tippy>
                  );
                }
              })}
              
            </div>
          ) : (
            /* noEvents variant - empty state */
            <div className="bg-zinc-50 dark:bg-zinc-800 px-4 rounded-lg">
              
              
              {/* Empty State */}
              <EmptyState
                size="xs"
                icon="sensors"
                animated={true}
                animationType="pulse"
                title='Ready to receive events'
                body={`Listening to changes in your Semaphore project`}
                className="pt-6 pb-4"
              />
            </div>
          )}
        </div>

        {/* React Flow Handles */}
        <Handle
          type="target"
          position={Position.Left}
          className="!w-1 !h-12 !bg-blue-500 dark:!bg-zinc-300 !border-none !border-white dark:!border-zinc-50 z-50 !rounded-md !hidden"
          aria-label="Input connection point"
        />
        <Handle
          type="source"
          position={Position.Right}
          className="!w-1 !h-12 !bg-blue-500 dark:!bg-zinc-300 !border-none !border-white dark:!border-zinc-50 z-50 !rounded-md"
          aria-label="Output connection point"
        />
        </div>
      </>
    );
  }

  // Edit Mode
  return (
    <>
      <div 
      className={clsx(
        'bg-white dark:bg-zinc-800 relative transition-all duration-200 hover:shadow-lg min-w-[320px]',
        
      )}
      style={{ width: 380, boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)' }}
      role="article"
    >
      {/* Edit Header */}
      <div 
          className="action-buttons absolute -top-14 left-1/2 transform -translate-x-1/2 flex gap-1 bg-white dark:bg-zinc-800 shadow-xs rounded-lg p-1 border border-gray-200 dark:border-zinc-600 z-50"
          onClick={(e) => e.stopPropagation()}
        >
         

            <Button
              type="button"
              plain
              className="flex items-center gap-2"
            >
              <MaterialSymbol name="code" size="md"/>
              Code
            </Button>
            
          
            
            
            <Dropdown>
              <DropdownButton plain className='flex items-center gap-2 !pr-1'>
                <MaterialSymbol name="save" size="md"/> 
                Save
                <MaterialSymbol name="expand_more" size="md"/>
              </DropdownButton>
              <DropdownMenu anchor="bottom start">
                <DropdownItem className='flex items-center gap-2' onClick={handleSaveAndCommit}><DropdownLabel>Save & Commit</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2' onClick={handleSave}><DropdownLabel>Save as Draft</DropdownLabel></DropdownItem>
               
              </DropdownMenu>
            </Dropdown>
           
            <Button
              type="button"
              plain
              onClick={handleCancel}
              className="flex items-center gap-2"
            >
              
              Cancel
            </Button>
            <Tippy content="More options" placement="top">
            <Dropdown>
              <DropdownButton plain>
                <MaterialSymbol name="more_vert" size="md"/>
              </DropdownButton>
              <DropdownMenu anchor="bottom start">
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="play_arrow" size="md"/><DropdownLabel>Run</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="tune" size="md"/><DropdownLabel>Advanced configuration</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2'><MaterialSymbol name="menu_book" size="md"/><DropdownLabel>Documentation</DropdownLabel></DropdownItem>
                <DropdownItem className='flex items-center gap-2 text-red-600 dark:text-red-200' color='red'><MaterialSymbol name="delete" size="md"/><DropdownLabel>Delete</DropdownLabel></DropdownItem>

              </DropdownMenu>
            </Dropdown>
          </Tippy>
         
          
          
        </div>

      {/* Edit Form */}
      <div>
        {/* Main content area with blue border */}
        <div className="border-2 border-blue-500 rounded-lg bg-white dark:bg-zinc-800">
          {/* Header with icon and title */}
          <div className='flex flex-col p-4'>
            <div className="flex items-center gap-2 mb-2">
              <div className="flex items-center">
              <img width="24" src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAMAAABF0y+mAAAAM1BMVEVHcEwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADbQS4qAAAAEXRSTlMAYq64jCpx/8oGF/mjNBDW6uM72ZcAAACJSURBVHgBzdBFAoAwEATBjbv8/7WwTHA50ziFhv6ekEpp80jWIR/uJt1W/LCbwpTV6a7ZcYV3vePq1QwOGu8n1sifJvb7Nm1EgVd8J6x0vWqlkBxU98XmkxlaxwM8jYzjxLwX+Gtr2hWGO1F1m8Ik0VWTtmMU6FR0aLe73g0FP8zSU0YrJQX9vAn47gbljcJgwwAAAABJRU5ErkJggg==" alt=""/>
              </div>
              
                <h2 className="text-md font-semibold text-gray-900 dark:text-white">
                  {data.title}
                </h2>
              
            
            </div>
            <span className='text-xs font-medium text-zinc-600 dark:text-zinc-400'>Description</span>
          </div>
          

          {/* Semaphore integration section */}
          <div className="space-y-3 border-t border-gray-200 dark:border-zinc-600 p-4">
            {inlineIntegration ? (
              /* Inline integration form */
              <div className="space-y-4 border border-gray-300 dark:border-zinc-600 rounded-lg p-4">
                <Field>
                  <Label className="text-sm font-medium text-gray-900 dark:text-white">
                    {data.icon === 'github' ? 'GitHub organization/owner URL' : 'Semaphore Organization URL'}
                  </Label>
                  <Input
                    type="url"
                    value={inlineIntegrationData.orgUrl}
                    onChange={(e) => setInlineIntegrationData(prev => ({ ...prev, orgUrl: e.target.value }))}
                    placeholder={data.icon === 'github' ? 'https://github.com/owner' : 'https://your-org.semaphoreci.com'}
                    className="w-full"
                  />
                </Field>
                
                <Field>
                  <Label className="text-sm font-medium text-gray-900 dark:text-white">
                    API Token
                  </Label>
                  <Input
                    type="password"
                    value={inlineIntegrationData.apiToken}
                    onChange={(e) => setInlineIntegrationData(prev => ({ ...prev, apiToken: e.target.value }))}
                    placeholder="Enter your API token"
                    className="w-full"
                  />
                </Field>
              </div>
            ) : availableIntegrations.length > 0 ? (
              /* Show integration dropdown */
              <div className="space-y-3">
                <Dropdown>
                  <DropdownButton outline className="flex items-center w-full !justify-between">
                    {selectedIntegration 
                      ? availableIntegrations.find(i => i.id === selectedIntegration)?.name || 'Select integration'
                      : (data.icon === 'github' ? 'Select GitHub integration' : 'Select Semaphore integration')
                    }
                    <MaterialSymbol name="keyboard_arrow_down" />
                  </DropdownButton>
                  <DropdownMenu>
                    {availableIntegrations.map((integration) => (
                      <DropdownItem
                        key={integration.id}
                        onClick={() => setSelectedIntegration(integration.id)}
                      >
                        <DropdownLabel>{integration.name}</DropdownLabel>
                      </DropdownItem>
                    ))}
                    <DropdownItem onClick={handleCreateIntegration}>
                      <div className="flex items-center gap-2 text-blue-600 dark:text-blue-400">
                        <MaterialSymbol name="add" size="sm" />
                        <span>Create new integration</span>
                      </div>
                    </DropdownItem>
                  </DropdownMenu>
                </Dropdown>
                <Field>
                  <Label className="text-sm font-medium text-gray-900 dark:text-white">{data.icon === 'github' ? 'Repository Name' : 'Semaphore Project'}</Label>
                  <Input  type="text" placeholder={data.icon === 'github' ? 'owner/repo' : 'Enter your Semaphore project name'} className="w-full" />
                </Field>
              </div>
            ) : (
              /* Empty state */
              <div className="text-center py-8 bg-zinc-50 dark:bg-zinc-700 border border-gray-200 dark:border-gray-700 rounded-md">
                <div className="text-gray-500 dark:text-zinc-400 mb-3">
                  {data.icon === 'github' 
                    ? "Looks like you haven't connected any GitHub accounts yet"
                    : "Looks like you haven't connected any Semaphore organizations yet"}
                </div>
                <Button
                  onClick={handleCreateIntegration}
                  color='blue'
                  className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md text-sm font-medium"
                >
                  Create integration
                </Button>
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

      {/* Integration Creation Modal */}
      <Dialog 
        open={showIntegrationModal} 
        onClose={() => setShowIntegrationModal(false)}
        className="relative z-50"
        size="md"
      >
        <DialogTitle>Create {data.icon === 'github' ? 'GitHub' : 'Semaphore'} Integration</DialogTitle>
        <DialogDescription>
          New integration will be saved to integrations page. Manage integrations  
          <Link href="/integrations" className='text-blue-600 dark:text-blue-400'> here</Link>.
        </DialogDescription>
        
        <DialogBody className="space-y-6">
          {/* Org/Repo URL */}
          <Field>
            <Label className="text-sm font-medium text-gray-900 dark:text-white">
              {data.icon === 'github' ? 'GitHub organization/owner URL' : 'Semaphore Org URL'}
            </Label>
            <Input
              type="url"
              value={integrationData.orgUrl}
              onChange={(e) => setIntegrationData(prev => ({ ...prev, orgUrl: e.target.value }))}
              placeholder={data.icon === 'github' ? 'https://github.com/owner' : 'https://your-org.semaphoreci.com'}
              className="w-full"
            />
          </Field>

          {/* Integration Name */}
          <Field>
            <Label className="text-sm font-medium text-gray-900 dark:text-white">
              Integration Name
            </Label>
            <Input
              type="text"
              value={integrationData.name}
              onChange={(e) => setIntegrationData(prev => ({ ...prev, name: e.target.value }))}
              placeholder="Enter integration name"
              className="w-full"
            />
          </Field>

          {data.icon === 'github' && (
            <div className="rounded-md border border-gray-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800 p-4">
              <div className="flex items-start gap-3">
                <div className="mt-0.5 text-zinc-600 dark:text-zinc-300">
                  <MaterialSymbol name="info" size="md" />
                </div>
                <div className="flex-1">
                  <div className="text-sm font-medium text-gray-900 dark:text-white">GitHub Personal Access Token (PAT) required</div>
                  <p className="text-sm text-zinc-700 dark:text-zinc-300 mt-1">
                    To connect GitHub, create a fine‑grained Personal Access Token and provide it as the API token.
                  </p>
                  <button
                    type="button"
                    className="mt-2 text-sm text-blue-600 dark:text-blue-300 hover:underline"
                    aria-expanded={showGitHubPatInfo}
                    onClick={() => setShowGitHubPatInfo(v => !v)}
                  >
                    {showGitHubPatInfo ? 'Hide details' : 'Show how to configure PAT'}
                  </button>
                  {showGitHubPatInfo && (
                    <div className="mt-3 space-y-2 text-sm text-zinc-700 dark:text-zinc-300">
                      <p>When creating a fine‑grained PAT</p>
                      <div><strong>Chose the access scope:</strong></div>
                      <ul className="list-disc ml-5 mt-1 space-y-1">
                        <li>All repositories</li>
                        <li>Or select specific repositories</li>
                      </ul>
                      <div className="mt-2"><strong>Set required permissions:</strong></div>
                      <ul className="list-disc ml-5 mt-1 space-y-1">
                        <li>Actions - Read AND Write</li>
                        <li>Webhooks - Read AND Write</li>
                      </ul>
                      <p className="text-xs text-zinc-600 dark:text-zinc-400">
                        Tip: You can manage or rotate the PAT anytime in your GitHub developer settings.
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

          {/* API Token Section */}
          <div className="space-y-4">
            <div className="text-sm font-medium text-gray-900 dark:text-white flex items-center justify-between">
              API Token
              <Link href={'#'} className='flex items-center gap-2 text-blue-600 dark:text-blue-200 hidden'>
                <MaterialSymbol name="add" size="md"/>Add new secret
              </Link>
            </div>
            
            {/* API Token Tabs */}
            <div>
              <ControlledTabs
                tabs={apiTokenTabs}
                activeTab={apiTokenTab}
                variant='pills'
                className='w-full'
                onTabChange={(tabId) => setApiTokenTab(tabId as 'existing' | 'new')}
              />
              
              <div className="pt-4">
                {apiTokenTab === 'existing' ? (
                  /* Select existing secret */
                  <div className="space-y-4">
                    <Field>
                      
                      <Dropdown>
                        <DropdownButton outline className='flex items-center w-full !justify-between'>
                          {integrationData.apiToken.secretName || 'Select secret'}
                          <MaterialSymbol name="keyboard_arrow_down" />
                        </DropdownButton>
                        <DropdownMenu anchor="bottom start">
                          {secrets.map((secret) => (
                            <DropdownItem
                              key={secret.id}
                              onClick={() => setIntegrationData(prev => ({
                                ...prev,
                                apiToken: { ...prev.apiToken, secretName: secret.name, secretKey: secret.keys[0]?.key || '' }
                              }))}
                            >
                              <DropdownLabel>{secret.name}</DropdownLabel>
                            </DropdownItem>
                          ))}
                        </DropdownMenu>
                      </Dropdown>
                    </Field>
                    {selectedExistingSecret && (
                      <Field className='flex items-start gap-3 w-full'>
                        <div className='w-50'>
                          <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
                            Key name
                          </Label>
                          <Input
                            type="text"
                            value={selectedExistingSecret.keys[0]?.key || ''}
                            readOnly
                            className="w-full bg-gray-50 dark:bg-zinc-800 cursor-default"
                          />
                        </div>
                        <div className='w-50'>
                          <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
                            Value
                          </Label>
                          <Input
                            type="password"
                            value={selectedExistingSecret.keys[0]?.value || ''}
                            readOnly
                            disabled
                            className="w-full bg-gray-50 dark:bg-zinc-800 cursor-not-allowed opacity-75"
                          />
                        </div>
                      </Field>
                    )}
                  </div>
                ) : (
                  /* Create new secret */
                  <div className="space-y-4 w-full">
                    <Text className='text-xs text-gray-500 dark:text-zinc-400'>
                      New secret will be created in your canvas secrets. 
                      You can review and manage your secrets in the secrets tab <Link href="#" className='text-blue-600 dark:text-blue-200'>here</Link>
                    </Text>

                    <Field className='flex items-start gap-3 w-full'>
                      <div className='w-50'>
                        <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
                          Key name
                        </Label>
                        <Input
                          type="text"
                          value="my-api-token"
                          className="w-full"
                        />
                      </div>
                      <div className='w-50'>
                        <Label className="text-sm font-medium text-gray-700 dark:text-zinc-300">
                          Value
                        </Label>
                        <Input
                          type="password"
                          value={newSecretToken}
                          onChange={(e) => setNewSecretToken(e.target.value)}
                          placeholder="Enter your API token"
                          className="w-full"
                        />
                      </div>
                    </Field>
                  </div>
                )}
              </div>
            </div>
          </div>
        </DialogBody>

        <DialogActions>
          <Button
            onClick={() => setShowIntegrationModal(false)}
          >
            Cancel
          </Button>
          <Button
            color='blue'
            onClick={handleSaveIntegration}
            disabled={!integrationData.name || !integrationData.orgUrl || !integrationData.apiToken.secretName || !integrationData.apiToken.secretKey}
          >
            Create
          </Button>
        </DialogActions>
      </Dialog>

      </div>

      
    </>
  );
}