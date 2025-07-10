import React, { useState } from 'react'
import { Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Dropdown, DropdownButton, DropdownMenu, DropdownItem, DropdownLabel } from './lib/Dropdown/dropdown'
import { Dialog, DialogTitle, DialogDescription, DialogBody, DialogActions } from './lib/Dialog/dialog'
import { Input } from './lib/Input/input'

interface WorkflowPageProps {
  onBack?: () => void
  onOpenWorkflow?: (workflowId: string, workflowName: string) => void
}


export function WorkflowPage({ onBack, onOpenWorkflow }: WorkflowPageProps) {
  const [isCreateWorkflowOpen, setIsCreateWorkflowOpen] = useState(false)
  const [workflowName, setWorkflowName] = useState('')
  const [workflowDescription, setWorkflowDescription] = useState('')

  // Sample saved workflows
  const savedWorkflows = [
    {
      id: '1',
      name: 'Customer Onboarding',
      description: 'Automated workflow for new customer registration and setup',
      status: 'active',
      lastModified: '2 hours ago',
      nodeCount: 5,
      executions: 342
    },
    {
      id: '2',
      name: 'Invoice Processing',
      description: 'Process and validate incoming invoices automatically',
      status: 'active',
      lastModified: '1 day ago',
      nodeCount: 4,
      executions: 156
    },
    {
      id: '3',
      name: 'Employee Offboarding',
      description: 'Handle employee departures and access revocation',
      status: 'draft',
      lastModified: '3 days ago',
      nodeCount: 7,
      executions: 0
    },
    {
      id: '4',
      name: 'Deployment Pipeline',
      description: 'CI/CD pipeline for application deployment',
      status: 'active',
      lastModified: '1 week ago',
      nodeCount: 6,
      executions: 89
    }
  ]

  // Sample workflow templates
  const workflowTemplates = [
    {
      id: 'customer-onboarding',
      name: 'Customer Onboarding',
      description: 'Automated workflow for new customer registration and setup',
    },
    {
      id: 'deployment-pipeline',
      name: 'Deployment Pipeline',
      description: 'CI/CD pipeline for application deployment',
    },
  ]

  // Load workflow template
  const loadTemplate = (templateId: string) => {
    const template = workflowTemplates.find(t => t.id === templateId)
    if (template && onOpenWorkflow) {
      onOpenWorkflow(`template-${templateId}`, template.name)
      setIsCreateWorkflowOpen(false)
    }
  }

  // Create new workflow
  const createNewWorkflow = () => {
    if (onOpenWorkflow) {
      onOpenWorkflow('new', workflowName || 'New Workflow')
      setWorkflowName('')
      setWorkflowDescription('')
      setIsCreateWorkflowOpen(false)
    }
  }

  // Open existing workflow
  const openWorkflow = (workflow: any) => {
    if (onOpenWorkflow) {
      onOpenWorkflow(workflow.id, workflow.name)
    }
  }

  return (
      <div className="space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div>
            <Subheading level={1} className="mb-2">Workflows</Subheading>
            <Text className="text-zinc-600 dark:text-zinc-400">
              Create and manage automated workflows for your organization
            </Text>
          </div>
          <Button color="blue" onClick={() => setIsCreateWorkflowOpen(true)}>
            <MaterialSymbol name="add" />
            Create Workflow
          </Button>
        </div>

        {/* Workflows Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {savedWorkflows.map((workflow) => (
            <div 
              key={workflow.id} 
              className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700 hover:border-blue-300 dark:hover:border-blue-600 transition-colors cursor-pointer"
              onClick={() => openWorkflow(workflow)}
            >
              <div className="flex items-start justify-between mb-4">
                <div className={`p-2 rounded-lg ${
                  workflow.status === 'active' 
                    ? 'bg-green-50 dark:bg-green-900/20'
                    : 'bg-zinc-50 dark:bg-zinc-900/20'
                }`}>
                  <MaterialSymbol 
                    name="account_tree" 
                    className={
                      workflow.status === 'active'
                        ? 'text-green-600 dark:text-green-400'
                        : 'text-zinc-600 dark:text-zinc-400'
                    }
                    size="lg" 
                  />
                </div>
                <Dropdown>
                  <DropdownButton plain onClick={(e) => e.stopPropagation()}>
                    <MaterialSymbol name="more_vert" size="sm" />
                  </DropdownButton>
                  <DropdownMenu>
                    <DropdownItem onClick={() => openWorkflow(workflow)}>
                      <MaterialSymbol name="edit" />
                      Edit
                    </DropdownItem>
                    <DropdownItem>
                      <MaterialSymbol name="copy" />
                      Duplicate
                    </DropdownItem>
                    <DropdownItem>
                      <MaterialSymbol name="download" />
                      Export
                    </DropdownItem>
                    <DropdownItem>
                      <MaterialSymbol name="delete" />
                      Delete
                    </DropdownItem>
                  </DropdownMenu>
                </Dropdown>
              </div>

              <Subheading level={3} className="mb-2">{workflow.name}</Subheading>
              <Text className="text-zinc-600 dark:text-zinc-400 mb-4 line-clamp-2">
                {workflow.description}
              </Text>

              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-zinc-500 dark:text-zinc-400">
                    {workflow.nodeCount} steps
                  </span>
                  <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                    workflow.status === 'active'
                      ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                      : 'bg-zinc-100 text-zinc-800 dark:bg-zinc-900/20 dark:text-zinc-400'
                  }`}>
                    {workflow.status}
                  </span>
                </div>
                <div className="flex items-center justify-between text-sm text-zinc-500 dark:text-zinc-400">
                  <span>Modified {workflow.lastModified}</span>
                  <span>{workflow.executions} runs</span>
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Create Workflow Modal */}
        <Dialog open={isCreateWorkflowOpen} onClose={() => setIsCreateWorkflowOpen(false)} size="lg">
          <DialogTitle>Create New Workflow</DialogTitle>
          <DialogDescription>
            Start with a blank workflow or choose from one of our templates.
          </DialogDescription>
          <DialogBody>
            <div className="space-y-6">
              {/* Blank Workflow Option */}
              <div>
                <Subheading level={3} className="mb-4">Create from Scratch</Subheading>
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                      Workflow Name *
                    </label>
                    <Input
                      type="text"
                      placeholder="Enter workflow name"
                      value={workflowName}
                      onChange={(e) => setWorkflowName(e.target.value)}
                      className="w-full"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                      Description
                    </label>
                    <Input
                      type="text"
                      placeholder="Describe what this workflow does"
                      value={workflowDescription}
                      onChange={(e) => setWorkflowDescription(e.target.value)}
                      className="w-full"
                    />
                  </div>
                </div>
              </div>

              {/* Template Options */}
              <div>
                <Subheading level={3} className="mb-4">Or Choose a Template</Subheading>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  {workflowTemplates.map((template) => (
                    <button
                      key={template.id}
                      onClick={() => loadTemplate(template.id)}
                      className="p-4 border border-zinc-200 dark:border-zinc-700 rounded-lg hover:border-blue-500 dark:hover:border-blue-400 transition-colors text-left"
                    >
                      <div className="font-medium text-zinc-900 dark:text-white mb-2">
                        {template.name}
                      </div>
                      <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                        {template.description}
                      </Text>
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </DialogBody>
          <DialogActions>
            <Button plain onClick={() => setIsCreateWorkflowOpen(false)}>
              Cancel
            </Button>
            <Button 
              color="blue" 
              onClick={createNewWorkflow}
              disabled={!workflowName.trim()}
            >
              Create Workflow
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    )
  }
