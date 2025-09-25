import React, { useState, useEffect } from 'react';
import { useCanvasStore } from '../store/canvasStore';
import GithubLogo from '@/assets/github-mark.svg';
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';
import DockerLogo from '@/assets/docker-logo.svg';
import KubernetesLogo from '@/assets/kubernetes-logo.svg';
import TerraformLogo from '@/assets/terraform-logo.svg';
import AwsLogo from '@/assets/aws-logo.svg';
import GitLogo from '@/assets/git-logo.svg';
import NpmLogo from '@/assets/npm-logo.svg';
import PythonLogo from '@/assets/python-logo.svg';
import HelmLogo from '@/assets/helm-logo.svg';
import AnsibleLogo from '@/assets/ansible-logo.svg';
import SonarqubeLogo from '@/assets/sonarqube-logo.svg';
import { NodeType } from '../utils/nodeFactories';
import { SidebarItem } from './SidebarItem';
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol';
import { Button } from '../../../components/Button/button';
import { Input } from '../../../components/Input/input';
import { Switch } from '../../../components/Switch/switch';
import { Badge } from '../../../components/Badge/badge';
import Tippy from '@tippyjs/react';
import 'tippy.js/dist/tippy.css';
import { SuperplaneConnectionType } from '@/api-client';

export type ConnectionInfo = { name: string; type: SuperplaneConnectionType };

export interface ComponentSidebarProps {
  isOpen: boolean;
  onClose: () => void;
  onNodeAdd: (nodeType: NodeType, executorType?: string, eventSourceType?: string, focusedNodeInfo?: ConnectionInfo | null) => void;
  className?: string;
}

interface ComponentDefinition {
  id: NodeType;
  name: string;
  description: string;
  icon?: string;
  image?: string;
  category: string;
  category_description: string;
  executorType?: string;
  eventSourceType?: string;
  comingSoon?: boolean;
}

