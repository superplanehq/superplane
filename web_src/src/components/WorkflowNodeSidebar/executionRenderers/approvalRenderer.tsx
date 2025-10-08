import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { registerExecutionRenderer, CollapsedViewProps, ExpandedViewProps } from './registry'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Button } from '../../ui/button'
import { Badge } from '../../ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '../../ui/dialog'
import { Label } from '../../ui/label'
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
            className="flex-1 bg-green-600 hover:bg-green-700 text-white"
          >
            <MaterialSymbol name="check_circle" size="sm" />
            Approve
          </Button>
          <Button
            onClick={() => setIsRejectModalOpen(true)}
            variant="destructive"
            className="flex-1"
          >
            <MaterialSymbol name="cancel" size="sm" />
            Reject
          </Button>
        </div>
      </div>

      {/* Approve Modal */}
      <Dialog open={isApproveModalOpen} onOpenChange={(open) => !open && setIsApproveModalOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Approve Execution</DialogTitle>
            <DialogDescription>
              Add an optional comment for this approval
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>Comment (optional)</Label>
            <Textarea
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              placeholder="Enter an optional comment"
              rows={3}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsApproveModalOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleApprove}
              disabled={invokeActionMutation.isPending}
              className="bg-green-600 hover:bg-green-700 text-white"
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
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Reject Modal */}
      <Dialog open={isRejectModalOpen} onOpenChange={(open) => !open && setIsRejectModalOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Reject Execution</DialogTitle>
            <DialogDescription>
              Provide a reason for rejecting this execution
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label>
              Reason <span className="text-red-500">*</span>
            </Label>
            <Textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Enter a reason for rejection"
              rows={3}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setIsRejectModalOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleReject}
              disabled={invokeActionMutation.isPending}
              variant="destructive"
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
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

// Custom renderer for Approval component executions
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
    let badgeVariant: 'default' | 'secondary' | 'destructive' | 'outline' = 'secondary'
    let badgeClassName = 'bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 border-orange-200 dark:border-orange-800'

    if (isRejected) {
      label = 'Rejected'
      iconName = 'cancel'
      badgeVariant = 'destructive'
      badgeClassName = ''
    } else if (isComplete) {
      label = 'Approved'
      iconName = 'check_circle'
      badgeClassName = 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 border-green-200 dark:border-green-800'
    } else {
      label = `${approvals.length}/${requiredCount} Approvals`
      iconName = 'pending'
      badgeClassName = 'bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 border-orange-200 dark:border-orange-800'
    }

    return (
      <div
        className="flex items-center gap-3 cursor-pointer"
        onClick={onClick}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <Badge variant={badgeVariant} className={badgeClassName}>
              <MaterialSymbol name={iconName} size="sm" />
              {label}
            </Badge>
          </div>
        </div>
        <div className="flex-shrink-0 flex items-center gap-2">
          <p className="text-xs text-gray-400 dark:text-zinc-500">
            {formatTimeAgo(new Date(execution.createdAt))}
          </p>
          <MaterialSymbol
            name="expand_more"
            size="xl"
            className="text-gray-600 dark:text-zinc-400"
          />
        </div>
      </div>
    )
  },

  renderExpanded: ({ execution }: ExpandedViewProps) => {
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
