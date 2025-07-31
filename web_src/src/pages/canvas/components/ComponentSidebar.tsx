import React, { useState } from 'react';


export interface ComponentSidebarProps {
  isOpen: boolean;
  onClose: () => void;
  onNodeAdd: (nodeType: string, executorType?: string, eventSourceType?: string) => void;
  className?: string;
}

interface ComponentDefinition {
  id: string;
  name: string;
  description: string;
  icon: string;
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

  const components: ComponentDefinition[] = [
    {
      id: 'stage',
      name: 'Semaphore Stage',
      description: 'Add a Semaphore-based stage to your canvas',
      icon: 'rocket_launch',
      category: 'Stages',
      executorType: 'semaphore'
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
      id: 'connection_group',
      name: 'Connection Group',
      description: 'Add a connection group to your canvas',
      icon: 'group',
      category: 'Deployment',
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
      icon: 'webhook',
      category: 'Event Sources',
      eventSourceType: 'semaphore'
    },

  ];

  const categories = Array.from(new Set(components.map(c => c.category)));

  const handleAddComponent = (componentType: string, executorType?: string, eventSourceType?: string) => {
    onNodeAdd(componentType, executorType, eventSourceType);
  };

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="max-w-100 h-[calc(100vh-42px)] fixed top-[42px] left-0 right-0 bottom-0 z-40 overflow-y-auto text-left bg-white dark:bg-gray-800"
        aria-hidden="true"
      >

        {/* Sidebar */}
        <div
          className={`bg-white z-50 transform transition-transform duration-300 ease-in-out ${className}`}
          role="dialog"
          aria-modal="true"
          aria-labelledby="sidebar-title"
        >
          <div className="flex flex-col">
            {/* Header */}
            <div className="flex items-center justify-between px-4 py-2 border-b border-gray-200">
              <h2 id="sidebar-title" className="text-md font-semibold text-gray-900">
                Components
              </h2>
              <button
                type="button"
                onClick={onClose}
                className="text-gray-400 hover:text-gray-600 transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 rounded-md p-1"
                aria-label="Close sidebar"
              >
                <span className="material-symbols-outlined">close</span>
              </button>
            </div>

            {/* Search */}
            <div className="p-6 border-b border-gray-200">
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <span className="material-symbols-outlined text-gray-400">search</span>
                </div>
                <input
                  type="text"
                  placeholder="Search components..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-md leading-5 bg-white placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 focus:ring-1 focus:ring-primary-500 focus:border-primary-500 text-sm"
                  aria-label="Search components"
                />
              </div>
            </div>

            {/* Component Categories */}
            <div className="flex-1 overflow-y-auto">
              {categories.map((category) => (
                <div key={category} className="p-6 border-b border-gray-100">
                  <h3 className="text-sm font-medium text-gray-500 uppercase tracking-wide mb-4">
                    {category}
                  </h3>
                  <div className="space-y-3">
                    {components
                      .filter(c => c.category === category)
                      .filter(c => c.name.toLowerCase().includes(searchQuery.toLowerCase()))
                      .map((component) => (
                        <button
                          key={`${component.id}-${component.name}`}
                          type="button"
                          onClick={() => handleAddComponent(component.id, component.executorType, component.eventSourceType)}
                          className="w-full text-left p-4 border border-gray-200 rounded-lg hover:border-primary-300 hover:bg-primary-50 transition-colors group focus:outline-none focus:ring-2 focus:ring-primary-500"
                          aria-label={`Add ${component.name} component`}
                        >
                          <div className="flex items-start">
                            <div className="flex-shrink-0">
                              <div className="w-10 h-10 bg-gray-100 group-hover:bg-primary-100 rounded-lg flex items-center justify-center transition-colors">
                                <span className="material-symbols-outlined text-gray-600 group-hover:text-primary-600 transition-colors">
                                  {component.icon}
                                </span>
                              </div>
                            </div>
                            <div className="ml-3 flex-1 min-w-0">
                              <h4 className="text-sm font-medium text-gray-900 group-hover:text-primary-900 transition-colors">
                                {component.name}
                              </h4>
                              <p className="text-xs text-gray-500 mt-1 line-clamp-2">
                                {component.description}
                              </p>
                            </div>
                            <div className="ml-2 flex-shrink-0">
                              <span className="material-symbols-outlined text-gray-400 group-hover:text-primary-500 transition-colors">
                                add
                              </span>
                            </div>
                          </div>
                        </button>
                      ))}
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