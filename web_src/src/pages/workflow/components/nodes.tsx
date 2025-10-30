import { NodeProps } from '@xyflow/react'
import { WorkflowNodeWrapper } from './WorkflowNodeWrapper'
import { IfNode } from '../../../pages/custom-component/components/nodes/IfNode'
import { HttpNode } from '../../../pages/custom-component/components/nodes/HttpNode'
import { FilterNode } from '../../../pages/custom-component/components/nodes/FilterNode'
import { ApprovalNode } from '../../../pages/custom-component/components/nodes/ApprovalNode'
import { DefaultNode } from '../../../pages/custom-component/components/nodes/DefaultNode'
import { StartTriggerNode } from '../../custom-component/components/nodes/StartTriggerNode'
import { ScheduledTriggerNode } from '../../../pages/custom-component/components/nodes/ScheduledTriggerNode'
import { WebhookTriggerNode } from '../../../pages/custom-component/components/nodes/WebhookTriggerNode'
import { GithubTriggerNode } from '../../../pages/custom-component/components/nodes/GithubTriggerNode'
import { SemaphoreTriggerNode } from '../../../pages/custom-component/components/nodes/SemaphoreTriggerNode'

export const WorkflowIfNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={IfNode}
    onEdit={props.data.onEdit as (() => void) | undefined}
    onEmit={props.data.onEmit as (() => void) | undefined}
  />
)

export const WorkflowHttpNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={HttpNode}
    onEdit={props.data.onEdit as (() => void) | undefined}
    onEmit={props.data.onEmit as (() => void) | undefined}
  />
)

export const WorkflowFilterNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={FilterNode}
    onEdit={props.data.onEdit as (() => void) | undefined}
    onEmit={props.data.onEmit as (() => void) | undefined}
  />
)

export const WorkflowApprovalNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={ApprovalNode}
    onEdit={props.data.onEdit as (() => void) | undefined}
    onEmit={props.data.onEmit as (() => void) | undefined}
  />
)

export const WorkflowDefaultNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={DefaultNode}
    onEdit={props.data.onEdit as (() => void) | undefined}
    onEmit={props.data.onEmit as (() => void) | undefined}
  />
)

export const WorkflowStartTriggerNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={StartTriggerNode}
    onEdit={props.data.onEdit as (() => void) | undefined}
    onEmit={props.data.onEmit as (() => void) | undefined}
  />
)

export const WorkflowScheduledTriggerNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={ScheduledTriggerNode}
    onEdit={props.data.onEdit as (() => void) | undefined}
    onEmit={props.data.onEmit as (() => void) | undefined}
  />
)

export const WorkflowWebhookTriggerNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={WebhookTriggerNode}
    onEdit={props.data.onEdit as (() => void) | undefined}
    onEmit={props.data.onEmit as (() => void) | undefined}
  />
)

export const WorkflowGithubTriggerNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={GithubTriggerNode}
    onEdit={props.data.onEdit as (() => void) | undefined}
    onEmit={props.data.onEmit as (() => void) | undefined}
  />
)

export const WorkflowSemaphoreTriggerNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={SemaphoreTriggerNode}
    onEdit={props.data.onEdit as (() => void) | undefined}
    onEmit={props.data.onEmit as (() => void) | undefined}
  />
)
