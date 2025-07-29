import React, { useState } from 'react';
import { MaterialSymbol } from '../MaterialSymbol/material-symbol';
import { Button } from '../Button/button';
import { ControlledTabs, type Tab } from '../Tabs/tabs';
import { Text } from '../Text/text';
import { Subheading } from '../Heading/heading';
import { Link } from '../Link/link';
import { Badge } from '../Badge/badge';
import { Dropdown, DropdownButton, DropdownMenu, DropdownItem } from '../Dropdown/dropdown';
import clsx from 'clsx';
import { Divider } from '../Divider/divider';

interface RunData {
  id: string;
  name: string;
  status: 'success' | 'running' | 'failed' | 'pending';
  timestamp: string;
  duration: string;
  project?: string;
  pipeline?: string;
  inputs?: Record<string, string>;
  outputs?: Record<string, string>;
}

interface QueueItem {
  id: string;
  name: string;
  timestamp: string;
  status: 'pending' | 'approved' | 'waiting';
  approvalInfo?: {
    approvedBy?: number;
    waitingFor?: number;
  };
  inputs?: Record<string, string>;
  scheduledFor?: string;
}

interface NodeDetailsSidebarProps {
  nodeId?: string;
  nodeTitle?: string;
  nodeIcon?: string;
  isOpen?: boolean;
  onClose?: () => void;
  className?: string;
}

const mockRuns: RunData[] = [
  {
    id: 'run-2',
    name: 'Run #2',
    status: 'running',
    timestamp: 'Jan 16, 2022 10:23:45',
    duration: '00h 00m 25s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '1045a77',
      Image: 'v.1.2.1',
      Terraform: '32.32',
      Something: 'adsfasdf'
    }
  },
  {
    id: 'run-1',
    name: 'Run #1',
    status: 'success',
    timestamp: 'Jan 16, 2022 10:23:45',
    duration: '00h 00m 25s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '1045a77',
      Image: 'v.1.2.0',
      Terraform: '32.32',
      Something: 'adsfasdf'
    },
    outputs: {
      Code: '1045a77',
      Image: 'v.1.2.0',
      Terraform: '32.32',
      Something: 'adsfasdf'
    }
  }
];

const mockQueue: QueueItem[] = [
  {
    id: 'msg-1',
    name: 'Msg #2dlsf32fw',
    timestamp: 'Jan 16, 2022 10:23:45',
    status: 'approved',
    scheduledFor: 'Run next Monday',
    approvalInfo: {
      approvedBy: 1,
      waitingFor: 2
    },
    inputs: {
      Code: '1045a77',
      Image: 'v.1.2.3',
      Terraform: '32.32',
      Something: 'adsfasdf'
    }
  },
  {
    id: 'msg-2',
    name: 'Msg #2dlsf32fw',
    timestamp: '11 minutes ago',
    status: 'pending',
    inputs: {
      code: '1045a77',
      image: 'v.1.2.4'
    }
  }
];

