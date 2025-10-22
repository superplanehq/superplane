import { useState } from 'react'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { Button } from '../ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '../ui/tabs'
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
  componentLabel?: string
  organizationId: string
  blueprintId?: string
}

export const WorkflowNodeSidebar = ({ workflowId, nodeId, onClose, isBlueprintNode, nodeType, organizationId, blueprintId }: WorkflowNodeSidebarProps) => {
  const [activeTab, setActiveTab] = useState<Tab>('queue')

  return (
    <div className="bg-white dark:bg-zinc-900 border-l border-zinc-200 dark:border-zinc-800 flex flex-col z-50 h-full">
      {/* Header with tabs and close button */}
      <div className="flex items-center justify-between px-4 pt-4 pb-2 gap-4">
        <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as Tab)} className="flex-1">
          <TabsList className="w-full">
            <TabsTrigger value="queue" className="flex-1">
              Queue
            </TabsTrigger>
            <TabsTrigger value="executions" className="flex-1">
              Executions
            </TabsTrigger>
          </TabsList>
        </Tabs>
        <Button variant="ghost" size="icon-sm" onClick={onClose} className="flex-shrink-0">
          <MaterialSymbol name="close" />
        </Button>
      </div>

      {/* Tab Content */}
      <div className="flex-1 overflow-hidden">
        <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as Tab)} className="h-full flex flex-col">
          <TabsContent value="queue" className="flex-1 overflow-hidden data-[state=inactive]:hidden">
            <WorkflowNodeQueueTab workflowId={workflowId} nodeId={nodeId} />
          </TabsContent>
          <TabsContent value="executions" className="flex-1 overflow-hidden data-[state=inactive]:hidden">
            <WorkflowNodeExecutionsTab
              workflowId={workflowId}
              nodeId={nodeId}
              isBlueprintNode={isBlueprintNode}
              nodeType={nodeType}
              organizationId={organizationId}
              blueprintId={blueprintId}
            />
          </TabsContent>
        </Tabs>
      </div>
    </div>
  )
}
