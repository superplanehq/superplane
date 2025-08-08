import React, { useState } from 'react';
import { ComponentSidebarProps } from '../types';
import { SidebarItem } from './lib/SidebarItem';
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol';
import { Button } from './lib/Button/button';
import { Input } from './lib/Input/input';
import { InputGroup } from './lib/Input/input';
import { Text } from './lib/Text/text';
import { Badge } from './lib/Badge/badge';
import { Dropdown, DropdownButton, DropdownMenu, DropdownItem } from './lib/Dropdown/dropdown';
import { Switch } from './lib/Switch/switch';
import { Field, Label } from './lib/Fieldset/fieldset';
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
  // Get URL parameters
  const urlParams = new URLSearchParams(window.location.search);
  const withSubtitle = urlParams.get('withSubtitle') === 'true';
  const soonSwitch = urlParams.get('soonSwitch') === 'true';
  
  // State for filtering components
  const [filterType, setFilterType] = useState<'all' | 'available' | 'coming-soon'>('all');
  const [hideComingSoon, setHideComingSoon] = useState(false);
  const [showConfigPanel, setShowConfigPanel] = useState(false);
  const components = [
    // Stages
    {
      id: 'semaphore-stage',
      name: 'Semaphore Stage',
      description: 'Run CI/CD pipeline',
      icon: 'semaphore',
      category: 'Stages',
      category_description: 'Execute remote operations',
    },
    {
      id: 'http-stage',
      name: 'HTTP Stage',
      description: 'Make HTTP requests to external services',
      icon: 'http',
      category: 'Stages',
      category_description: 'Execute remote operations',
    },
    {
      id: 'docker-stage',
      name: 'Docker Build',
      description: 'Build and push Docker containers',
      icon: 'package',
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'kubernetes-stage',
      name: 'Kubernetes Deploy',
      description: 'Deploy applications to Kubernetes',
      icon: 'cloud_upload',
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'terraform-stage',
      name: 'Terraform Apply',
      description: 'Provision infrastructure with Terraform',
      icon: 'construction',
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'aws-stage',
      name: 'AWS CLI',
      description: 'Execute AWS CLI commands',
      icon: 'cloud',
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'git-stage',
      name: 'Git Operations',
      description: 'Perform Git operations and version control',
      icon: 'code_blocks',
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'npm-stage',
      name: 'NPM Build',
      description: 'Build Node.js applications with NPM',
      icon: 'javascript',
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'pytest-stage',
      name: 'Python Tests',
      description: 'Run Python tests with pytest',
      icon: 'bug_report',
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'helm-stage',
      name: 'Helm Deploy',
      description: 'Deploy applications with Helm charts',
      icon: 'sailing',
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'ansible-stage',
      name: 'Ansible Playbook',
      description: 'Run Ansible playbooks for configuration',
      icon: 'settings',
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    {
      id: 'sonarqube-stage',
      name: 'SonarQube Scan',
      description: 'Code quality analysis with SonarQube',
      icon: 'search',
      category: 'Stages',
      category_description: 'Execute remote operations',
      comingSoon: true,
    },
    // Event Sources
    {
      id: 'webhook-event',
      name: 'Webhook Event Source',
      description: 'Trigger workflows from webhook events',
      icon: 'webhook',
      category: 'Event Sources',
      category_description: 'Emit events that can be used to trigger executions',
    },
    {
      id: 'semaphore-event',
      name: 'Semaphore Event Source',
      description: 'Trigger workflows from Semaphore events',
      icon: 'semaphore',
      category: 'Event Sources',
      category_description: 'Emit events that can be used to trigger executions',
    },
    // Groups
    {
      id: 'connection-group',
      name: 'Connection Group',
      description: 'Group related workflow connections',
      icon: 'account_tree',
      category: 'Groups',
      category_description: 'Group related workflow connections',
    },
  ];

  const categories = Array.from(new Set(components.map(c => c.category)));

  const handleAddComponent = (componentType: string) => {
    onNodeAdd(componentType);
  };

  if (!isOpen) return null;

  return (
    <>
    

        {/* Sidebar */}
        <div
          className={`bg-white absolute top-0 bottom-0 transform transition-transform duration-300 ease-in-out ${className}`}
          role="dialog"
          aria-modal="true"
          aria-labelledby="sidebar-title"
        >
          <div className="flex flex-col h-full">
            {/* Header */}
            <div className="flex items-center justify-between px-4 pt-4 pb-0 sticky top-0">
              <div className="flex items-center gap-3">
              <Button 
                color='white'

                onClick={onClose}
                aria-label="Close sidebar"
                className='!px-1 !py-0'
              >
                <MaterialSymbol name="menu_open" size="lg" />
              </Button>
              
                <h2 id="sidebar-title" className="text-md font-semibold text-gray-900">
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
            {/* Configuration Panel - shown when Configure Components button is clicked */}
            {showConfigPanel && (
              <div className="px-4 pt-2">
                
                  <Field className="flex items-center text-xs ">
                   <Switch
                      checked={!hideComingSoon}
                      onChange={() => setHideComingSoon(!hideComingSoon)}
                      aria-label="Show coming soon components"
                      color='blue'
                    />
                    <span className="text-sm text-gray-600 dark:text-gray-400 ml-2">Include components with </span>
                    <Badge color="indigo" className='mx-2'>
                      Soon
                    </Badge>
                   
                    
                  </Field>
                
              </div>
            )}
            {/* Search */}
            <div className="p-4 pb-0">
                <InputGroup className='flex items-center'>
                  <MaterialSymbol name="search" size="md" data-slot="icon"/>
                  <Input name="search" placeholder="Search&hellip;" aria-label="Search"/>
                </InputGroup>
            </div>

            

            

            {/* Component Categories */}
            <div className="flex-1 overflow-y-auto">
              {categories.map((category) => (
                <div key={category} className="p-4">
                  <h3 className="text-sm font-medium text-gray-500 uppercase tracking-wide mb-1">
                    {category}
                  </h3>
                  <Text className="!text-xs text-gray-500 mb-3">
                    {!withSubtitle && (
                      components.find(c => c.category === category)?.category_description
                    )} 
                  </Text>
                  <div className="space-y-1">
                    {components
                      .filter(c => c.category === category)
                      .filter(c => {
                        // If using switch mode (soonSwitch=true), use hideComingSoon state
                        
                        return !hideComingSoon || !c.comingSoon;
                        
                      })
                      .map((component) => (
                        <SidebarItem
                          key={component.id}
                          title={component.name}
                          subtitle={component.description}
                          comingSoon={component.comingSoon}
                          showSubtitle={withSubtitle}
                          icon={
                            component.icon === 'semaphore' ? (
                              <img width={24} height={24} src='/images/semaphore-logo-sign-black.svg' alt="Semaphore" className="flex-shrink-0" />
                            ) : component.icon === 'package' ? (
                              <img width={24} height={24} src='/images/docker-logo.svg' alt="Docker" className="flex-shrink-0" />
                            ) : component.icon === 'cloud_upload' ? (
                              <img width={24} height={24} src='/images/kubernetes-logo.svg' alt="Kubernetes" className="flex-shrink-0" />
                            ) : component.icon === 'construction' ? (
                              <img width={24} height={24} src='/images/terraform-logo.svg' alt="Terraform" className="flex-shrink-0" />
                            ) : component.icon === 'cloud' ? (
                              <img width={24} height={24} src='/images/aws-logo.svg' alt="AWS" className="flex-shrink-0" />
                            ) : component.icon === 'code_blocks' ? (
                              <img width={24} height={24} src='/images/git-logo.svg' alt="Git" className="flex-shrink-0" />
                            ) : component.icon === 'javascript' ? (
                              <img width={24} height={24} src='/images/npm-logo.svg' alt="NPM" className="flex-shrink-0" />
                            ) : component.icon === 'bug_report' ? (
                              <img width={24} height={24} src='/images/python-logo.svg' alt="Python" className="flex-shrink-0" />
                            ) : component.icon === 'sailing' ? (
                              <img width={24} height={24} src='/images/helm-logo.svg' alt="Helm" className="flex-shrink-0" />
                            ) : component.icon === 'settings' ? (
                              <img width={24} height={24} src='/images/ansible-logo.svg' alt="Ansible" className="flex-shrink-0" />
                            ) : component.icon === 'search' ? (
                              <img width={24} height={24} src='/images/sonarqube-logo.svg' alt="SonarQube" className="flex-shrink-0" />
                            ) : component.icon === 'http' ? (
                              <MaterialSymbol name="rocket_launch" size="lg" className="text-gray-600 flex-shrink-0" />
                            ) : component.icon === 'webhook' ? (
                              <MaterialSymbol name="webhook" size="lg" className="text-gray-600 flex-shrink-0" />
                            ) : component.icon === 'account_tree' ? (
                              <MaterialSymbol name="account_tree" size="lg" className="text-gray-600 flex-shrink-0" />
                            ) : (
                              <MaterialSymbol name="drag_indicator" size="lg" className="text-gray-600 flex-shrink-0" />
                            )
                          }
                          onClickAddNode={component.comingSoon ? undefined : () => handleAddComponent(component.id)}
                          className={component.comingSoon ? "" : "cursor-pointer hover:bg-blue-50 border border-gray-200"}
                        />
                      ))}
                  </div>
                </div>
              ))}
            </div>

          </div>
        </div>
      
    </>
  );
};