export function NodeDetailsSidebar({
  nodeId,
  nodeTitle = 'Sync Cluster',
  nodeIcon = 'sync',
  isOpen = false,
  onClose,
  className
}: NodeDetailsSidebarProps) {
  const [activeTab, setActiveTab] = useState<'activity' | 'history' | 'settings'>('activity');
  const [expandedRuns, setExpandedRuns] = useState<Set<string>>(new Set());
  const [expandedQueue, setExpandedQueue] = useState<Set<string>>(new Set());

  const tabs: Tab[] = [
    { id: 'activity', label: 'Activity' },
    { id: 'history', label: 'History' },
    { id: 'settings', label: 'Settings' }
  ];
  const getStatusConfig = (status: string) => {
    switch (status?.toLowerCase()) {
      case 'success':
      case 'passed':
        return {
          bgColor: 'bg-green-50',
          borderColor: 'border-t border-t-green-400',
          textColor: 'text-green-700',
          icon: 'check_circle',
          iconColor: 'text-green-500',
        };
      case 'error':
      case 'failed':
        return {
          bgColor: 'bg-red-50',
          borderColor: 'border-t border-t-red-400',
          textColor: 'text-red-700',
          icon: 'cancel',
          iconColor: 'text-red-500',
        };
      case 'running':
        return {
          bgColor: 'bg-blue-50',
          borderColor: 'border-t border-t-blue-400',
          textColor: 'text-blue-700',
          icon: 'sync',
          iconColor: 'text-blue-500 animate-spin',
        };
      case 'pending':
      case 'queued':
        return {
          bgColor: 'bg-yellow-50',
          borderColor: 'border-t border-t-yellow-400',
          textColor: 'text-yellow-700',
          icon: 'schedule',
          iconColor: 'text-yellow-500',
        };
      default:
        return {
          bgColor: 'bg-gray-50',
          borderColor: 'border-t border-t-gray-400',
          textColor: 'text-gray-700',
          icon: 'help',
          iconColor: 'text-gray-500',
        };
    }
  }
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return { icon: 'check_circle', color: 'text-green-500' };
      case 'running':
        return { icon: 'sync', color: 'text-blue-500 animate-spin' };
      case 'failed':
        return { icon: 'error', color: 'text-red-500' };
      case 'pending':
        return { icon: 'schedule', color: 'text-orange-500' };
      default:
        return { icon: 'help', color: 'text-gray-500' };
    }
  };

  const toggleRunExpansion = (runId: string) => {
    setExpandedRuns(prev => {
      const newSet = new Set(prev);
      if (newSet.has(runId)) {
        newSet.delete(runId);
      } else {
        newSet.add(runId);
      }
      return newSet;
    });
  };

  const toggleQueueExpansion = (queueId: string) => {
    setExpandedQueue(prev => {
      const newSet = new Set(prev);
      if (newSet.has(queueId)) {
        newSet.delete(queueId);
      } else {
        newSet.add(queueId);
      }
      return newSet;
    });
  };

  const renderInputsOutputs = (inputs?: Record<string, string>, outputs?: Record<string, string>) => (
    <div className="mt-3 space-y-3">
      {inputs && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <MaterialSymbol name="input" size="sm" className="text-gray-500" />
            <Text className="text-xs font-semibold text-gray-700 uppercase tracking-wide">
              INPUTS
            </Text>
          </div>
          <div className="space-y-1 text-xs font-mono">
            {Object.entries(inputs).map(([key, value]) => (
              <div key={key} className="flex justify-between">
                <span className="text-gray-600">{key}</span>
                <span className="text-gray-900">{value}</span>
              </div>
            ))}
          </div>
        </div>
      )}
      
      {outputs && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <MaterialSymbol name="output" size="sm" className="text-gray-500" />
            <Text className="text-xs font-semibold text-gray-700 uppercase tracking-wide">
              OUTPUTS
            </Text>
          </div>
          <div className="space-y-1 text-xs font-mono">
            {Object.entries(outputs).map(([key, value]) => (
              <div key={key} className="flex justify-between">
                <span className="text-gray-600">{key}</span>
                <span className="text-gray-900">{value}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );

  if (!isOpen) return null;

  return (
    <div className={clsx(
      'absolute right-0 top-0 h-full w-96 bg-white border-l border-gray-200 shadow-lg z-50 flex flex-col',
      className
    )}>
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200">
        <div className="flex items-center gap-3">
          <MaterialSymbol name={nodeIcon} size="lg" className="text-gray-700" />
          <Subheading level={3} className="text-lg font-semibold text-gray-900">
            {nodeTitle}
          </Subheading>
        </div>
        <Button plain onClick={onClose}>
          <MaterialSymbol name="close" size="lg" className="text-gray-500" />
        </Button>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
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
            {/* Recent Runs */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <Text className="text-sm font-semibold text-gray-700 uppercase tracking-wide">
                  RECENT RUNS
                </Text>
                <Link href="#" className="text-sm text-blue-600 hover:text-blue-700">
                  View all
                </Link>
              </div>
              
              <div className="space-y-3">
                {mockRuns.map((run) => {
                  const statusConfig = getStatusIcon(run.status);
                  const statusConfig2 = getStatusConfig(run.status);
                  const isExpanded = expandedRuns.has(run.id);
                  
                  return (
                    <div key={run.id} className={statusConfig2.bgColor + " " + statusConfig2.borderColor } >
                      <div 
                        className="p-3"
                        
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                              
                              <MaterialSymbol 
                                name={statusConfig.icon} 
                                size="md" 
                                className={statusConfig.color}
                              />
                            <Text className="font-bold">{run.name}</Text>
                            <Dropdown>
                              <DropdownButton plain>
                                <MaterialSymbol name="more_vert" size="sm" />
                              </DropdownButton>
                              <DropdownMenu>
                                <DropdownItem>View details</DropdownItem>
                                <DropdownItem>Restart run</DropdownItem>
                                <DropdownItem>Download logs</DropdownItem>
                              </DropdownMenu>
                            </Dropdown>
                          </div>
                          <Button plain onClick={() => toggleRunExpansion(run.id)}>
                          <MaterialSymbol 
                              name={isExpanded ? 'expand_less' : 'expand_more'} 
                              size="lg" 
                              className="text-gray-400" 
                            />
                          </Button>
                        </div>
                        
                        {isExpanded && (
                          <div className="mt-3 px-6">
                            {run.project && (
                              <div className="flex items-center gap-2 mb-2">
                                <MaterialSymbol name="folder" size="sm" className="text-blue-500" />
                                <Link href="#" className="text-sm text-blue-600 hover:text-blue-700">
                                  {run.project}/{run.pipeline}
                                </Link>
                              </div>
                            )}
                            
                            <div className="flex items-center gap-4 text-xs text-gray-500 mb-3">
                              <div className="flex items-center gap-1">
                                <MaterialSymbol name="schedule" size="sm" />
                                <span>{run.timestamp}</span>
                              </div>
                              <div className="flex items-center gap-1">
                                <MaterialSymbol name="timer" size="sm" />
                                <span>{run.duration}</span>
                              </div>
                            </div>
                            
                            {renderInputsOutputs(run.inputs, run.outputs)}
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
                <Divider />
              </div>
            </div>

            {/* Queue */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <Text className="text-sm font-semibold text-gray-700 uppercase tracking-wide">
                  QUEUE ({mockQueue.length})
                </Text>
                <Link href="#" className="text-sm text-blue-600 hover:text-blue-700">
                  Manage queue
                </Link>
              </div>
              
              <div className="space-y-3">
                {mockQueue.map((item) => {
                  const isExpanded = expandedQueue.has(item.id);
                  
                  return (
                    <div key={item.id} className="border border-gray-200 rounded-lg">
                      <div 
                        className="p-3 cursor-pointer hover:bg-gray-50"
                        onClick={() => toggleQueueExpansion(item.id)}
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-3">
                            <MaterialSymbol 
                              name={isExpanded ? 'expand_less' : 'expand_more'} 
                              size="sm" 
                              className="text-gray-400" 
                            />
                            <div className="w-6 h-6 rounded-full bg-orange-100 flex items-center justify-center">
                              <MaterialSymbol name="schedule" size="sm" className="text-orange-500" />
                            </div>
                            <Text className="font-medium text-gray-900">{item.name}</Text>
                          </div>
                          <div className="flex items-center gap-2">
                            <Text className="text-xs text-gray-500">{item.timestamp}</Text>
                            <Dropdown>
                              <DropdownButton plain>
                                <MaterialSymbol name="more_vert" size="sm" />
                              </DropdownButton>
                              <DropdownMenu>
                                <DropdownItem>Approve</DropdownItem>
                                <DropdownItem>Reject</DropdownItem>
                                <DropdownItem>View details</DropdownItem>
                              </DropdownMenu>
                            </Dropdown>
                          </div>
                        </div>
                        
                        {isExpanded && (
                          <div className="mt-3 pl-9">
                            {renderInputsOutputs(item.inputs)}
                            
                            {item.scheduledFor && (
                              <div className="mt-3 flex items-center gap-2 text-xs text-gray-500">
                                <MaterialSymbol name="schedule" size="sm" />
                                <span>{item.scheduledFor}</span>
                              </div>
                            )}
                            
                            {item.approvalInfo && (
                              <div className="mt-3 flex items-center gap-2 text-xs text-gray-600">
                                <MaterialSymbol name="check_circle" size="sm" className="text-green-500" />
                                <span>
                                  approved by <Link href="#" className="text-blue-600">1 person</Link>, waiting for {item.approvalInfo.waitingFor} more
                                </span>
                                <Button plain className="ml-2">
                                  <MaterialSymbol name="check" size="sm" />
                                </Button>
                              </div>
                            )}
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        )}
        
        {activeTab === 'history' && (
          <div className="p-4">
            <Text className="text-gray-500">History view coming soon...</Text>
          </div>
        )}
        
        {activeTab === 'settings' && (
          <div className="p-4">
            <Text className="text-gray-500">Settings view coming soon...</Text>
          </div>
        )}
      </div>
    </div>
  );
}