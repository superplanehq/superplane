import React, { useState } from 'react';
import { MaterialSymbol } from '../MaterialSymbol/material-symbol';
import { getStatusConfig } from '../../../utils/status-config';
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
  icon: string;
  status: 'pending' | 'approved' | 'waiting';
  executionMethod: 'manual' | 'timed' | 'queued' | 'blocked';
  approvalInfo?: {
    approvedBy?: number;
    waitingFor?: number;
  };
  inputs?: Record<string, string>;
  scheduledFor?: string;
  delayTime?: string;
  blockedReason?: string;
}

interface NodeDetailsSidebarProps {
  nodeId?: string;
  nodeTitle?: string;
  nodeIcon?: string;
  isOpen?: boolean;
  onClose?: () => void;
  className?: string;
}

const mockRuns2: RunData[] = [
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
    id: 'run-332',
    name: 'Run #3',
    status: 'failed',
    timestamp: 'Jan 16, 2022 10:23:45',
    duration: '00h 10m 35s',
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
    name: 'laskdjf-a43re423-rfewlkjsdf234r-234234kl',
    timestamp: 'Jan 16, 2022 10:23:45',
    status: 'waiting',
    executionMethod: 'manual',
    icon: 'how_to_reg',
    scheduledFor: 'Pending approval',
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
    name: 'asdf324-asdf-rfewlkjsdf234r-sdf3244424',
    timestamp: '11 minutes ago',
    status: 'pending',
    executionMethod: 'timed',
    icon: 'schedule',
    delayTime: '15 minutes',
    scheduledFor: 'Run in 15 minutes',
    inputs: {
      code: '1045a77',
      image: 'v.1.2.4'
    }
  },
  {
    id: 'msg-3',
    name: 'asdf324-asdf-rfewlkjsdf234r-sdf3244424',
    timestamp: '1 hour ago',
    status: 'pending',
    executionMethod: 'queued',
    icon: 'input',
    scheduledFor: 'Position #2 in queue',
    inputs: {
      Environment: 'staging',
      Branch: 'feature/new-ui',
      Version: '2.1.0'
    }
  },
  {
    id: 'msg-4',
    name: 'sdfsdfdsf3324-asdf-rfewlkjsdf234r-sdf3244424',
    timestamp: '3 hours ago',
    status: 'waiting',
    executionMethod: 'blocked',
    icon: 'pause',
    scheduledFor: 'Blocked - dependency failed',
    blockedReason: 'Previous task "Build Infrastructure" failed',
    inputs: {
      TestSuite: 'integration',
      Coverage: '85%',
      Browser: 'chrome'
    }
  },
  {
    id: 'msg-5',
    name: 'asdf324-asdf-rfewlkjsdf234r-sdf3244424',
    timestamp: '5 hours ago',
    status: 'pending',
    executionMethod: 'queued',
    icon: 'input',
    scheduledFor: 'Position #5 in queue',
    inputs: {
      Docker: 'node:18-alpine',
      Memory: '2GB',
      CPU: '1 core'
    }
  },
  {
    id: 'msg-6',
    name: 'asdf324-asdf-rfewlkjsdf234r-sdf3244424',
    timestamp: 'Yesterday 4:30 PM',
    status: 'pending',
    executionMethod: 'timed',
    icon: 'schedule',
    scheduledFor: 'Run tomorrow 9:00 AM',
    delayTime: 'until 9:00 AM tomorrow',
    inputs: {
      Tag: 'v3.0.0',
      Changelog: 'Major release',
      Distribution: 'all-regions'
    }
  },
  {
    id: 'msg-7',
    name: 'asdf324-asdf-rfewlkjsdf234r-sdf3244424',
    timestamp: '2 days ago',
    status: 'waiting',
    executionMethod: 'blocked',
    icon: 'pause',
    scheduledFor: 'Blocked - resource unavailable',
    blockedReason: 'Database maintenance window not available',
    inputs: {
      Database: 'postgresql-14',
      Backup: 'enabled',
      Downtime: '15 minutes'
    }
  },
  {
    id: 'msg-8',
    name: 'asdf324-asdf-rfewlkjsdf234r-sdf3244424',
    timestamp: '3 days ago',
    status: 'waiting',
    executionMethod: 'manual',
    icon: 'how_to_reg',
    scheduledFor: 'Manual trigger required',
    approvalInfo: {
      approvedBy: 0,
      waitingFor: 1
    },
    inputs: {
      Environment: 'production',
      ConfigFile: 'app.config.json',
      Restart: 'required'
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
  // Check for showIcons URL parameter
  const showIcons = new URLSearchParams(window.location.search).get('showIcons') === 'true';
  
  const [activeTab, setActiveTab] = useState<'activity' | 'history' | 'settings'>('activity');
  const [expandedRuns, setExpandedRuns] = useState<Set<string>>(new Set());
  const [expandedQueue, setExpandedQueue] = useState<Set<string>>(new Set());

  const tabs: Tab[] = [
    { id: 'activity', label: 'Activity' },
    { id: 'history', label: 'History' },
    { id: 'settings', label: 'Settings' }
  ];
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
            <span  className="text-xs font-semibold text-gray-900 dark:text-gray-100 uppercase tracking-wide">
              INPUTS
            </span>
          </div>
          <div className="space-y-1 text-xs">
            {Object.entries(inputs).map(([key, value]) => (
              <div key={key} className="flex justify-between">
                <span className="text-gray-600">{key}</span>
                <Badge className="text-gray-900 font-mono !text-xs">{value}</Badge>
              </div>
            ))}
          </div>
        </div>
      )}
      
      {outputs && (
        <div>
          <div className="flex items-center gap-2 mb-2">
            <span  className="text-xs font-semibold text-gray-900 dark:text-gray-100 uppercase tracking-wide">
              OUTPUTS
            </span>
          </div>
          <div className="space-y-1 text-xs font-mono">
            {Object.entries(outputs).map(([key, value]) => (
              <div key={key} className="flex justify-between">
                <span className="text-gray-600">{key}</span>
                <Badge className="text-gray-900 font-mono !text-xs">{value}</Badge>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );

  const renderInputsOutputs2 = (inputs?: Record<string, string>, outputs?: Record<string, string>) => (
    <div className="mt-3 space-y-3">
      {inputs && (
       <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900">
       <div className="flex items-start gap-3">
         <div className="w-8 h-8 rounded-lg bg-zinc-900/10 dark:bg-zinc-700 flex items-center justify-center hidden">
           <MaterialSymbol name="input" size="md" className="text-gray-700 dark:text-zinc-400" />
         </div>
         <div className="flex-1">
           <span className="text-sm font-semibold mb-2 block dark:text-white">
             Inputs
           </span>
           <div className="space-y-1">
             {Object.entries(inputs || {}).map(([key, value]) => (
               <div key={key} className="flex items-center justify-between">
                 <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium">{key}</span>
                 <div className="flex items-center gap-2">
                   <Badge className='font-mono !text-xs'>
                     {value}
                   </Badge>
                 
                 </div>
               </div>
             ))}
           </div>
         </div>
       </div>
     </div>
      )}
      {outputs && (
        <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900">
          <div className="flex items-start gap-3">
            <div className="w-8 h-8 rounded-lg bg-zinc-900/10 dark:bg-zinc-700 flex items-center justify-center hidden">
              <MaterialSymbol name="output" size="md" className="text-gray-700 dark:text-zinc-400" />
            </div>
            <div className="flex-1">
              <span className="text-sm font-semibold mb-2 block dark:text-white">
                Outputs
              </span>
              <div className="space-y-1">
                {Object.entries(outputs).map(([key, value]) => (
                  <div key={key} className="flex items-center justify-between">
                    <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium">{key}</span>
                    <div className="flex items-center gap-2">
                      <Badge className='font-mono !text-xs'>
                        {value}
                      </Badge>
                    
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}
    
    </div>
  );

  const renderExecutionMethod = (item: QueueItem) => {
    const getExecutionMethodConfig = (method: string) => {
      switch (method) {
        case 'manual':
          return {
            icon: 'person',
            title: 'Manual Approval',
            description: 'Requires manual approval before execution',
            bgColor: 'bg-orange-50 dark:bg-orange-900/20',
            iconColor: 'text-orange-600 dark:text-orange-400',
            textColor: 'text-orange-800 dark:text-orange-300'
          };
        case 'timed':
          return {
            icon: 'schedule',
            title: 'Timed Execution',
            description: `Scheduled to run ${item.delayTime ? `in ${item.delayTime}` : 'at specified time'}`,
            bgColor: 'bg-blue-50 dark:bg-blue-900/20',
            iconColor: 'text-blue-600 dark:text-blue-400',
            textColor: 'text-blue-800 dark:text-blue-300'
          };
        case 'queued':
          return {
            icon: 'queue',
            title: 'Queue Execution',
            description: 'Waiting in queue for execution order',
            bgColor: 'bg-purple-50 dark:bg-purple-900/20',
            iconColor: 'text-purple-600 dark:text-purple-400',
            textColor: 'text-purple-800 dark:text-purple-300'
          };
        case 'blocked':
          return {
            icon: 'block',
            title: 'Blocked',
            description: item.blockedReason || 'Execution is blocked/paused',
            bgColor: 'bg-red-50 dark:bg-red-900/20',
            iconColor: 'text-red-600 dark:text-red-400',
            textColor: 'text-red-800 dark:text-red-300'
          };
        default:
          return {
            icon: 'help',
            title: 'Unknown',
            description: 'Execution method not specified',
            bgColor: 'bg-gray-50 dark:bg-gray-900/20',
            iconColor: 'text-gray-600 dark:text-gray-400',
            textColor: 'text-gray-800 dark:text-gray-300'
          };
      }
    };

    const config = getExecutionMethodConfig(item.executionMethod);
    
    return (
      item.executionMethod != 'queued' && (
        <div className={`p-3 border border-t-0 bg-orange-50 dark:bg-orange-900/20 border-zinc-200 dark:border-zinc-700`}>
          {item.executionMethod === 'manual' && (
            <div className='flex justify-between items-center'>
              <div className='flex items-center'>
                <MaterialSymbol name="how_to_reg" size="sm" className="text-gray-500 dark:text-zinc-200 mr-2" /> 
                <span className="text-xs text-gray-700 dark:text-zinc-400"><a href="#" className="black underline">1 person</a> approved, 2 more needed</span>
              </div>
              <Link href="#" className="text-xs text-gray-700 dark:text-zinc-300  flex items-center">
                <MaterialSymbol name="check" size="sm" className="text-gray-500 dark:text-zinc-400 mr-1" /> 
                <span className='underline'>Approve</span>
              </Link>
            </div>
          )}
          {item.executionMethod === 'timed' && (
            <div className='flex items-center'>
                <MaterialSymbol name={config.icon} size="sm" className="text-gray-500 dark:text-zinc-200 mr-2" /> 
                <span className='text-xs text-gray-700 dark:text-zinc-400'>{config.description}</span>
            </div>
          )}
          {item.executionMethod === 'blocked' && (
            <div className='flex items-center'>
                <MaterialSymbol name="pause" size="sm" className="text-gray-500 dark:text-zinc-200 mr-2" /> 
                <span className='text-xs text-gray-700 dark:text-zinc-400'>Freezed by <Link href="#" className="underline text-zinc-600 dark:text-zinc-400">1 person</Link></span>
            </div>
          )}
        
          
        </div>
      )
    );
  };

  if (!isOpen) return null;

  return (
    <div className={clsx(
      'absolute right-0 top-0 h-full w-96 bg-white dark:bg-zinc-900 border-l border-gray-200 dark:border-zinc-700 shadow-lg z-50 flex flex-col',
      className
    )}>
      {/* Header */}
      <div className="flex items-center justify-between p-4 pb-2">
        <div className="flex items-center gap-3">
          <MaterialSymbol name={nodeIcon} size="lg" className="text-gray-700 dark:text-zinc-300" />
          <Subheading level={3} className="text-lg font-semibold text-gray-900 dark:text-white">
            {nodeTitle}
          </Subheading>
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
            {/* Recent Runs */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <Text className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide">
                  RECENT RUNS
                </Text>
                <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                  View all
                </Link>
              </div>
              
              <div className="space-y-3">
               
                 {mockRuns2.map((run) => {
                  const statusConfig = getStatusConfig(run.status);
                  const isExpanded = expandedRuns.has(run.id);
                  
                  return (
                    <div key={run.id} className={"border-b border-l border-r border-gray-200 dark:border-zinc-700 cursor-pointer "+statusConfig.bgColor + " " + statusConfig.borderColor } >
                      <div 
                        className="p-3"
                        onClick={() => toggleRunExpansion(run.id)}
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                              
                              <MaterialSymbol 
                                name={statusConfig.icon} 
                                size="lg" 
                                className={statusConfig.iconColor}
                              />
                            <span className="font-bold truncate text-sm dark:text-white">{run.name}</span>
                         
                          </div>
                          <div className="flex items-center gap-3">
                            {!isExpanded && (
                              <span className="text-xs text-gray-500 dark:text-zinc-400">2 min ago</span>
                            )}
                            <MaterialSymbol 
                                name={isExpanded ? 'expand_less' : 'expand_more'} 
                                size="lg" 
                                className="text-gray-600 dark:text-zinc-400" 
                              />
                          </div>
                        </div>
                        
                        {isExpanded && (
                          <div className="mt-3 space-y-3">
                            
                            <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900">
                              <div className="flex items-start gap-3">
                                <div className="w-8 h-8 rounded-lg bg-zinc-900/10 dark:bg-zinc-700 flex items-center justify-center hidden">
                                  <MaterialSymbol name="timer" size="md" className="text-gray-700 dark:text-zinc-400" />
                                </div>
                                <div className="flex-1">
                                  <span className="text-sm font-semibold mb-2 block dark:text-white">
                                    Execution details
                                  </span>
                                  <div className="space-y-1 flex flex-col text-xs">
                                  <div className="flex items-center justify-between">
                                    <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium">Project</span>
                                    <div className="flex items-center gap-2 font-mono dark:text-zinc-300">
                                      Semaphore/project
                                    </div>
                                  </div>
                                  
                                  <div className="flex items-center justify-between">
                                    <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium">Started on</span>
                                    <div className="flex items-center gap-2 font-mono dark:text-zinc-300">
                                      {run.timestamp}
                                    </div>
                                  </div>
                                  <div className="flex items-center justify-between">
                                    <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium">Duration</span>
                                    <div className="flex items-center gap-2 font-mono dark:text-zinc-300">
                                      {run.duration}
                                    </div>
                                  </div>
                                  </div>
                                </div>
                              </div>
                            </div>
                          
                            {renderInputsOutputs2(run.inputs, run.outputs)}
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Queue */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <Text className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide">
                  QUEUE ({mockQueue.length})
                </Text>
                <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                  Manage queue
                </Link>
              </div>
              
              <div className="space-y-3">
              
                {mockQueue.map((item) => {
                  const isExpanded = expandedQueue.has(item.id);
                  
                  return (
                    <div key={item.id} className="queueItem" >
                      <div 
                        className="p-3 bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 cursor-pointer"
                        onClick={() => toggleQueueExpansion(item.id)}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex items-center gap-2 truncate">
                          { showIcons && (
                              <MaterialSymbol 
                                name={item.icon} 
                                size="lg" 
                                className="text-orange-600 dark:text-orange-400"
                              />
                            )}
                            <span className="font-medium truncate text-sm dark:text-white">{item.name}</span>
                          </div>
                          <div className="flex items-center gap-3">
                            {!isExpanded && (
                              <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">2 min ago</span>
                            )}
                            <MaterialSymbol 
                                name={isExpanded ? 'expand_less' : 'expand_more'} 
                                size="lg" 
                                className="text-gray-600 dark:text-zinc-400" 
                              />
                          </div>
                        </div>
                
                        {isExpanded && (
                          <div className="mt-3 space-y-3">
                            
                            {renderInputsOutputs2(item.inputs)}
                          
                           
                          </div>
                        )}
                      </div>
                      {item.executionMethod != 'queued' && (
                          <div className={`px-3 py-2 border border-t-0 bg-orange-50 dark:bg-orange-900/20 border-zinc-200 dark:border-zinc-700`}>
                          {item.executionMethod === 'manual' && (
                            <div className='flex justify-between items-center'>
                              <div className='flex items-center'>
                              { !showIcons && (
                                <MaterialSymbol name="how_to_reg" size="md" className="text-orange-700 dark:text-orange-200 mr-2" /> 
                              )}
                                <span className="text-xs text-gray-700 dark:text-zinc-400"><a href="#" className="black underline">1 person</a> approved, 2 more needed</span>
                              </div>
                              <Link href="#" className="text-xs text-gray-700 dark:text-zinc-300  flex items-center">
                                <MaterialSymbol name="check" size="sm" className="text-gray-500 dark:text-zinc-400 mr-1" /> 
                                <span className='underline'>Approve</span>
                              </Link>
                            </div>
                          )}
                          {item.executionMethod === 'timed' && (
                            <div className='flex items-center'>
                              { !showIcons && (
                                <MaterialSymbol name={item.icon} size="md" className="text-orange-700 dark:text-orange-200 mr-2" /> 
                              )}
                                <span className='text-xs text-gray-700 dark:text-zinc-400'>{item.scheduledFor}</span>
                            </div>
                          )}
                          {item.executionMethod === 'blocked' && (
                            <div className='flex items-center'>
                              { !showIcons && (
                                <MaterialSymbol name="pause" size="md" className="text-orange-700 dark:text-orange-200 mr-2" /> 
                              )}
                                <span className='text-xs text-gray-700 dark:text-zinc-400'>Freezed by <Link href="#" className="underline text-zinc-600 dark:text-zinc-400">1 person</Link></span>
                            </div>
                          )}
                        
                          
                        </div>
                      )}
                      
                    </div>
                  );
                })}
                           
              </div>
            </div>
          </div>
        )}
        
        {activeTab === 'history' && (
          <div className="p-4">
            <Text className="text-gray-500 dark:text-zinc-400">History view coming soon...</Text>
          </div>
        )}
        
        {activeTab === 'settings' && (
          <div className="p-4">
            <Text className="text-gray-500 dark:text-zinc-400">Settings view coming soon...</Text>
          </div>
        )}
      </div>
    </div>
  );
}