import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { componentsListComponentActions, workflowsInvokeNodeExecutionAction } from '../../api-client/sdk.gen'
import { withOrganizationHeader } from '../../utils/withOrganizationHeader'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { Button } from '../ui/button'
import { Label } from '../ui/label'
import { Input } from '../ui/input'
import { Textarea } from '../Textarea/textarea'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '../ui/dialog'
import { showSuccessToast, showErrorToast } from '../../utils/toast'

interface ComponentActionsProps {
  executionId: string
  componentName: string
  executionState: string
}

export const ComponentActions = ({ executionId, componentName, executionState }: ComponentActionsProps) => {
  const [selectedAction, setSelectedAction] = useState<any | null>(null)
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [actionParameters, setActionParameters] = useState<Record<string, any>>({})
  const queryClient = useQueryClient()

  // Fetch available actions for this component
  const { data: actionsData, isLoading } = useQuery({
    queryKey: ['component-actions', componentName],
    queryFn: async () => {
      const response = await componentsListComponentActions(
        withOrganizationHeader({
          path: {
            name: componentName,
          },
        })
      )
      return response.data
    },
  })

  // Mutation to invoke an action
  const invokeActionMutation = useMutation({
    mutationFn: async ({ actionName, parameters }: { actionName: string; parameters: any }) => {
      await workflowsInvokeNodeExecutionAction(
        withOrganizationHeader({
          path: {
            executionId,
            actionName,
          },
          body: {
            parameters,
          },
        })
      )
    },
    onSuccess: (_, variables) => {
      showSuccessToast(`Action "${variables.actionName}" executed successfully`)
      setIsModalOpen(false)
      setSelectedAction(null)
      setActionParameters({})
      // Refetch executions to show updated state
      queryClient.invalidateQueries({ queryKey: ['workflow-node-executions'] })
    },
    onError: (error: any, variables) => {
      showErrorToast(`Failed to execute action "${variables.actionName}": ${error.message}`)
    },
  })

  const handleActionClick = (action: any) => {
    setSelectedAction(action)
    setActionParameters({})
    setIsModalOpen(true)
  }

  const handleCloseModal = () => {
    setIsModalOpen(false)
    setSelectedAction(null)
    setActionParameters({})
  }

  const handleParameterChange = (paramName: string, value: any) => {
    setActionParameters((prev) => ({
      ...prev,
      [paramName]: value,
    }))
  }

  const handleInvokeAction = () => {
    if (!selectedAction) return
    invokeActionMutation.mutate({ actionName: selectedAction.name, parameters: actionParameters })
  }

  // Don't show actions section if no actions or execution is finished
  if (isLoading || !actionsData?.actions || actionsData.actions.length === 0) {
    return null
  }

  // Only show actions for started executions
  if (executionState !== 'STATE_STARTED') {
    return null
  }

  return (
    <>
      <div className="space-y-3">
        <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
          Actions
        </div>
        <div className="space-y-2">
          {actionsData.actions.map((action: any) => (
            <button
              key={action.name}
              onClick={() => handleActionClick(action)}
              className="w-full px-4 py-3 flex items-center gap-2 bg-white dark:bg-zinc-800 hover:bg-gray-50 dark:hover:bg-zinc-700/50 border border-gray-200 dark:border-zinc-700 rounded-lg transition-colors"
            >
              <MaterialSymbol name="play_circle" size="sm" className="text-blue-600 dark:text-blue-400" />
              <div className="text-left flex-1">
                <div className="text-sm font-medium text-gray-900 dark:text-zinc-100">
                  {action.name}
                </div>
                <div className="text-xs text-gray-500 dark:text-zinc-400">
                  {action.description}
                </div>
              </div>
            </button>
          ))}
        </div>
      </div>

      {/* Action Modal */}
      <Dialog open={isModalOpen} onOpenChange={(open) => !open && handleCloseModal()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{selectedAction?.name}</DialogTitle>
            <DialogDescription>
              {selectedAction?.description}
            </DialogDescription>
          </DialogHeader>
          {selectedAction?.parameters && selectedAction.parameters.length > 0 ? (
            <div className="space-y-4">
              {selectedAction.parameters.map((param: any) => (
                <div key={param.name} className="space-y-2">
                  <Label>
                    {param.name}
                    {param.required && <span className="text-red-500 ml-1">*</span>}
                  </Label>
                  {param.description && (
                    <div className="text-xs text-gray-500 dark:text-zinc-400">
                      {param.description}
                    </div>
                  )}
                  {param.type === 'string' ? (
                    param.name.includes('reason') || param.name.includes('comment') || param.name.includes('message') ? (
                      <Textarea
                        value={actionParameters[param.name] || ''}
                        onChange={(e) => handleParameterChange(param.name, e.target.value)}
                        placeholder={`Enter ${param.name}`}
                        rows={3}
                      />
                    ) : (
                      <Input
                        type="text"
                        value={actionParameters[param.name] || ''}
                        onChange={(e) => handleParameterChange(param.name, e.target.value)}
                        placeholder={`Enter ${param.name}`}
                      />
                    )
                  ) : param.type === 'number' ? (
                    <Input
                      type="number"
                      value={actionParameters[param.name] || ''}
                      onChange={(e) => handleParameterChange(param.name, parseFloat(e.target.value))}
                      placeholder={`Enter ${param.name}`}
                    />
                  ) : (
                    <Input
                      type="text"
                      value={actionParameters[param.name] || ''}
                      onChange={(e) => handleParameterChange(param.name, e.target.value)}
                      placeholder={`Enter ${param.name}`}
                    />
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div className="text-sm text-gray-500 dark:text-zinc-400">
              No parameters required
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={handleCloseModal}>
              Cancel
            </Button>
            <Button
              onClick={handleInvokeAction}
              disabled={invokeActionMutation.isPending}
            >
              {invokeActionMutation.isPending ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white" />
                  Executing...
                </>
              ) : (
                <>
                  <MaterialSymbol name="send" size="sm" />
                  Execute
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
