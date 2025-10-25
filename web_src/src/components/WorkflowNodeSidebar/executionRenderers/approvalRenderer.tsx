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
import { formatTimeAgo } from '../../../utils/date'

interface Item {
  type: string
  user?: string
  role?: string
  group?: string
}

interface ItemRecord extends Item {
  index: number
  state: string
  at?: string
  by?: User
  comment?: string
}

interface User {
  id: string
  name: string
}

interface Metadata {
  records?: ItemRecord[]
}

const RecordItemCard = ({
  record,
  index,
  execution,
  workflowId,
}: {
  record: ItemRecord
  index: number
  execution: any
  workflowId: string
}) => {
  const [isApproveModalOpen, setIsApproveModalOpen] = useState(false)
  const [isRejectModalOpen, setIsRejectModalOpen] = useState(false)
  const [comment, setComment] = useState('')
  const [reason, setReason] = useState('')
  const queryClient = useQueryClient()
  const isApproved = record.state === 'approved'

  const invokeActionMutation = useMutation({
    mutationFn: async ({ actionName, parameters }: { actionName: string; parameters: any }) => {
      await workflowsInvokeNodeExecutionAction(
        withOrganizationHeader({
          path: {
            workflowId,
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
      const action = variables.actionName === 'approve' ? 'approved' : 'rejected'
      showSuccessToast(`Successfully ${action}`)
      setIsApproveModalOpen(false)
      setIsRejectModalOpen(false)
      setComment('')
      setReason('')
      queryClient.invalidateQueries({ queryKey: ['workflow-node-executions'] })
    },
    onError: (error: any, variables) => {
      const action = variables.actionName === 'approve' ? 'approve' : 'reject'
      showErrorToast(`Failed to ${action}: ${error.message}`)
    },
  })

  const handleApprove = () => {
    invokeActionMutation.mutate({
      actionName: 'approve',
      parameters: {
        index: index,
        comment: comment || undefined
      }
    })
  }

  const handleReject = () => {
    if (!reason.trim()) {
      showErrorToast('Reason is required for rejection')
      return
    }
    invokeActionMutation.mutate({
      actionName: 'reject',
      parameters: {
        index: index,
        reason: reason
      }
    })
  }

  const getRecordLabel = () => {
    if (record.type === 'user' && record.user) {
      return `User: ${record.user}`
    }
    if (record.type === 'role' && record.role) {
      return `Role: ${record.role}`
    }
    if (record.type === 'group' && record.group) {
      return `Group: ${record.group}`
    }

    return `Record #${index + 1}`
  }

  return (
    <>
      <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 rounded-lg p-3">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            {isApproved ? (
              <MaterialSymbol name="check_circle" className="text-green-600 dark:text-green-400" />
            ) : (
              <MaterialSymbol name="pending" className="text-orange-600 dark:text-orange-400" />
            )}
            <div>
              <div className="text-sm font-semibold text-gray-900 dark:text-zinc-100">
                {getRecordLabel()}
              </div>
              {record.state == 'approved' && (
                <div className="text-xs text-gray-500 dark:text-zinc-400">
                  Approved by {record.by?.name} {record.at && `at ${new Date(record.at).toLocaleString()}`}
                </div>
              )}
            </div>
          </div>
          {record.state == "pending" && execution.state === 'STATE_STARTED' && (
            <div className="flex gap-2">
              <Button
                onClick={() => setIsApproveModalOpen(true)}
                size="sm"
                className="bg-green-600 hover:bg-green-700 text-white"
              >
                <MaterialSymbol name="check_circle" size="sm" />
                Approve
              </Button>
              <Button
                onClick={() => setIsRejectModalOpen(true)}
                size="sm"
                variant="destructive"
              >
                <MaterialSymbol name="cancel" size="sm" />
                Reject
              </Button>
            </div>
          )}
        </div>
        {record?.comment && (
          <div className="text-xs text-gray-700 dark:text-zinc-300 bg-white dark:bg-zinc-900 rounded px-2 py-1 mt-2">
            {record.comment}
          </div>
        )}
      </div>

      {/* Approve Modal */}
      <Dialog open={isApproveModalOpen} onOpenChange={(open) => !open && setIsApproveModalOpen(false)}>
        <DialogContent className="max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Approve: {getRecordLabel()}</DialogTitle>
            <DialogDescription>
              Add an optional comment for this approval
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">

            {/* Comment Field */}
            <div className="space-y-2">
              <Label>Comment (optional)</Label>
              <Textarea
                value={comment}
                onChange={(e) => setComment(e.target.value)}
                placeholder="Enter an optional comment"
                rows={3}
              />
            </div>
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
            <DialogTitle>Reject: {getRecordLabel()}</DialogTitle>
            <DialogDescription>
              Provide a reason for rejecting this approval requirement
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

registerExecutionRenderer('approval', {
  renderCollapsed: ({ execution, onClick, isExpanded }: CollapsedViewProps) => {
    const metadata = (execution.metadata || {}) as Metadata
    const records = metadata.records || []

    const approvedCount = records.reduce((count, record) => {
      if (record.state === 'approved') {
        return count + 1
      }
      return count
    }, 0)

    const requiredCount = records.length
    const isComplete = approvedCount >= requiredCount && requiredCount > 0
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
      label = 'Pending'
      iconName = 'pending'
      badgeClassName = 'bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 border-orange-200 dark:border-orange-800'
    }

    return (
      <div
        className="flex items-center gap-3 cursor-pointer"
        onClick={onClick}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 text-sm">
            <Badge variant={badgeVariant} className={badgeClassName}>
              <MaterialSymbol name={iconName} size="sm" />
              {label}
            </Badge>
            <span className="font-semibold text-gray-700 dark:text-gray-300">
              {approvedCount}/{requiredCount}
            </span>
          </div>
        </div>
        <div className="flex-shrink-0 flex items-center gap-2">
          <p className="text-xs text-gray-400 dark:text-zinc-500">
            {formatTimeAgo(new Date(execution.createdAt!))}
          </p>
          <MaterialSymbol
            name={isExpanded ? 'expand_less' : 'expand_more'}
            size="xl"
            className="text-gray-600 dark:text-zinc-400"
          />
        </div>
      </div>
    )
  },

  renderExpanded: ({ execution, workflowId }: ExpandedViewProps) => {
    const metadata = (execution.metadata || {}) as Metadata
    const records = metadata.records || []

    return (
      <div className="space-y-4 text-left">
        {/* Inputs Section */}
        {execution.input && Object.keys(execution.input).length > 0 && (
          <>
            <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
              Inputs
            </div>
            <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs text-left">
              <pre className="text-gray-800 dark:text-gray-200 font-mono text-xs overflow-x-auto">
                {JSON.stringify(execution.input, null, 2)}
              </pre>
            </div>
          </>
        )}

        {/* Records */}
        {records.length > 0 && (
          <>
            <div className="space-y-2">
              {records.map((record, index) => (
                <RecordItemCard
                  key={index}
                  index={index}
                  record={record}
                  execution={execution}
                  workflowId={workflowId}
                />
              ))}
            </div>
          </>
        )}
      </div>
    )
  }
})
