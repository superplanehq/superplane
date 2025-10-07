import { useState } from 'react'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { Button } from '../Button/button'
import { WorkflowNodeQueueTab } from './WorkflowNodeQueueTab'
import { WorkflowNodeExecutionsTab } from './WorkflowNodeExecutionsTab'

type Tab = 'queue' | 'executions'

interface WorkflowNodeSidebarProps {
  workflowId: string
  nodeId: string
  nodeName: string
  onClose: () => void
  isBlueprintNode?: boolean
  nodeType?: string
}

export const WorkflowNodeSidebar = ({ workflowId, nodeId, nodeName, onClose, isBlueprintNode, nodeType }: WorkflowNodeSidebarProps) => {
  const [activeTab, setActiveTab] = useState<Tab>('queue')

  return (
    <div className="w-96 bg-white dark:bg-zinc-900 border-l border-zinc-200 dark:border-zinc-800 flex flex-col z-50 h-full">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-zinc-200 dark:border-zinc-800">
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <MaterialSymbol name="widgets" size="lg" className="text-zinc-600 dark:text-zinc-400 flex-shrink-0" />
          <div className="min-w-0 flex-1">
            <h2 className="text-sm font-semibold text-gray-900 dark:text-zinc-100 truncate">
              {nodeName}
            </h2>
            <p className="text-xs text-gray-500 dark:text-zinc-400 truncate">
              {nodeId}
            </p>
          </div>
        </div>
        <Button plain onClick={onClose} className="flex-shrink-0">
          <MaterialSymbol name="close" />
        </Button>
      </div>

      {/* Tabs */}
      <div className="flex border-b border-zinc-200 dark:border-zinc-700">
        <button
          onClick={() => setActiveTab('queue')}
          className={`flex-1 px-4 py-3 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === 'queue'
              ? 'text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400 bg-blue-50 dark:bg-blue-950/20'
              : 'text-gray-500 dark:text-zinc-400 hover:text-gray-700 dark:hover:text-zinc-300 hover:bg-gray-50 dark:hover:bg-zinc-800/50'
          }`}
        >
          <MaterialSymbol name="queue" size="sm" />
          Queue
        </button>
        <button
          onClick={() => setActiveTab('executions')}
          className={`flex-1 px-4 py-3 text-sm font-medium transition-colors flex items-center justify-center gap-2 ${
            activeTab === 'executions'
              ? 'text-blue-600 dark:text-blue-400 border-b-2 border-blue-600 dark:border-blue-400 bg-blue-50 dark:bg-blue-950/20'
              : 'text-gray-500 dark:text-zinc-400 hover:text-gray-700 dark:hover:text-zinc-300 hover:bg-gray-50 dark:hover:bg-zinc-800/50'
          }`}
        >
          <MaterialSymbol name="history" size="sm" />
          Executions
        </button>
      </div>

      {/* Tab Content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === 'queue' && (
          <WorkflowNodeQueueTab workflowId={workflowId} nodeId={nodeId} />
        )}
        {activeTab === 'executions' && (
          <WorkflowNodeExecutionsTab workflowId={workflowId} nodeId={nodeId} isBlueprintNode={isBlueprintNode} nodeType={nodeType} />
        )}
      </div>
    </div>
  )
}
