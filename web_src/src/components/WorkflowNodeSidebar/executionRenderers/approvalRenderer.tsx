import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { registerExecutionRenderer, CollapsedViewProps, ExpandedViewProps } from './registry'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Button } from '../../Button/button'
import { Dialog, DialogTitle, DialogDescription, DialogBody, DialogActions } from '../../Dialog/dialog'
import { Field, Label } from '../../Fieldset/fieldset'
import { Textarea } from '../../Textarea/textarea'
import { workflowsInvokeNodeExecutionAction } from '../../../api-client/sdk.gen'
import { withOrganizationHeader } from '../../../utils/withOrganizationHeader'
import { showSuccessToast, showErrorToast } from '../../../utils/toast'

const formatTimeAgo = (date: Date): string => {
  const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000)

  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

const ApprovalActions = ({ execution }: { execution: any }) => {
  const [isApproveModalOpen, setIsApproveModalOpen] = useState(false)
  const [isRejectModalOpen, setIsRejectModalOpen] = useState(false)
  const [comment, setComment] = useState('')
  const [reason, setReason] = useState('')
  const queryClient = useQueryClient()

  const invokeActionMutation = useMutation({
    mutationFn: async ({ actionName, parameters }: { actionName: string; parameters: any }) => {
      await workflowsInvokeNodeExecutionAction(
        withOrganizationHeader({
          path: {
            executionId: execution.id,
            actionName,
          },
          body: {
            parameters,
          },
        })
      )
    },
    onSuccess: (_, variables) => {
      showSuccessToast(`Successfully ${variables.actionName === 'approve' ? 'approved' : 'rejected'}`)
      setIsApproveModalOpen(false)
      setIsRejectModalOpen(false)
      setComment('')
      setReason('')
      queryClient.invalidateQueries({ queryKey: ['workflow-node-executions'] })
    },
    onError: (error: any) => {
      showErrorToast(`Failed to execute action: ${error.message}`)
    },
  })

  const handleApprove = () => {
    invokeActionMutation.mutate({
      actionName: 'approve',
      parameters: { comment: comment || undefined }
    })
  }

  const handleReject = () => {
    if (!reason.trim()) {
      showErrorToast('Reason is required for rejection')
      return
    }
    invokeActionMutation.mutate({
      actionName: 'reject',
      parameters: { reason }
    })
  }

  // Only show actions for waiting or started executions
  if (execution.state !== 'STATE_WAITING' && execution.state !== 'STATE_STARTED') {
    return null
  }

  return (
    <>
      <div className="space-y-3">
        <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
          Actions
        </div>
        <div className="flex gap-2">
          <Button
            onClick={() => setIsApproveModalOpen(true)}
            color="green"
            className="flex-1"
          >
            <MaterialSymbol name="check_circle" size="sm" />
            Approve
          </Button>
          <Button
            onClick={() => setIsRejectModalOpen(true)}
            color="red"
            className="flex-1"
          >
            <MaterialSymbol name="cancel" size="sm" />
            Reject
          </Button>
        </div>
      </div>

      {/* Approve Modal */}
      <Dialog open={isApproveModalOpen} onClose={() => setIsApproveModalOpen(false)}>
        <DialogTitle>Approve Execution</DialogTitle>
        <DialogDescription>
          Add an optional comment for this approval
        </DialogDescription>
        <DialogBody>
          <Field>
            <Label>Comment (optional)</Label>
            <Textarea
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              placeholder="Enter an optional comment"
              rows={3}
            />
          </Field>
        </DialogBody>
        <DialogActions>
          <Button plain onClick={() => setIsApproveModalOpen(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleApprove}
            disabled={invokeActionMutation.isPending}
            color="green"
          >
            {invokeActionMutation.isPending ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white" />
                Approving...
              </>
            ) : (
              <>
                <MaterialSymbol name="check_circle" size="sm" />
                Approve
              </>
            )}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Reject Modal */}
      <Dialog open={isRejectModalOpen} onClose={() => setIsRejectModalOpen(false)}>
        <DialogTitle>Reject Execution</DialogTitle>
        <DialogDescription>
          Provide a reason for rejecting this execution
        </DialogDescription>
        <DialogBody>
          <Field>
            <Label>
              Reason <span className="text-red-500">*</span>
            </Label>
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Enter a reason for rejection"
              rows={3}
            />
          </Field>
        </DialogBody>
        <DialogActions>
          <Button plain onClick={() => setIsRejectModalOpen(false)}>
            Cancel
          </Button>
          <Button
            onClick={handleReject}
            disabled={invokeActionMutation.isPending}
            color="red"
          >
            {invokeActionMutation.isPending ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white" />
                Rejecting...
              </>
            ) : (
              <>
                <MaterialSymbol name="cancel" size="sm" />
                Reject
              </>
            )}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}

