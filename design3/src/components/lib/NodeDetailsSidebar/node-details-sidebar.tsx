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
  },
  {
    id: 'msg-3',
    name: 'Deploy #f8a9b3c',
    timestamp: '1 hour ago',
    status: 'waiting',
    scheduledFor: 'Run at 3:00 PM',
    approvalInfo: {
      approvedBy: 2,
      waitingFor: 1
    },
    inputs: {
      Environment: 'staging',
      Branch: 'feature/new-ui',
      Version: '2.1.0'
    }
  },
  {
    id: 'msg-4',
    name: 'Test #x7y2z9w',
    timestamp: '3 hours ago',
    status: 'approved',
    scheduledFor: 'Run immediately',
    inputs: {
      TestSuite: 'integration',
      Coverage: '85%',
      Browser: 'chrome'
    }
  },
  {
    id: 'msg-5',
    name: 'Build #k4m8n2p',
    timestamp: '5 hours ago',
    status: 'pending',
    inputs: {
      Docker: 'node:18-alpine',
      Memory: '2GB',
      CPU: '1 core'
    }
  },
  {
    id: 'msg-6',
    name: 'Release #q1w2e3r',
    timestamp: 'Yesterday 4:30 PM',
    status: 'approved',
    scheduledFor: 'Run tomorrow 9:00 AM',
    approvalInfo: {
      approvedBy: 3,
      waitingFor: 0
    },
    inputs: {
      Tag: 'v3.0.0',
      Changelog: 'Major release',
      Distribution: 'all-regions'
    }
  },
  {
    id: 'msg-7',
    name: 'Migrate #a5s6d7f',
    timestamp: '2 days ago',
    status: 'waiting',
    inputs: {
      Database: 'postgresql-14',
      Backup: 'enabled',
      Downtime: '15 minutes'
    }
  },
  {
    id: 'msg-8',
    name: 'Config #z9x8c7v',
    timestamp: '3 days ago',
    status: 'pending',
    scheduledFor: 'Manual trigger required',
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

  if (!isOpen) return null;

  return (
    <div className={clsx(
      'absolute right-0 top-0 h-full w-96 bg-white dark:bg-zinc-900 border-l border-gray-200 dark:border-zinc-700 shadow-lg z-50 flex flex-col',
      className
    )}>
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-zinc-700">
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
                    <div key={run.id} className={"border-b border-l border-r border-gray-200 dark:border-zinc-700 "+statusConfig.bgColor + " " + statusConfig.borderColor } >
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
                    <div key={item.id} className="" >
                      <div 
                        className="p-3 bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700"
                        onClick={() => toggleQueueExpansion(item.id)}
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                              
                              <MaterialSymbol 
                                name="input" 
                                size="lg" 
                                className="text-orange-600 dark:text-orange-400"
                              />
                            <span className="font-medium truncate text-sm dark:text-white">{item.name}</span>
                         
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
                            
                            {renderInputsOutputs2(item.inputs)}
                          
                           
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