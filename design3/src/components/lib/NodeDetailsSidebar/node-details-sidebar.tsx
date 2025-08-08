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
import { EmptyState } from '../EmptyState';

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
    name: 'alsdkfjl43rlkewj-srlfksdj3r-sdlfkjwer',
    status: 'running',
    timestamp: 'Jan 16, 2022 10:23:45',
    duration: '00h 00m 25s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '34234234',
      Image: 'v.1.3.1',
      Terraform: '32.32'
    }
  },
  {
    id: 'run-332',
    name: '0234riefkjjsfgd-srlfksdj3r-sdlfkjwer',
    status: 'failed',
    timestamp: 'Jan 16, 2022 10:23:45',
    duration: '00h 10m 35s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '34324ew342523re',
      Image: 'v.2.3.1',
      Terraform: '32.32'
    }
  },
  {
    id: 'run-1',
    name: '23rsdf322ertf-srlfksdj3r-sdlfkjwer',
    status: 'success',
    timestamp: 'Jan 16, 2022 10:23:45',
    duration: '00h 00m 25s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '1045a77',
      Image: 'v.1.2.0',
      Terraform: '32.32'
    },
    outputs: {
      Code: '1045a77',
      Image: 'v.1.2.0',
      Terraform: '32.32'
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
      Terraform: '32.32'
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

const mockHistoryRuns: RunData[] = [
  {
    id: 'hist-1',
    name: 'kj3f9d8s2-wr5e7t1q4-mn6b8h2v',
    status: 'success',
    timestamp: 'Aug 1, 2025 14:23:12',
    duration: '00h 02m 45s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '8f23d91',
      Image: 'v.2.1.3',
      Environment: 'production'
    },
    outputs: {
      Code: '8f23d91',
      Image: 'v.2.1.3',
      DeploymentId: 'dep-8f23d91',
      Status: 'deployed'
    }
  },
  {
    id: 'hist-2',
    name: 'pl9x4k7m1-qr8t6w3e2-df5g9j1h',
    status: 'failed',
    timestamp: 'Aug 1, 2025 13:45:33',
    duration: '00h 01m 12s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '4d82f17',
      Image: 'v.1.9.2',
      TestSuite: 'integration'
    }
  },
  {
    id: 'hist-3',
    name: 'gh4j7k2l8-st9u6v3w1-cd2f5n7m',
    status: 'success',
    timestamp: 'Aug 1, 2025 12:18:47',
    duration: '00h 03m 21s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: 'a95c6e8',
      Image: 'v.2.0.1',
      Branch: 'feature/auth'
    },
    outputs: {
      Code: 'a95c6e8',
      Image: 'v.2.0.1',
      BuildId: 'build-a95c6e8',
      ArtifactUrl: 'https://artifacts.ex'
    }
  },
  {
    id: 'hist-4',
    name: 'xr8s5t2w9-mn4b7g3k6-qj1p8l5h',
    status: 'failed',
    timestamp: 'Aug 1, 2025 11:02:15',
    duration: '00h 00m 58s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '2e71b94',
      Image: 'v.1.8.5',
      Database: 'postgresql'
    }
  },
  {
    id: 'hist-5',
    name: 'vb3n6m9k2-ht5g8f1j4-pr7s4w6q',
    status: 'success',
    timestamp: 'Aug 1, 2025 10:34:28',
    duration: '00h 04m 17s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: 'f18c542',
      Image: 'v.2.2.0',
      Config: 'staging'
    },
    outputs: {
      Code: 'f18c542',
      Image: 'v.2.2.0',
      ServiceUrl: 'https://staging.example.com',
      HealthCheck: 'passed'
    }
  },
  {
    id: 'hist-6',
    name: 'fg2j5h8k1-dr4t7w9e3-mk6n3b5l',
    status: 'success',
    timestamp: 'Aug 1, 2025 09:56:44',
    duration: '00h 02m 33s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '6d9a3f7',
      Image: 'v.1.7.8',
      Memory: '2GB'
    },
    outputs: {
      Code: '6d9a3f7',
      Image: 'v.1.7.8',
      ContainerId: 'cnt-6d9a3f7',
      Port: '8080'
    }
  },
  {
    id: 'hist-7',
    name: 'ql8w1e4r7-zx6c9v2b5-nm3g6j9h',
    status: 'failed',
    timestamp: 'Aug 1, 2025 08:41:52',
    duration: '00h 01m 44s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '9c4e8b1',
      Image: 'v.1.6.3',
      Timeout: '30s'
    }
  },
  {
    id: 'hist-8',
    name: 'tr5y8u1i4-op2s6d9f3-gh7k4l8m',
    status: 'success',
    timestamp: 'Aug 1, 2025 07:29:16',
    duration: '00h 03m 07s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: 'b8f2d67',
      Image: 'v.2.3.1',
      Replicas: '3'
    },
    outputs: {
      Code: 'b8f2d67',
      Image: 'v.2.3.1',
      LoadBalancer: 'lb-b8f2d67',
      Replicas: '3'
    }
  },
  {
    id: 'hist-9',
    name: 'bn4m7v2c8-wq9e3r6t1-xz5s8f2j',
    status: 'failed',
    timestamp: 'Jul 31, 2025 23:15:39',
    duration: '00h 02m 23s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '7a3c9e5',
      Image: 'v.1.5.2',
      Region: 'us-west-2'
    }
  },
  {
    id: 'hist-10',
    name: 'lk6p9o2i5-df3g6h9j4-qw8e1r7t',
    status: 'success',
    timestamp: 'Jul 31, 2025 22:47:08',
    duration: '00h 05m 12s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: 'e5f8c92',
      Image: 'v.2.4.0',
      Volume: '10GB'
    },
    outputs: {
      Code: 'e5f8c92',
      Image: 'v.2.4.0',
      VolumeId: 'vol-e5f8c92',
      MountPath: '/data'
    }
  },
  {
    id: 'hist-11',
    name: 'mx8z3c6v1-hy4j7k2l9-rt6u3i8o',
    status: 'success',
    timestamp: 'Jul 31, 2025 21:33:54',
    duration: '00h 02m 48s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '3d7b4f8',
      Image: 'v.1.9.7',
      Cache: 'enabled'
    },
    outputs: {
      Code: '3d7b4f8',
      Image: 'v.1.9.7',
      CacheKey: 'cache-3d7b4f8',
      Size: '2.3GB'
    }
  },
  {
    id: 'hist-12',
    name: 'pq2w5e8r1-sd4f7g9h6-jk3l6m2n',
    status: 'failed',
    timestamp: 'Jul 31, 2025 20:19:27',
    duration: '00h 01m 35s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '1f9e6a4',
      Image: 'v.1.4.1',
      Network: 'bridge'
    }
  },
  {
    id: 'hist-13',
    name: 'cv7b4n1m8-xl5z2a9s6-qw3e7r1t',
    status: 'success',
    timestamp: 'Jul 31, 2025 19:02:43',
    duration: '00h 03m 56s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '8c2e5d9',
      Image: 'v.2.1.5',
      SSL: 'enabled'
    },
    outputs: {
      Code: '8c2e5d9',
      Image: 'v.2.1.5',
      Certificate: 'cert-8c2e5d9',
      Domain: 'api.example.com'
    }
  },
  {
    id: 'hist-14',
    name: 'hg9f6d3s2-jk4l7m1n8-po5i2u7y',
    status: 'failed',
    timestamp: 'Jul 31, 2025 18:45:19',
    duration: '00h 00m 42s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '4a1f3c7',
      Image: 'v.1.3.9',
      Backup: 'daily'
    }
  },
  {
    id: 'hist-15',
    name: 'rt3y6u9i2-as5df8gh1-zx4cv7bn',
    status: 'success',
    timestamp: 'Jul 31, 2025 17:38:05',
    duration: '00h 04m 29s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    inputs: {
      Code: '6e9b2f5',
      Image: 'v.2.0.8',
      Workers: '5'
    },
    outputs: {
      Code: '6e9b2f5',
      Image: 'v.2.0.8',
      QueueId: 'queue-6e9b2f5',
      Workers: '5'
    }
  }
];

