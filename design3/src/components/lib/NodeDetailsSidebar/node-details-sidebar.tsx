import { useState } from 'react';
import { MaterialSymbol } from '../MaterialSymbol/material-symbol';
import { getStatusConfig } from '../../../utils/status-config';
import { Button } from '../Button/button';
import { ControlledTabs, type Tab } from '../Tabs/tabs';
import { Text } from '../Text/text';
import { Link } from '../Link/link';
import { Badge, BadgeButton } from '../Badge/badge';
import { Dropdown, DropdownButton, DropdownMenu, DropdownItem } from '../Dropdown/dropdown';
import clsx from 'clsx';
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
  queuedAt?: string;
  conditionMetAt?: string;
  approvedBy?: string;
  branchTagPr?: string;
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

interface EventData {
  id: string;
  url: string;
  status: 'pending' | 'discarded' | 'forwarded';
  timestamp: string;
  processingTime?: number;
  type?: string;
  payload?: Record<string, any>;
  headers?: Record<string, string>;
}

interface NodeDetailsSidebarProps {
  nodeId?: string;
  nodeTitle?: string;
  nodeIcon?: string;
  isOpen?: boolean;
  onClose?: () => void;
  className?: string;
  source?: 'workflow' | 'eventSource';
  eventSourceType?: 'semaphore' | 'webhook' | 'http';
  onNodeUpdate?: (nodeId: string, updates: { icon?: string; eventSourceType?: string }) => void;
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

const mockRuns2: RunData[] = [
  {
    id: 'run-2',
    name: 'alsdkfjl43rlkewj-srlfksdj3r-sdlfkjwer',
    status: 'running',
    timestamp: 'Jan 16, 2022 10:23:45',
    duration: '00h 00m 25s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    queuedAt: 'Jan 16, 2022 10:22:15',
    conditionMetAt: 'Jan 16, 2022 10:23:30',
    approvedBy: 'john.doe@example.com',
    branchTagPr: 'feature/auth-improvements',
    inputs: {
      Code: '34234234',
      Image: 'v.1.3.1',
      Terraform: '32.32'
    }
  },
  {
    id: 'run-332',
    name: 'Install Semaphore update',
    status: 'failed',
    timestamp: 'Jan 16, 2022 10:23:45',
    duration: '00h 10m 35s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    queuedAt: 'Jan 16, 2022 10:12:20',
    conditionMetAt: 'Jan 16, 2022 10:23:40',
    approvedBy: 'admin@example.com',
    branchTagPr: 'v2.1.0',
    inputs: {
      Code: '34324ew342523re',
      Image: 'v.2.3.1',
      Terraform: '32.32'
    }
  },
  {
    id: 'run-1',
    name: 'Install Semaphore update',
    status: 'success',
    timestamp: 'Jan 16, 2022 10:23:45',
    duration: '00h 00m 25s',
    project: 'Semaphore project',
    pipeline: 'Pipeline name',
    queuedAt: 'Jan 16, 2022 10:20:10',
    conditionMetAt: 'Jan 16, 2022 10:23:35',
    approvedBy: 'alice.wilson@example.com',
    branchTagPr: 'PR #142',
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

// Mock event data for EventSource nodes
const mockEventSourceEvents: EventData[] = [
  {
    id: 'event-1',
    url: 'https://hooks.semaphoreci.com/api/v1/webhooks/b8f2c4d1-9e3a-4f7b-8c1d-5e9a3f7b2c4d',
    status: 'forwarded',
    timestamp: new Date(Date.now() - 5 * 60 * 1000).toISOString(),
    type: 'pipeline_done',
    processingTime: 156,
    payload: {
      pipeline_id: 'pipeline_123',
      status: 'passed',
      branch: 'main',
      commit: 'a1b2c3d4e5f6',
      author: 'john.doe@example.com'
    },
    headers: {
      'x-semaphore-event': 'pipeline_done',
      'content-type': 'application/json',
      'user-agent': 'Semaphore-Webhook/1.0'
    }
  },
  {
    id: 'event-2',
    url: 'https://github.com/zawkey/superplane/push',
    status: 'pending',
    timestamp: new Date(Date.now() - 12 * 60 * 1000).toISOString(),
    type: 'push',
    payload: {
      ref: 'refs/heads/feature/auth',
      before: 'd6fde92930d4715a2b49857d24b940956b26d243',
      after: '6dcb09b5b57875f334f61aebed695e2e4193db5e',
      repository: {
        name: 'superplane',
        full_name: 'zawkey/superplane'
      }
    },
    headers: {
      'x-github-event': 'push',
      'x-github-delivery': 'f2ca3bc6-e8ea-11e3-ac10-0800200c9a66',
      'user-agent': 'GitHub-Hookshot/760256b'
    }
  },
  {
    id: 'event-3',
    url: 'https://gitlab.com/api/v4/projects/42/hooks/webhook',
    status: 'discarded',
    timestamp: new Date(Date.now() - 18 * 60 * 1000).toISOString(),
    type: 'merge_request',
    payload: {
      object_kind: 'merge_request',
      action: 'open',
      merge_request: {
        id: 99,
        title: 'MS-viewport meta tag',
        source_branch: 'ms-viewport',
        target_branch: 'master'
      }
    },
    headers: {
      'x-gitlab-event': 'Merge Request Hook',
      'x-gitlab-token': 'secret-token',
      'content-type': 'application/json'
    }
  },
  {
    id: 'event-4',
    url: 'https://hooks.semaphoreci.com/api/v1/webhooks/c9a3e5f7-1b4d-4a8c-9f2e-6d7b4c3e1a5f',
    status: 'forwarded',
    timestamp: new Date(Date.now() - 28 * 60 * 1000).toISOString(),
    type: 'deployment_done',
    processingTime: 234,
    payload: {
      deployment_id: 'dep_456',
      environment: 'production',
      status: 'passed',
      commit: 'f8e7d6c5b4a3',
      deployed_by: 'alice.wilson@example.com'
    },
    headers: {
      'x-semaphore-event': 'deployment_done',
      'content-type': 'application/json',
      'user-agent': 'Semaphore-Webhook/1.0'
    }
  },
  {
    id: 'event-5',
    url: 'https://bitbucket.org/zawkey/superplane/webhooks/repo:push',
    status: 'pending',
    timestamp: new Date(Date.now() - 35 * 60 * 1000).toISOString(),
    type: 'repo:push',
    payload: {
      repository: {
        name: 'superplane',
        full_name: 'zawkey/superplane'
      },
      push: {
        changes: [{
          new: {
            name: 'feature/ui-improvements',
            target: {
              hash: 'b8f2c4d19e3a'
            }
          }
        }]
      }
    },
    headers: {
      'x-event-key': 'repo:push',
      'x-request-uuid': 'b8f2c4d1-9e3a-4f7b-8c1d-5e9a3f7b2c4d',
      'user-agent': 'Bitbucket-Webhooks/2.0'
    }
  },
  {
    id: 'event-6',
    url: 'https://jenkins.internal.com/webhook/build-complete',
    status: 'discarded',
    timestamp: new Date(Date.now() - 42 * 60 * 1000).toISOString(),
    type: 'build_complete',
    payload: {
      job_name: 'superplane-ci',
      build_number: 128,
      result: 'FAILURE',
      branch: 'develop',
      duration: 180000
    },
    headers: {
      'x-jenkins-event': 'build_complete',
      'content-type': 'application/json',
      'x-jenkins-cli-port': '50000'
    }
  }
];

// Mock historical event data for EventSource nodes
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
    headers: {
      'x-github-event': 'push',
      'user-agent': 'GitHub-Hookshot'
    }
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
    headers: {
      'x-github-event': 'release',
      'user-agent': 'GitHub-Hookshot'
    }
  },
  {
    id: 'hist-event-3',
    url: 'https://github.com/owner/repo/issues',
    status: 'discarded',
    timestamp: new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString(),
    type: 'issues',
    payload: {
      action: 'opened',
      number: 123,
      title: 'Bug report'
    },
    headers: {
      'x-github-event': 'issues',
      'user-agent': 'GitHub-Hookshot'
    }
  }
];

export function NodeDetailsSidebar({
  nodeId,
  nodeTitle = 'Sync Cluster',
  nodeIcon,
  isOpen = false,
  onClose,
  className,
  source = 'workflow',
  eventSourceType = 'semaphore',
  onNodeUpdate,
  events = []
}: NodeDetailsSidebarProps) {
  // Check for showIcons URL parameter
  const showIcons = new URLSearchParams(window.location.search).get('showIcons') === 'true';
  
  const [activeTab, setActiveTab] = useState<'activity' | 'history' | 'settings'>('activity');
  const [expandedRuns, setExpandedRuns] = useState<Set<string>>(new Set());
  const [expandedQueue, setExpandedQueue] = useState<Set<string>>(new Set());
  const [expandedHistoryRuns, setExpandedHistoryRuns] = useState<Set<string>>(new Set());
  const [isManagingQueue, setIsManagingQueue] = useState(false);
  const [queueItems, setQueueItems] = useState(mockQueue);
  
  // Event Source specific state  
  const [expandedEvents, setExpandedEvents] = useState<Set<string>>(new Set());
  const [expandedHistoryEvents, setExpandedHistoryEvents] = useState<Set<string>>(new Set());

  const tabs: Tab[] = [
    { id: 'activity', label: 'Activity' },
    { id: 'history', label: 'History' },
    { id: 'settings', label: 'Settings' }
  ];

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

  // Event Source specific functions
  const getEventStatusConfig = (status: string) => {
    switch (status) {
      case 'pending':
        return {
          icon: 'schedule',
          color: 'text-yellow-700 dark:text-yellow-400',
          bgColor: 'bg-yellow-50 dark:bg-yellow-900/20',
          labelColor: 'text-yellow-700 dark:text-yellow-400',
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
          labelColor: 'text-zinc-600 dark:text-zinc-400',
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
          labelColor: 'text-green-600 dark:text-green-400',
          borderColor: 'border-green-200 dark:border-green-800',
          dotColor: 'bg-green-500',
          label: 'Forwarded',
          shortLabel: 'C',
          description: 'Completed'
        };
      default:
        return {
          icon: 'bolt',
          color: 'text-zinc-600 dark:text-zinc-400',
          bgColor: 'bg-zinc-50 dark:bg-zinc-800',
          labelColor: 'text-zinc-600 dark:text-zinc-400',
          borderColor: 'border-zinc-200 dark:border-zinc-700',
          dotColor: 'bg-zinc-500',
          label: 'Unknown',
          shortLabel: '?',
          description: ''
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

  const formatEventTimestamp = (timestamp: string) => {
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

  const truncateEventUrl = (url: string, maxLength: number = 30) => {
    if (url.length <= maxLength) return url;
    return url.substring(0, maxLength) + '...';
  };

  // Get icon and content based on event source type
  const getEventSourceConfig = (type: 'semaphore' | 'webhook' | 'http') => {
    switch (type) {
      case 'semaphore':
        return {
          icon: (
            <img width={24} src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAMAAABF0y+mAAAAM1BMVEVHcEwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADbQS4qAAAAEXRSTlMAYq64jCpx/8oGF/mjNBDW6uM72ZcAAACJSURBVHgBzdBFAoAwEATBjbv8/7WwTHA50ziFhv6ekEpp80jWIR/uJt1W/LCbwpTV6a7ZcYV3vePq1QwOGu8n1sifJvb7Nm1EgVd8J6x0vWqlkBxU98XmkxlaxwM8jYzjxLwX+Gtr2hWGO1F1m8Ik0VWTtmMU6FR0aLe73g0FP8zSU0YrJQX9vAn47gbljcJgwwAAAABJRU5ErkJggg==" alt="Semaphore" />
          ),
          title: 'Semaphore Integration'
        };
      case 'webhook':
        return {
          icon: <MaterialSymbol name={currentIcon} size="md" className="text-gray-700 dark:text-zinc-300" />,
          title: 'Webhook Endpoint'
        };
      case 'http':
        return {
          icon: <MaterialSymbol name="http" size="md" className="text-gray-700 dark:text-zinc-300" />,
          title: 'HTTP Endpoint'
        };
      default:
        return {
          icon: <MaterialSymbol name="sensors" size="md" className="text-gray-700 dark:text-zinc-300" />,
          title: 'Event Source'
        };
    }
  };

  // Convert passed events to EventData format with randomized statuses
  const convertedEvents: EventData[] = events.length > 0 
    ? events.map((event, index) => {
        // Randomize status if not provided
        const statuses: ('pending' | 'forwarded' | 'discarded')[] = ['pending', 'forwarded', 'discarded'];
        const randomStatus = event.status || statuses[index % statuses.length];
        
        return {
          id: event.id,
          url: event.url,
          status: randomStatus,
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
        };
      })
    : mockEventSourceEvents;

  // State for event detail tabs
  const [eventDetailTabs, setEventDetailTabs] = useState<Record<string, 'details' | 'headers' | 'payload'>>({});
  
  // State for current icon (for webhook nodes)
  const [currentIcon, setCurrentIcon] = useState<string>(nodeIcon || 'webhook');
  

  // Render event details for expanded events
  const renderEventDetails = (event: EventData) => {
    const statusConfig = getEventStatusConfig(event.status);
    const activeDetailTab = eventDetailTabs[event.id] || 'details';
    
    const setActiveDetailTab = (tab: 'details' | 'headers' | 'payload') => {
      setEventDetailTabs(prev => ({
        ...prev,
        [event.id]: tab
      }));
    };

    // Count headers
    const headerCount = event.headers ? Object.keys(event.headers).length : 0;
    
    // Define tabs for this event
    const eventTabs: Tab[] = [
      { id: 'details', label: 'Details' },
      { id: 'headers', label: 'Headers' },
      { id: 'payload', label: 'Payload' }
    ];
    
    return (
      <div className="mt-3 space-y-4">
        {/* Event Overview */}
        <div className="border border-gray-200 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-900">
          {/* Tabs */}
          <div className="border-b border-gray-200 dark:border-zinc-700">
            <ControlledTabs
              tabs={eventTabs}
              activeTab={activeDetailTab}
              size="xs"
              onTabChange={(tabId) => setActiveDetailTab(tabId as 'details' | 'headers' | 'payload')}
              variant="underline"
            />
          </div>

          {/* Tab Content */}
          <div className="px-4 py-3">
            {activeDetailTab === 'details' && (
              <div className="space-y-4">
                {/* Event ID - Full Width */}
                

                {/* Two Column Layout */}
                <div className="grid grid-cols-2 gap-6 text-sm">
                  {/* Left Column */}
                  <div>
                    <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">
                      STATE
                    </div> 
                    <div className="text-blue-600 dark:text-blue-400 text-xs font-medium">
                      <div className="flex items-center gap-2">
                        <div className={`w-2 h-2 ${statusConfig.dotColor} rounded-full flex-shrink-0`}></div>
                        <span className={`text-xs font-medium ${statusConfig.labelColor}`}>
                          {statusConfig.label}
                        </span>
                      </div>
                    </div>
                  </div>
                  <div>
                    <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">
                      RECEIVED ON
                    </div>
                    <div className="text-xs text-gray-900 dark:text-zinc-200">
                      {event.id === 'event-3' ? 'Aug 14, 2025 19:03:12' : new Date(event.timestamp).toLocaleString('en-US', {
                        month: 'short',
                        day: 'numeric',
                        year: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit',
                        second: '2-digit'
                      })}
                    </div>
                  </div>
                  
                  
                  <div>
                    <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">
                      SOURCE
                    </div>
                    <div className="text-xs text-gray-900 dark:text-zinc-200">
                      {event.id === 'event-3' ? 'Zawkey semaphore org' : 
                        event.url.includes('github.com') ? 'GitHub Webhook' : 
                        event.url.includes('gitlab.com') ? 'GitLab Webhook' :
                        event.url.includes('bitbucket.org') ? 'Bitbucket Webhook' :
                        'External Webhook'}
                    </div>
                  </div>
                  
                  

                 
                  
                  <div>
                    <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">
                      TYPE
                    </div>
                    <div className="text-xs font-medium">
                      {event.id === 'event-3' ? 'pipeline_done' : event.type || 'webhook'}
                    </div>
                  </div>
                  <div className="col-span-2">
                    <div>
                      <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">
                        EVENT ID
                      </div>
                      <div className="font-mono text-xs text-gray-900 dark:text-zinc-200 break-all">
                        {event.id === 'event-3' ? '423ae53e-f67a-43f4-9bea-8e90d5db3a27' : event.id}
                      </div>
                    </div>
                  </div>  
                    
                  

                </div>
              </div>
            )}

            {activeDetailTab === 'headers' && (
              <div>
                {event.headers && Object.keys(event.headers).length > 0 ? (
                  <div className="space-y-2">
                    {Object.entries(event.headers).map(([key, value]) => (
                      <div key={key} className="bg-zinc-50 dark:bg-zinc-800 rounded border border-gray-200 dark:border-zinc-700 p-3 h-60 max-h-60">
                        <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium pr-2">{key}</span>
                        <span className="text-xs font-mono text-gray-900 dark:text-zinc-200 text-right break-all">{value}</span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-xs text-gray-500 dark:text-zinc-400 italic">
                    No headers available
                  </div>
                )}
              </div>
            )}

            {activeDetailTab === 'payload' && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs font-medium text-gray-500 dark:text-zinc-400">Request Body</span>
                  <div className="flex items-center">
                    <Link className="!text-xs flex items-center" href="#">
                      <MaterialSymbol name="content_copy" size="sm" className="mr-1" />
                      Copy
                    </Link>
                  </div>
                </div>
                {event.payload ? (
                  <div className="bg-zinc-50 dark:bg-zinc-800 rounded border border-gray-200 dark:border-zinc-700 p-3 h-60 max-h-60 overflow-y-auto">
                    <pre className="text-xs font-mono text-gray-900 dark:text-zinc-200 whitespace-pre-wrap">
                      {JSON.stringify(event.payload, null, 2)}
                    </pre>
                  </div>
                ) : (
                  <div className="bg-gray-50 dark:bg-zinc-800 rounded border p-3">
                    <div className="text-xs text-gray-500 dark:text-zinc-400 italic">
                      No payload data available
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    );
  };


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
                 <span className="text-xs text-gray-600 dark:text-zinc-400 font-medium font-mono">Node_name.{key}</span>
                 <div className="flex items-center gap-2 truncate">
                   
                   <Badge color='zinc' className='font-mono !text-xs truncate'>
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


  if (!isOpen) return null;

  return (
    <div className={clsx(
      'absolute right-0 top-0 h-full w-110 bg-white dark:bg-zinc-900 border-l border-gray-200 dark:border-zinc-700 shadow-lg z-50 flex flex-col',
      className
    )}>
      {/* Header */}
      <div className="flex items-center justify-between p-4 pb-2">
        <div className="flex items-center gap-3">
        <div className='rounded-lg bg-zinc-100 dark:bg-zinc-700 p-2'>
          {source === 'eventSource' ? getEventSourceConfig(eventSourceType).icon : (
            <img width={24} src="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABwAAAAcCAMAAABF0y+mAAAAM1BMVEVHcEwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAADbQS4qAAAAEXRSTlMAYq64jCpx/8oGF/mjNBDW6uM72ZcAAACJSURBVHgBzdBFAoAwEATBjbv8/7WwTHA50ziFhv6ekEpp80jWIR/uJt1W/LCbwpTV6a7ZcYV3vePq1QwOGu8n1sifJvb7Nm1EgVd8J6x0vWqlkBxU98XmkxlaxwM8jYzjxLwX+Gtr2hWGO1F1m8Ik0VWTtmMU6FR0aLe73g0FP8zSU0YrJQX9vAn47gbljcJgwwAAAABJRU5ErkJggg==" alt="" />
          )}
        </div>
        <h3 className="font-semibold text-gray-900 dark:text-white">
          {source === 'eventSource' ? getEventSourceConfig(eventSourceType).title : nodeTitle}
        </h3>

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
            {source === 'eventSource' ? (
              /* LATEST EVENTS for EventSource */
              <div>
                <div className="flex items-center justify-between mb-4">
                  <Text className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide">
                    LATEST EVENTS
                  </Text>
                  <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                    View all
                  </Link>
                </div>
                
                <div className="space-y-3">
                  {convertedEvents.map((event) => {
                    const statusConfig = getEventStatusConfig(event.status);
                    const isExpanded = expandedEvents.has(event.id);
                    
                    return (
                      <div key={event.id} className={`border bg-zinc-50 dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 rounded-lg`}>
                        <div 
                          className="p-3"
                          
                        >
                          <div className="cursor-pointer flex items-center justify-between" onClick={() => toggleEventExpansion(event.id)}>
                            <div className="flex items-center gap-2 truncate pr-2">
                            <div className={`w-2 h-2 rounded-full flex-shrink-0 ${statusConfig.dotColor}`}></div>
                              
                              <span className="font-medium truncate text-sm dark:text-white font-mono">
                                {truncateEventUrl(event.url)}
                              </span>
                            </div>
                            <div className="flex items-center gap-3">
                              {!isExpanded && (
                                <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">
                                  {formatEventTimestamp(event.timestamp)}
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
            ) : (
              /* Recent Runs for workflow nodes */
              <>
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
                    <div key={run.id} className={"border-b border-l border-r border-gray-200 dark:border-zinc-700 "+statusConfig.bgColor + " " + statusConfig.borderColor } >
                      <div 
                        className="p-3"
                        
                      >
                        <div className="flex items-center justify-between cursor-pointer" onClick={() => toggleRunExpansion(run.id)}>
                        <div className='text-xs gap-2'>
                              <div className='flex items-center gap-2 mb-2'>
                                {run.status == 'success' && (
                                <BadgeButton color='green' className='!flex !items-center'>
                                   <MaterialSymbol name='check_circle' size='sm'/>
                                  <span className='uppercase'>Passed</span>
                                </BadgeButton>
                                )}
                                {run.status == 'failed' && (
                                <BadgeButton color='red' className='!flex !items-center'>
                                   <MaterialSymbol name='cancel' size='sm'/>
                                  <span className='uppercase'>{run.status}</span>
                                </BadgeButton>
                                )}
                                {run.status == 'running' && (
                                <BadgeButton color='blue' className='!flex !items-center'>
                                  <MaterialSymbol name='sync' size='sm' className='animate-spin'/>
                                  <span className='uppercase'>{run.status}</span>
                                </BadgeButton>
                                )}
                                <Link href="#" className="font-medium text-blue-600 dark:text-blue-400 flex items-center gap-1 text-sm">{run.name} 
                                <MaterialSymbol name='arrow_outward' size='sm'/>
                                </Link>
                                
                              </div>
                             
                              
                              <div className='flex items-center gap-4 mb-1'>
                              <div className='flex items-center gap-1'>
                                  <MaterialSymbol name='calendar_today' size='md' className='text-gray-600 dark:text-zinc-400'/>
                                  <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">
                                    {run.status == 'running' ? 'Started on ' + run.timestamp : 'Finished on ' + run.timestamp}</span>
                                </div>
                                <div className='flex items-center gap-1'>
                                  <MaterialSymbol name='timer' size='md' className='text-gray-600 dark:text-zinc-400'/>
                                  <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">{run.duration}</span>
                                </div>
                               
                              </div>
                              <div className='flex items-center gap-1'>
                                <MaterialSymbol name='bolt' size='md' className='text-gray-600 dark:text-zinc-400'/>
                                <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap"><Link href="#" className="text-blue-600 dark:text-blue-400">AI Agent triade</Link> &bull; Event ID: <Link href="#" className="text-blue-600 dark:text-blue-400">324234234-23423424-23423</Link></span>
                              </div>
                            </div>
                          <div className="flex items-center gap-3">
                            
                            <MaterialSymbol 
                                name={isExpanded ? 'expand_less' : 'expand_more'} 
                                size="lg" 
                                className="text-gray-600 dark:text-zinc-400" 
                              />
                          </div>
                        </div>
                        
                        {isExpanded && (
                          <div className="mt-3 space-y-3">
                            {/* Run details */}
                            

                            
              
                            {renderInputsOutputs2(run.inputs, run.outputs)}
                            {/* Queue Information */}
                            <div className='bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700 p-4 text-xs'>
                              <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-2">
                                Queue Information
                              </div>
                              <div className='space-y-1'>
                                {run.queuedAt && (
                                  <div className='flex items-center gap-1'>
                                    <MaterialSymbol name='schedule' size='md' className='text-gray-600 dark:text-zinc-400'/>
                                    <span className="text-xs text-gray-500 dark:text-zinc-400">Added to queue on {run.queuedAt}</span>
                                  </div>
                                )}
                                {run.conditionMetAt && (
                                  <div className='flex items-center gap-1'>
                                    <MaterialSymbol name='check_circle' size='md' className='text-gray-600 dark:text-zinc-400'/>
                                    <span className="text-xs text-gray-500 dark:text-zinc-400">Approved on {run.conditionMetAt}</span>
                                  </div>
                                )}
                                {run.approvedBy && (
                                  <div className='flex items-center gap-1'>
                                    <MaterialSymbol name='person' size='md' className='text-gray-600 dark:text-zinc-400'/>
                                    <span className="text-xs text-gray-500 dark:text-zinc-400">Approved by <Link href="#" className="text-blue-600 dark:text-blue-400">{run.approvedBy}</Link></span>
                                  </div>
                                )}
                                
                              </div>
                            
                            </div>

                            {/* Queue Information 2 - Grid Layout */}
                            <div className='bg-white dark:bg-zinc-900 border border-gray-200 dark:border-zinc-700 p-4'>
                              
                              <div className="grid grid-cols-2 gap-6 text-sm">
                                {/* Left Column */}
                                <div>
                                  <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">
                                    QUEUED ON
                                  </div>
                                  <div className="text-xs text-gray-900 dark:text-zinc-200">
                                    {run.queuedAt || 'N/A'}
                                  </div>
                                </div>
                                
                                <div>
                                  <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">
                                    APPROVED ON
                                  </div>
                                  <div className="text-xs text-gray-900 dark:text-zinc-200">
                                    {run.conditionMetAt || 'N/A'}
                                  </div>
                                </div>
                                
                                {/* Right Column */}
                                <div>
                                  <div className="text-xs font-semibold text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">
                                    APPROVED BY
                                  </div>
                                  <div className="text-xs text-gray-900 dark:text-zinc-200">
                                    {run.approvedBy ? (
                                      <Link href="#" className="text-blue-600 dark:text-blue-400">{run.approvedBy}</Link>
                                    ) : 'N/A'}
                                  </div>
                                </div>
                                
                                
                              </div>
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Queue - only show for workflow nodes */}
            {source == 'workflow' && (
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
            )}
            </>
            )}
          </div>
        )}
        
        {activeTab === 'history' && (
          <div className="p-4 space-y-6">
            {source === 'eventSource' ? (
              /* Historical Events for EventSource */
              <div>
                <div className="flex items-center justify-between mb-4">
                  <Text className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide">
                    EVENT HISTORY ({mockHistoryEvents.length})
                  </Text>
                </div>
                
                <div className="space-y-3">
                  {mockHistoryEvents.map((event) => {
                    const statusConfig = getEventStatusConfig(event.status);
                    const isExpanded = expandedHistoryEvents.has(event.id);
                    
                    return (
                      <div key={event.id} className={`border bg-zinc-50 dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 rounded-lg`}>
                        <div 
                          className="p-3"
                          
                        >
                          <div className="cursor-pointer flex items-center justify-between" onClick={() => toggleHistoryEventExpansion(event.id)}>
                            <div className="flex items-center gap-2 truncate pr-2">
                            <div className={`w-2 h-2 rounded-full flex-shrink-0 ${statusConfig.dotColor}`}></div>
                              
                              <span className="font-medium truncate text-sm dark:text-white font-mono">
                                {truncateEventUrl(event.url)}
                              </span>
                            </div>
                            <div className="flex items-center gap-3">
                              {!isExpanded && (
                                <span className="text-xs text-gray-500 dark:text-zinc-400 whitespace-nowrap">
                                  {formatEventTimestamp(event.timestamp)}
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
            ) : (
              /* Historical Runs for workflow nodes */
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
            )}
          </div>
        )}
        
        {activeTab === 'settings' && (
          <div className="p-4">
            {source === 'eventSource' ? (
              <div className="space-y-6">
                {eventSourceType === 'semaphore' && (
                  <div className="space-y-4">
                    <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-4 bg-white dark:bg-zinc-900">
                      <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide mb-3">
                        Semaphore Configuration
                      </div>
                      <div className="space-y-3">
                        <div>
                          <div className="text-xs font-medium text-gray-600 dark:text-zinc-400 mb-1">Integration</div>
                          <div className="text-sm text-gray-900 dark:text-zinc-200">Semaphore CI/CD Integration</div>
                        </div>
                        <div>
                          <div className="text-xs font-medium text-gray-600 dark:text-zinc-400 mb-1">Project</div>
                          <div className="text-sm text-gray-900 dark:text-zinc-200">Zawkey semaphore org</div>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
                
                {eventSourceType === 'webhook' && (
                  <div className="space-y-4">

                    {/* Webhook Configuration */}
                    <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-4 bg-white dark:bg-zinc-900">
                      <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide mb-3">
                        Webhook Configuration
                      </div>
                      <div className="space-y-3">
                        <div>
                          <div className="text-xs font-medium text-gray-600 dark:text-zinc-400 mb-1">Webhook URL</div>
                          <div className="flex items-center gap-2">
                            <code className="flex-1 text-sm font-mono bg-gray-50 dark:bg-zinc-800 px-3 py-2 rounded border text-gray-900 dark:text-zinc-200">
                              https://hooks.superplane.com/webhook/abc123def456
                            </code>
                            <Button plain className="text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300">
                              <MaterialSymbol name="content_copy" size="sm" />
                            </Button>
                          </div>
                        </div>
                        <div>
                          <div className="text-xs font-medium text-gray-600 dark:text-zinc-400 mb-1">Secret Key</div>
                          <div className="flex items-center gap-2">
                            <code className="flex-1 text-sm font-mono bg-gray-50 dark:bg-zinc-800 px-3 py-2 rounded border text-gray-900 dark:text-zinc-200">
                              wh_sec_789xyz012abc345def678ghi901jkl
                            </code>
                            <Button plain className="text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300">
                              <MaterialSymbol name="content_copy" size="sm" />
                            </Button>
                            <Button plain className="text-gray-600 hover:text-gray-700 dark:text-zinc-400 dark:hover:text-zinc-300">
                              <MaterialSymbol name="refresh" size="sm" />
                            </Button>
                          </div>
                          <div className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                            Use this key to validate webhook authenticity
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
                
                {eventSourceType === 'http' && (
                  <div className="space-y-4">
                    <div className="border border-gray-200 dark:border-zinc-700 rounded-lg p-4 bg-white dark:bg-zinc-900">
                      <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide mb-3">
                        HTTP Endpoint Configuration
                      </div>
                      <div className="space-y-3">
                        <div>
                          <div className="text-xs font-medium text-gray-600 dark:text-zinc-400 mb-1">Endpoint URL</div>
                          <div className="flex items-center gap-2">
                            <code className="flex-1 text-sm font-mono bg-gray-50 dark:bg-zinc-800 px-3 py-2 rounded border text-gray-900 dark:text-zinc-200">
                              https://api.superplane.com/events/http/xyz789abc123
                            </code>
                            <Button plain className="text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300">
                              <MaterialSymbol name="content_copy" size="sm" />
                            </Button>
                          </div>
                        </div>
                        <div>
                          <div className="text-xs font-medium text-gray-600 dark:text-zinc-400 mb-1">API Key</div>
                          <div className="flex items-center gap-2">
                            <code className="flex-1 text-sm font-mono bg-gray-50 dark:bg-zinc-800 px-3 py-2 rounded border text-gray-900 dark:text-zinc-200">
                              sp_key_456def789ghi012abc345jkl678mno901
                            </code>
                            <Button plain className="text-blue-600 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300">
                              <MaterialSymbol name="content_copy" size="sm" />
                            </Button>
                            <Button plain className="text-gray-600 hover:text-gray-700 dark:text-zinc-400 dark:hover:text-zinc-300">
                              <MaterialSymbol name="refresh" size="sm" />
                            </Button>
                          </div>
                          <div className="text-xs text-gray-500 dark:text-zinc-400 mt-1">
                            Include this key in Authorization header
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <Text className="text-gray-500 dark:text-zinc-400">Settings view coming soon...</Text>
            )}
          </div>
        )}
      </div>
    </div>
  );
}