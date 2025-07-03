import React from 'react';
import { ComponentSidebarProps } from '../types';

/**
 * ComponentSidebar component following SaaS guidelines
 * - Uses TypeScript with proper interfaces
 * - Implements accessibility features
 * - Follows responsive design principles
 * - Uses semantic HTML and ARIA attributes
 */
export const ComponentSidebar: React.FC<ComponentSidebarProps> = ({
  isOpen,
  onClose,
  onNodeAdd,
  className = '',
}) => {
  const components = [
    {
      id: 'deployment',
      name: 'Deployment Stage',
      description: 'Deploy your application to an environment',
      icon: 'rocket_launch',
      category: 'Deployment',
    },
    {
      id: 'test',
      name: 'Test Stage',
      description: 'Run automated tests on your code',
      icon: 'bug_report',
      category: 'Testing',
    },
    {
      id: 'build',
      name: 'Build Stage',
      description: 'Compile and package your application',
      icon: 'build',
      category: 'Build',
    },
    {
      id: 'notification',
      name: 'Notification',
      description: 'Send notifications about workflow status',
      icon: 'notifications',
      category: 'Communication',
    },
    {
      id: 'approval',
      name: 'Manual Approval',
      description: 'Require manual approval before proceeding',
      icon: 'how_to_reg',
      category: 'Control',
    },
    {
      id: 'condition',
      name: 'Conditional Step',
      description: 'Execute steps based on conditions',
      icon: 'alt_route',
      category: 'Control',
    },
  ];

  const categories = Array.from(new Set(components.map(c => c.category)));

  const handleAddComponent = (componentType: string) => {
    onNodeAdd(componentType);
  };

  if (!isOpen) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className=""
        onClick={onClose}
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
                      .map((component) => (
                        <button
                          key={component.id}
                          type="button"
                          onClick={() => handleAddComponent(component.id)}
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