// Custom renderer for Approval primitive executions
registerExecutionRenderer('approval', {
  renderCollapsed: ({ execution, onClick }: CollapsedViewProps) => {
    const metadata = execution.metadata || {}
    const requiredCount = metadata.required_count || 0
    const approvals = metadata.approvals || []
    const isComplete = approvals.length >= requiredCount
    const isRejected = execution.state === 'STATE_FINISHED' && execution.result === 'RESULT_FAILED'

    // Determine label and styling
    let label = ''
    let iconName = 'pending'
    let colorClasses = 'text-orange-600 dark:text-orange-400 bg-orange-100 dark:bg-orange-900/30'
    let badgeClasses = 'bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300'

    if (isRejected) {
      label = 'Rejected'
      iconName = 'cancel'
      colorClasses = 'text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-900/30'
      badgeClasses = 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300'
    } else if (isComplete) {
      label = 'Approved'
      iconName = 'check_circle'
      colorClasses = 'text-green-600 dark:text-green-400 bg-green-100 dark:bg-green-900/30'
      badgeClasses = 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300'
    } else {
      label = `${approvals.length}/${requiredCount} Approvals`
      iconName = 'pending'
      colorClasses = 'text-orange-600 dark:text-orange-400 bg-orange-100 dark:bg-orange-900/30'
      badgeClasses = 'bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300'
    }

    return (
      <div
        className="flex items-start gap-3 cursor-pointer"
        onClick={onClick}
      >
        <div className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center ${colorClasses}`}>
          <MaterialSymbol name={iconName} size="sm" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2">
            <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${badgeClasses}`}>
              {label}
            </span>
          </div>

          <p className="text-xs font-mono text-gray-600 dark:text-zinc-400 truncate mb-1">
            {execution.id}
          </p>

          <p className="text-xs text-gray-400 dark:text-zinc-500">
            {formatTimeAgo(new Date(execution.createdAt))}
          </p>
        </div>
        <div className="flex-shrink-0">
          <MaterialSymbol
            name="expand_more"
            size="xl"
            className="text-gray-600 dark:text-zinc-400"
          />
        </div>
      </div>
    )
  },

  renderExpanded: ({ execution, isDarkMode }: ExpandedViewProps) => {
    const metadata = execution.metadata || {}
    const approvals = metadata.approvals || []

    return (
      <div className="space-y-4 text-left">
        {/* Inputs Section */}
        {execution.inputs && Object.keys(execution.inputs).length > 0 && (
          <>
            <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
              Inputs
            </div>
            <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs text-left">
              <pre className="text-gray-800 dark:text-gray-200 font-mono text-xs overflow-x-auto">
                {JSON.stringify(execution.inputs, null, 2)}
              </pre>
            </div>
          </>
        )}

        {/* Approval Actions */}
        <ApprovalActions execution={execution} />

        {/* Approvals List */}
        {approvals.length > 0 && (
          <>
            <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
              Approvals ({approvals.length})
            </div>
            <div className="space-y-2">
              {approvals.map((approval: any, index: number) => (
                <div
                  key={index}
                  className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 rounded-lg p-3 text-xs"
                >
                  <div className="flex items-start gap-2">
                    <MaterialSymbol
                      name="check_circle"
                      size="md"
                      className="text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5"
                    />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="font-semibold text-gray-900 dark:text-zinc-100">
                          Approval #{index + 1}
                        </span>
                        {approval.approved_at && (
                          <span className="text-gray-500 dark:text-zinc-400">
                            {new Date(approval.approved_at).toLocaleString()}
                          </span>
                        )}
                      </div>
                      {approval.comment && (
                        <div className="text-gray-700 dark:text-zinc-300 bg-white dark:bg-zinc-900 rounded px-2 py-1 mt-2">
                          {approval.comment}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    )
  }
})