export const ComponentSidebar: React.FC<ComponentSidebarProps> = ({
  isOpen,
  onClose,
  onNodeAdd,
  className = '',
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [hideComingSoon, setHideComingSoon] = useState<boolean>(true);
  const [showConfigPanel, setShowConfigPanel] = useState(false);

  useEffect(() => {
    const storedHideComingSoon = localStorage.getItem('hideComingSoon');
    if (storedHideComingSoon !== null) {
      setHideComingSoon(JSON.parse(storedHideComingSoon));
    }
  }, []);

  const handleHideComingSoonToggle = () => {
    const newValue = !hideComingSoon;
    setHideComingSoon(newValue);
    localStorage.setItem('hideComingSoon', JSON.stringify(newValue));
  };
  const { eventSources, focusedNodeId, nodes } = useCanvasStore();

  // Check if there are any event sources in the canvas
  const hasEventSources = eventSources.length > 0;

  // Get focused node name and type for auto-connecting
  const getFocusedNodeInfo = () => {
    if (!focusedNodeId) return null;

    const node = nodes.find(node => node.id === focusedNodeId);
    if (node) {
      return {
        name: node.data?.name,
        type: `TYPE_${node.type?.toUpperCase()}` as SuperplaneConnectionType
      };
    }

    return null;
  };

  const components: ComponentDefinition[] = [
    // Stages - Available
    {
      id: 'stage',
      name: 'Semaphore Stage',
      description: 'Run CI/CD pipeline',
      image: SemaphoreLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      executorType: 'semaphore'
    },
    {
      id: 'stage',
      name: 'HTTP Stage',
      description: 'Make HTTP requests to external services',
      icon: 'rocket_launch',
      category: 'Stages',
      category_description: 'Execute remote operations',
      executorType: 'http'
    },
    {
      id: 'stage',
      name: 'GitHub Stage',
      description: 'Add a GitHub-based stage to your canvas',
      image: GithubLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      executorType: 'github'
    },
    {
      id: 'stage',
      name: 'No-Op Stage',
      description: 'A stage that does nothing but returns random outputs',
      icon: 'check_circle',
      category: 'Stages',
      category_description: 'Execute remote operations',
      executorType: 'noop'
    },
    // Stages - Coming Soon
    {
      id: 'stage',
      name: 'Docker Build',
      description: 'Build and push Docker containers',
      image: DockerLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'stage',
      name: 'Kubernetes Deploy',
      description: 'Deploy applications to Kubernetes',
      image: KubernetesLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'stage',
      name: 'Terraform Apply',
      description: 'Provision infrastructure with Terraform',
      image: TerraformLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'stage',
      name: 'AWS CLI',
      description: 'Execute AWS CLI commands',
      image: AwsLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'stage',
      name: 'Git Operations',
      description: 'Perform Git operations and version control',
      image: GitLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'stage',
      name: 'NPM Build',
      description: 'Build Node.js applications with NPM',
      image: NpmLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'stage',
      name: 'Python Tests',
      description: 'Run Python tests with pytest',
      image: PythonLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'stage',
      name: 'Helm Deploy',
      description: 'Deploy applications with Helm charts',
      image: HelmLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'stage',
      name: 'Ansible Playbook',
      description: 'Run Ansible playbooks for configuration',
      image: AnsibleLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'stage',
      name: 'SonarQube Scan',
      description: 'Code quality analysis with SonarQube',
      image: SonarqubeLogo,
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    // Event Sources
    {
      id: 'event_source',
      name: 'Webhook Event Source',
      description: 'Trigger workflows from webhook events',
      icon: 'webhook',
      category: 'Event Sources',
      category_description: 'Emit events that can be used to trigger executions',
      eventSourceType: 'webhook'
    },
    {
      id: 'event_source',
      name: 'Scheduled Event Source',
      description: 'Trigger workflows on a schedule',
      icon: 'schedule',
      category: 'Event Sources',
      category_description: 'Emit events that can be used to trigger executions',
      eventSourceType: 'scheduled'
    },
    {
      id: 'event_source',
      name: 'Semaphore Event Source',
      description: 'Trigger workflows from Semaphore events',
      image: SemaphoreLogo,
      category: 'Event Sources',
      category_description: 'Emit events that can be used to trigger executions',
      eventSourceType: 'semaphore'
    },
    {
      id: 'event_source',
      name: 'GitHub Event Source',
      description: 'Trigger workflows from GitHub events',
      image: GithubLogo,
      category: 'Event Sources',
      category_description: 'Emit events that can be used to trigger executions',
      eventSourceType: 'github'
    },
    // Groups
    {
      id: 'connection_group',
      name: 'Connection Group',
      description: 'Group related workflow connections',
      icon: 'account_tree',
      category: 'Groups',
      category_description: 'Group related workflow connections',
    },
  ];

  const categories = Array.from(new Set(components.map(c => c.category)));

  // Function to check if a component should be disabled
  const isComponentDisabled = (component: ComponentDefinition): boolean => {
    // Disable stages and connection groups if there are no event sources
    if ((component.category === 'Stages' || component.category === 'Groups') && !hasEventSources) {
      return true;
    }
    return false;
  };

  // Function to get the disabled message for a component
  const getDisabledMessage = (component: ComponentDefinition): string => {
    if ((component.category === 'Stages' || component.category === 'Groups') && !hasEventSources) {
      return 'Add an Event Source first to enable this component';
    }
    return '';
  };

  const handleAddComponent = (component: ComponentDefinition) => {
    if (!isComponentDisabled(component) && !component.comingSoon) {
      const focusedNodeInfo = getFocusedNodeInfo();
      onNodeAdd(component.id, component.executorType, component.eventSourceType, focusedNodeInfo);
    }
  };

  if (!isOpen) return null;

  return (
    <>
      {/* Sidebar */}
      <div
        className={`bg-white dark:bg-zinc-900 absolute top-0 bottom-0 transform transition-transform duration-300 ease-in-out z-50 ${className}`}
        role="dialog"
        aria-modal="true"
        aria-labelledby="sidebar-title"
      >
        <div className="flex flex-col h-full">
          {/* Header */}
          <div className="flex items-center justify-between px-4 pt-4 pb-0 sticky top-0">
            <div className="flex items-center gap-3">
              <button
                onClick={onClose}
                aria-label="Close sidebar"
                className="px-2 py-1 bg-white dark:bg-zinc-900 border border-gray-300 dark:border-zinc-700 rounded-md shadow-md hover:bg-gray-50 dark:hover:bg-zinc-800 transition-all duration-300 flex items-center gap-2"
              >
                <MaterialSymbol name="menu_open" size="lg" className="text-gray-600 dark:text-zinc-300" />
              </button>

              <h2 id="sidebar-title" className="text-md font-semibold text-gray-900 dark:text-zinc-100">
                Components
              </h2>
            </div>

            <Button
              plain
              onClick={() => setShowConfigPanel(!showConfigPanel)}
              className="text-xs flex items-center"
              aria-label="Configure Components"
            >
              <MaterialSymbol name="tune" size="sm" />
            </Button>
          </div>

          {/* Configuration Panel */}
          {showConfigPanel && (
            <div className="px-4 pt-2">
              <div className="flex items-center text-xs">
                <Switch
                  checked={!hideComingSoon}
                  onChange={handleHideComingSoonToggle}
                  aria-label="Show coming soon components"
                  color='blue'
                />
                <span className="text-sm text-gray-600 dark:text-gray-400 ml-2">Include components with </span>
                <Badge color="indigo" className='mx-2'>
                  Soon
                </Badge>
              </div>
            </div>
          )}

          {/* Search */}
          <div className="p-4 pb-0">
            <div className="relative flex items-center border border-gray-200 dark:border-zinc-700 rounded-lg">
              <MaterialSymbol name="search" size="md" className="absolute left-3 text-gray-400 dark:text-zinc-500 z-10" />
              <Input
                name="search"
                placeholder="Search…"
                aria-label="Search"
                className="pl-14 border-0 focus:ring-0 focus:border-0 pl-5"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
          </div>

          {/* Component Categories */}
          <div className="flex-1 overflow-y-auto text-left">
            {categories.map((category) => (
              <div key={category} className="p-4">
                <h3 className="text-sm font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-1">
                  {category}
                </h3>
                <div className="!text-xs text-gray-500 dark:text-zinc-400 mb-3">
                  {components.find(c => c.category === category)?.category_description}
                </div>
                <div className="space-y-1">
                  {components
                    .filter(c => c.category === category)
                    .filter(c => c.name.toLowerCase().includes(searchQuery.toLowerCase()))
                    .filter(c => {
                      // Filter based on the hideComingSoon state
                      return !hideComingSoon || !c.comingSoon;
                    })
                    .map((component) => {
                      const disabled = isComponentDisabled(component);
                      const disabledMessage = getDisabledMessage(component);

                      const sidebarItem = (
                        <SidebarItem
                          key={`${component.id}-${component.name}`}
                          title={component.name}
                          subtitle={component.description}
                          comingSoon={component.comingSoon}
                          disabled={disabled}
                          showSubtitle={false}
                          icon={
                            component.image ? (
                              <img
                                width={24}
                                height={24}
                                src={component.image}
                                alt={component.name}
                                className="flex-shrink-0"
                              />
                            ) : component.icon ? (
                              <MaterialSymbol name={component.icon} size="lg" className="text-gray-600 dark:text-zinc-300 flex-shrink-0" />
                            ) : (
                              <MaterialSymbol name="drag_indicator" size="lg" className="text-gray-600 dark:text-zinc-300 flex-shrink-0" />
                            )
                          }
                          onClickAddNode={disabled || component.comingSoon ? undefined : () => handleAddComponent(component)}
                          className={disabled || component.comingSoon ? "" : "cursor-pointer hover:bg-blue-50 dark:hover:bg-blue-900/20 border border-gray-200 dark:border-zinc-700"}
                        />
                      );

                      return disabled ? (
                        <Tippy key={`${component.id}-${component.name}`} content={disabledMessage} placement="top">
                          <div>{sidebarItem}</div>
                        </Tippy>
                      ) : (
                        sidebarItem
                      );
                    })}
                </div>
              </div>
            ))}
          </div>

        </div>
      </div>
    </>
  );
};