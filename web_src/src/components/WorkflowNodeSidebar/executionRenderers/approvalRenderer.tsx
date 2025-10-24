import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { registerExecutionRenderer, CollapsedViewProps, ExpandedViewProps } from './registry'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Button } from '../../ui/button'
import { Badge } from '../../ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '../../ui/dialog'
import { Label } from '../../ui/label'
import { Textarea } from '../../Textarea/textarea'
import { ConfigurationFieldRenderer } from '../../ConfigurationFieldRenderer'
import { workflowsInvokeNodeExecutionAction } from '../../../api-client/sdk.gen'
import { withOrganizationHeader } from '../../../utils/withOrganizationHeader'
import { showSuccessToast, showErrorToast } from '../../../utils/toast'
import { formatTimeAgo } from '../../../utils/date'
import { ComponentsConfigurationField } from '../../../api-client'

interface ApprovalRequirement {
  type: string
  user?: string
  role?: string
  group?: string
  parameters?: ComponentsConfigurationField[]
}

interface ApprovalRecord {
  requirementIndex: number
  approvedAt?: string
  approvedBy?: string
  comment?: string
  data?: Record<string, any>
}

interface ApprovalMetadata {
  approvals?: ApprovalRecord[]
}

interface ApprovalConfig {
  approvals?: ApprovalRequirement[]
}

const ApprovalRequirementCard = ({
  requirement,
  requirementIndex,
  approval,
  execution,
  workflowId,
  organizationId
}: {
  requirement: ApprovalRequirement
  requirementIndex: number
  approval?: ApprovalRecord
  execution: any
  workflowId: string
  organizationId: string
}) => {
  const [isApproveModalOpen, setIsApproveModalOpen] = useState(false)
  const [isRejectModalOpen, setIsRejectModalOpen] = useState(false)
  const [comment, setComment] = useState('')
  const [reason, setReason] = useState('')
  const [formData, setFormData] = useState<Record<string, any>>({})
  const queryClient = useQueryClient()

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
      setFormData({})
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
        requirementIndex: requirementIndex,
        data: Object.keys(formData).length > 0 ? formData : undefined,
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
        requirementIndex: requirementIndex,
        reason: reason
      }
    })
  }

  // Get the requirement label
  const getRequirementLabel = () => {
    if (requirement.type === 'user' && requirement.user) {
      return `User: ${requirement.user}`
    }
    if (requirement.type === 'role' && requirement.role) {
      return `Role: ${requirement.role}`
    }
    if (requirement.type === 'group' && requirement.group) {
      return `Group: ${requirement.group}`
    }
    return `Requirement #${requirementIndex + 1}`
  }

  const isApproved = !!approval

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
                {getRequirementLabel()}
              </div>
              {isApproved && approval && (
                <div className="text-xs text-gray-500 dark:text-zinc-400">
                  Approved by {approval.approvedBy} {approval.approvedAt && `at ${new Date(approval.approvedAt).toLocaleString()}`}
                </div>
              )}
            </div>
          </div>
          {!isApproved && execution.state === 'STATE_STARTED' && (
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
        {isApproved && approval?.comment && (
          <div className="text-xs text-gray-700 dark:text-zinc-300 bg-white dark:bg-zinc-900 rounded px-2 py-1 mt-2">
            {approval.comment}
          </div>
        )}
        {isApproved && approval?.data && Object.keys(approval.data).length > 0 && (
          <div className="text-xs text-gray-700 dark:text-zinc-300 bg-white dark:bg-zinc-900 rounded px-2 py-1 mt-2">
            <pre className="font-mono text-xs overflow-x-auto">
              {JSON.stringify(approval.data, null, 2)}
            </pre>
          </div>
        )}
      </div>

      {/* Approve Modal */}
      <Dialog open={isApproveModalOpen} onOpenChange={(open) => !open && setIsApproveModalOpen(false)}>
        <DialogContent className="max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Approve: {getRequirementLabel()}</DialogTitle>
            <DialogDescription>
              {requirement.parameters && requirement.parameters.length > 0
                ? 'Fill in the required information for this approval'
                : 'Add an optional comment for this approval'}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {/* Dynamic Parameters */}
            {requirement.parameters && requirement.parameters.length > 0 && (
              <div className="space-y-4">
                {requirement.parameters.map((field) => (
                  <ConfigurationFieldRenderer
                    key={field.name}
                    field={field}
                    value={formData[field.name!]}
                    onChange={(value) => setFormData({ ...formData, [field.name!]: value })}
                    allValues={formData}
                    domainId={organizationId}
                    domainType="DOMAIN_TYPE_ORGANIZATION"
                  />
                ))}
              </div>
            )}

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
            <DialogTitle>Reject: {getRequirementLabel()}</DialogTitle>
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
    const metadata = (execution.metadata || {}) as ApprovalMetadata
    const config = (execution.configuration || {}) as ApprovalConfig
    const requirements = config.approvals || []
    const approvals = metadata.approvals || []

    const approvedCount = approvals.length
    const requiredCount = requirements.length
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

  renderExpanded: ({ execution, workflowId, organizationId }: ExpandedViewProps) => {
    const metadata = (execution.metadata || {}) as ApprovalMetadata
    const config = (execution.configuration || {}) as ApprovalConfig
    const requirements = config.approvals || []
    const approvals = metadata.approvals || []

    // Create a map of approvals by requirement index
    const approvalsByRequirement = new Map<number, ApprovalRecord>()
    approvals.forEach((approval) => {
      approvalsByRequirement.set(approval.requirementIndex, approval)
    })

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

        {/* Approval Requirements */}
        {requirements.length > 0 && (
          <>
            <div className="space-y-2">
              {requirements.map((requirement, index) => (
                <ApprovalRequirementCard
                  key={index}
                  requirement={requirement}
                  requirementIndex={index}
                  approval={approvalsByRequirement.get(index)}
                  execution={execution}
                  workflowId={workflowId}
                  organizationId={organizationId || ''}
                />
              ))}
            </div>
          </>
        )}
      </div>
    )
  }
})
