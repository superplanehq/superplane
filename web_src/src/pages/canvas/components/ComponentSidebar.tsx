import React, { useState } from 'react';
import { useCanvasStore } from '../store/canvasStore';
import SemaphoreLogo from '@/assets/semaphore-logo-sign-black.svg';
import GithubLogo from '@/assets/github-mark.svg';
import Tippy from '@tippyjs/react';
import 'tippy.js/dist/tippy.css';
import { NodeType } from '../utils/nodeFactories';

export interface ComponentSidebarProps {
  isOpen: boolean;
  onClose: () => void;
  onNodeAdd: (nodeType: NodeType, executorType?: string, eventSourceType?: string) => void;
  className?: string;
}

interface ComponentDefinition {
  id: NodeType;
  name: string;
  description: string;
  icon?: string;
  image?: string;
  category: string;
  executorType?: string;
  eventSourceType?: string;
}

export const ComponentSidebar: React.FC<ComponentSidebarProps> = ({
  isOpen,
  onClose,
  onNodeAdd,
  className = '',
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const { eventSources } = useCanvasStore();

  // Check if there are any event sources in the canvas
  const hasEventSources = eventSources.length > 0;

  const components: ComponentDefinition[] = [
    {
      id: 'stage',
      name: 'Semaphore Stage',
      description: 'Add a Semaphore-based stage to your canvas',
      image: SemaphoreLogo,
      category: 'Stages',
      executorType: 'semaphore'
    },
    {
      id: 'stage',
      name: 'GitHub Stage',
      description: 'Add a GitHub-based stage to your canvas',
      image: GithubLogo,
      category: 'Stages',
      executorType: 'github'
    },
    {
      id: 'stage',
      name: 'HTTP Stage',
      description: 'Add an HTTP-based stage to your canvas',
      icon: 'rocket_launch',
      category: 'Stages',
      executorType: 'http'
    },
    {
      id: 'event_source',
      name: 'Webhook Event Source',
      description: 'Add a webhook-based event source to your canvas',
      icon: 'webhook',
      category: 'Event Sources',
      eventSourceType: 'webhook'
    },
    {
      id: 'event_source',
      name: 'Semaphore Event Source',
      description: 'Add a Semaphore-based event source to your canvas',
      image: SemaphoreLogo,
      category: 'Event Sources',
      eventSourceType: 'semaphore'
    },
    {
      id: 'event_source',
      name: 'GitHub Event Source',
      description: 'Add a GitHub-based event source to your canvas',
      image: GithubLogo,
      category: 'Event Sources',
      eventSourceType: 'github'
    },
    {
      id: 'connection_group',
      name: 'Connection Group',
      description: 'Add a connection group to your canvas',
      icon: 'schema',
      category: 'Groups',
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

  const handleAddComponent = (componentType: NodeType, executorType?: string, eventSourceType?: string) => {
    onNodeAdd(componentType, executorType, eventSourceType);
  };

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="max-w-100 h-[calc(100vh-42px)] fixed top-[42px] left-0 right-0 bottom-0 z-40 overflow-y-auto text-left bg-white dark:bg-zinc-900"
        aria-hidden="true"
      >

        {/* Sidebar */}
        <div
          className={`bg-white dark:bg-zinc-900 z-50 transform transition-transform duration-300 ease-in-out ${className}`}
          role="dialog"
          aria-modal="true"
          aria-labelledby="sidebar-title"
        >
          <div className="flex flex-col">
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-2 border-b border-gray-200 dark:border-zinc-700">
              <h2 id="sidebar-title" className="text-md font-semibold text-gray-900 dark:text-zinc-100">
                Components
              </h2>
              <button
                type="button"
                onClick={onClose}
                className="text-gray-400 dark:text-zinc-500 hover:text-gray-600 dark:hover:text-zinc-300 transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 rounded-md p-1"
                aria-label="Close sidebar"
              >
                <span className="material-symbols-outlined">close</span>
              </button>
            </div>

            {/* Search */}
            <div className="p-6 border-b border-gray-200 dark:border-zinc-700">
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <span className="material-symbols-outlined text-gray-400 dark:text-zinc-500">search</span>
                </div>
                <input
                  type="text"
                  placeholder="Search components..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="block w-full pl-10 pr-3 py-2 border border-gray-300 dark:border-zinc-600 rounded-md leading-5 bg-white dark:bg-zinc-800 text-gray-900 dark:text-zinc-100 placeholder-gray-500 dark:placeholder-zinc-400 focus:outline-none focus:placeholder-gray-400 dark:focus:placeholder-zinc-500 focus:ring-1 focus:ring-primary-500 focus:border-primary-500 text-sm"
                  aria-label="Search components"
                />
              </div>
            </div>

            {/* Component Categories */}
            <div className="flex-1 overflow-y-auto">
              {categories.map((category) => (
                <div key={category} className="p-6 border-b border-gray-100 dark:border-zinc-700">
                  <h3 className="text-sm font-medium text-gray-500 dark:text-zinc-400 uppercase tracking-wide mb-4">
                    {category}
                  </h3>
                  <div className="space-y-3">
                    {components
                      .filter(c => c.category === category)
                      .filter(c => c.name.toLowerCase().includes(searchQuery.toLowerCase()))
                      .map((component) => {
                        const disabled = isComponentDisabled(component);
                        const disabledMessage = getDisabledMessage(component);

                        const buttonElement = (
                          <button
                            type="button"
                            onClick={() => !disabled && handleAddComponent(component.id, component.executorType, component.eventSourceType)}
                            disabled={disabled}
                            className={`w-full text-left p-4 border rounded-lg transition-colors group focus:outline-none ${disabled
                              ? 'border-gray-200 dark:border-zinc-700 bg-gray-50 dark:bg-zinc-800 cursor-not-allowed opacity-60'
                              : 'border-gray-200 dark:border-zinc-700 hover:border-primary-300 dark:hover:border-primary-600 hover:bg-primary-50 dark:hover:bg-primary-900/20 focus:ring-2 focus:ring-primary-500'
                              }`}
                            aria-label={disabled ? disabledMessage : `Add ${component.name} component`}
                          >
                            <div className="flex items-start">
                              <div className="flex-shrink-0">
                                <div className={`w-10 h-10 rounded-lg flex items-center justify-center transition-colors ${disabled
                                  ? 'bg-gray-100 dark:bg-zinc-700'
                                  : 'bg-gray-100 dark:bg-zinc-900 '
                                  }`}>
                                  {component.image ? (
                                    <img
                                      src={component.image}
                                      alt={component.name}
                                      className="w-8 h-8 object-contain bg-white dark:bg-white p-1 rounded"
                                    />
                                  ) : (
                                    <span className={`material-symbols-outlined transition-colors ${disabled
                                      ? 'text-gray-400 dark:text-zinc-500'
                                      : 'text-gray-600 dark:text-zinc-300 group-hover:text-primary-600 dark:group-hover:text-primary-400'
                                      }`}>
                                      {component.icon}
                                    </span>)}
                                </div>
                              </div>
                              <div className="ml-3 flex-1 min-w-0">
                                <h4 className={`text-sm font-medium transition-colors ${disabled
                                  ? 'text-gray-500 dark:text-zinc-400'
                                  : 'text-gray-900 dark:text-zinc-100 group-hover:text-primary-900 dark:group-hover:text-primary-200'
                                  }`}>
                                  {component.name}
                                </h4>
                                <p className="text-xs text-gray-500 dark:text-zinc-400 mt-1 line-clamp-2">
                                  {disabled ? disabledMessage : component.description}
                                </p>
                              </div>
                              <div className="ml-2 flex-shrink-0">
                                <span className={`material-symbols-outlined transition-colors ${disabled
                                  ? 'text-gray-300 dark:text-zinc-600'
                                  : 'text-gray-400 dark:text-zinc-500 group-hover:text-primary-500 dark:group-hover:text-primary-400'
                                  }`}>
                                  {disabled ? 'block' : 'add'}
                                </span>
                              </div>
                            </div>
                          </button>
                        );

                        return (
                          <div key={`${component.id}-${component.name}`} className="relative">
                            {disabled ? (
                              <Tippy content={disabledMessage} placement="top">
                                <div>{buttonElement}</div>
                              </Tippy>
                            ) : (
                              buttonElement
                            )}
                          </div>
                        );
                      })}
                  </div>
                </div>
              ))}
            </div>

          </div>
        </div>
      </div>
    </>
  );
};