export function NodeDetailsSidebar({
  nodeId,
  nodeTitle = 'Sync Cluster',
  nodeIcon = 'semaphore',
  isOpen = false,
  onClose,
  className
}: NodeDetailsSidebarProps) {
  // Check for showIcons URL parameter
  const showIcons = new URLSearchParams(window.location.search).get('showIcons') === 'true';
  
  const [activeTab, setActiveTab] = useState<'activity' | 'history' | 'settings'>('activity');
  const [expandedRuns, setExpandedRuns] = useState<Set<string>>(new Set());
  const [expandedQueue, setExpandedQueue] = useState<Set<string>>(new Set());
  const [expandedHistoryRuns, setExpandedHistoryRuns] = useState<Set<string>>(new Set());
  const [isManagingQueue, setIsManagingQueue] = useState(false);
  const [queueItems, setQueueItems] = useState(mockQueue);

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

  const toggleHistoryRunExpansion = (runId: string) => {
    setExpandedHistoryRuns(prev => {
      const newSet = new Set(prev);
      if (newSet.has(runId)) {
        newSet.delete(runId);
      } else {
        newSet.add(runId);
      }
      return newSet;
    });
  };

  const handleManageQueue = () => {
    setIsManagingQueue(true);
  };

  const handleSaveQueue = () => {
    setIsManagingQueue(false);
    // Here you would typically save the queue order to your backend
  };

  const handleCancelQueue = () => {
    setIsManagingQueue(false);
    setQueueItems(mockQueue); // Reset to original order
  };

  const handleRemoveQueueItem = (id: string) => {
    setQueueItems(prev => prev.filter(item => item.id !== id));
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
         <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Inputs</div>
           <div className="space-y-1">
             {Object.entries(inputs || {}).map(([key, value]) => (
               <div key={key} className="flex items-center justify-between">
                 <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium">{key}</span>
                 <div className="flex items-center gap-2 truncate">
                   <Badge className='font-mono !text-xs truncate'>
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
              <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Outputs</div>
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
        <div className='rounded-lg dark:bg-white !p-1'>
          <img width={24} src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAMAAABF0y+mAAAAM1BMVEVHcEwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADbQS4qAAAAEXRSTlMAYq64jCpx/8oGF/mjNBDW6uM72ZcAAACJSURBVHgBzdBFAoAwEATBjbv8/7WwTHA50ziFhv6ekEpp80jWIR/uJt1W/LCbwpTV6a7ZcYV3vePq1QwOGu8n1sifJvb7Nm1EgVd8J6x0vWqlkBxU98XmkxlaxwM8jYzjxLwX+Gtr2hWGO1F1m8Ik0VWTtmMU6FR0aLe73g0FP8zSU0YrJQX9vAn47gbljcJgwwAAAABJRU5ErkJggg==" alt="" />
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
            {/* Recent Runs */}
            <div className="flex items-center justify-between mb-4">
              <Text className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide">
                RECENT RUNS
              </Text>
              
            </div>
            <EmptyState size='sm' className='bg-gray-50 dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700' title="No recent runs" body="No recent runs"/>
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
                          <div className="flex items-center gap-2 truncate pr-2">
                              
                              <MaterialSymbol 
                                name={statusConfig.icon} 
                                size="lg" 
                                className={statusConfig.iconColor}
                              />
                            <span className="font-medium truncate text-sm dark:text-white">{run.name}</span>
                         
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
                            <div className="grid grid-cols-2 gap-4 text-xs p-4 rounded-md bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700">
                              <div className="col-span-2">
                                <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Label</div>
                                <Link href="#" className="font-medium text-blue-600 dark:text-blue-400">https://semaphoreci.com</Link>
                              </div>
                              <div>
                                <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Project</div>
                                <Link href="#" className="font-medium text-blue-600 dark:text-blue-400">semaphore-project</Link>
                              </div>
                              <div>
                                <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Pipeline</div>
                                <Link href="#" className="font-medium text-blue-600 dark:text-blue-400">.semaphore.yml</Link>
                              </div>
                              <div>
                                <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Duration</div>
                                <div className="font-medium text-gray-900 dark:text-zinc-300 font-mono">00h 00m 25s</div>
                              </div>
                              <div>
                                <div className="text-xs text-gray-700 dark:text-zinc-400 uppercase tracking-wide mb-1 font-bold">Started on</div>
                                <div className="font-medium text-gray-900 dark:text-zinc-300">Jan 16 2022 10:23:45</div>
                              </div>
                            </div>
                            <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-3 bg-white dark:bg-zinc-900 hidden">
                              <div className="flex items-start gap-3">
                                <div className="w-8 h-8 rounded-lg bg-zinc-900/10 dark:bg-zinc-700 flex items-center justify-center hidden">
                                  <MaterialSymbol name="timer" size="md" className="text-gray-700 dark:text-zinc-400" />
                                </div>
                                
                                <div className="flex-1">
                                  <div className="text-xs text-gray-600 dark:text-zinc-400 uppercase tracking-wide mb-1">Execution Details</div>
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
                  QUEUE ({queueItems.length})
                </Text>
                {!isManagingQueue ? (
                  <Link 
                    href="#" 
                    onClick={(e) => { e.preventDefault(); handleManageQueue(); }}
                    className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300"
                  >
                    Manage queue
                  </Link>
                ) : (
                  <div className="flex items-center gap-2">
                    <Button 
                      onClick={handleSaveQueue}
                      plain
                    >
                      Save
                    </Button>
                    <Button 
                      onClick={handleCancelQueue}
                      plain
                    >
                      Cancel
                    </Button>
                  </div>
                )}
              </div>
              
              <div className="space-y-3">
              
                {queueItems.map((item) => {
                  const isExpanded = expandedQueue.has(item.id);
                  
                  return (
                    <div key={item.id} className="queueItem" >
                      <div 
                        className={`p-3 bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 ${!isManagingQueue ? 'cursor-pointer' : ''}`}
                        onClick={!isManagingQueue ? () => toggleQueueExpansion(item.id) : undefined}
                      >
                        <div className="flex items-center justify-between gap-2">
                          <div className="flex items-center gap-2 truncate">
                            {isManagingQueue && (
                              <MaterialSymbol 
                                name="drag_indicator" 
                                size="md" 
                                className="text-gray-400 dark:text-zinc-500 cursor-grab active:cursor-grabbing"
                              />
                            )}
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
                            {!isManagingQueue && !isExpanded && (
                              <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">2 min ago</span>
                            )}
                            {isManagingQueue ? (
                              <div className="flex items-center gap-2">
                                
                                <Dropdown>
                                  <DropdownButton as={Button} plain className="p-2 rounded hover:bg-gray-100 dark:hover:bg-zinc-700">
                                    <MaterialSymbol 
                                      name="more_vert" 
                                      size="md" 
                                      className="text-gray-600 dark:text-zinc-400"
                                    />
                                  </DropdownButton>
                                  <DropdownMenu className="w-48">
                                    <DropdownItem onClick={() => handleRemoveQueueItem(item.id)}>
                                      <MaterialSymbol name="delete" size="sm" className="mr-2" />
                                      Remove from queue
                                    </DropdownItem>
                                  </DropdownMenu>
                                </Dropdown>
                              </div>
                            ) : (
                              <MaterialSymbol 
                                name={isExpanded ? 'expand_less' : 'expand_more'} 
                                size="lg" 
                                className="text-gray-600 dark:text-zinc-400" 
                              />
                            )}
                          </div>
                        </div>
                
                        {!isManagingQueue && isExpanded && (
                          <div className="mt-3 space-y-3">
                            
                            {renderInputsOutputs2(item.inputs)}
                          
                           
                          </div>
                        )}
                      </div>
                      {!isManagingQueue && item.executionMethod != 'queued' && (
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
          <div className="p-4 space-y-6">
            {/* Historical Runs */}
            <div>
              <div className="flex items-center justify-between mb-4">
                <Text className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide">
                  HISTORICAL RUNS ({mockHistoryRuns.length})
                </Text>
              </div>
              
              <div className="space-y-3">
                {mockHistoryRuns.map((run) => {
                  const statusConfig = getStatusConfig(run.status);
                  const isExpanded = expandedHistoryRuns.has(run.id);
                  
                  return (
                    <div key={run.id} className={"border-b border-l border-r border-gray-200 dark:border-zinc-700 cursor-pointer "+statusConfig.bgColor + " " + statusConfig.borderColor } >
                      <div 
                        className="p-3"
                        onClick={() => toggleHistoryRunExpansion(run.id)}
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2 truncate pr-2">
                              
                              <MaterialSymbol 
                                name={statusConfig.icon} 
                                size="lg" 
                                className={statusConfig.iconColor}
                              />
                            <span className="font-medium truncate text-sm dark:text-white">{run.name}</span>
                         
                          </div>
                          <div className="flex items-center gap-3">
                            {!isExpanded && (
                              <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">
                                {run.timestamp.includes('Aug 1, 2025') ? 
                                  run.timestamp.split(' ')[3] : 
                                  run.timestamp.includes('Jul 31, 2025') ? 'Yesterday' : run.timestamp}
                              </span>
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
                                <div className="text-xs text-gray-600 dark:text-zinc-400 uppercase tracking-wide mb-1">Execution Details</div>

                                  <div className="space-y-1 flex flex-col text-xs">
                                  <div className="flex items-center justify-between">
                                    <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium">Project</span>
                                    <div className="flex items-center gap-2 font-mono dark:text-zinc-300">
                                      {run.project}
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