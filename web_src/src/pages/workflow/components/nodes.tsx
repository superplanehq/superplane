import { NodeProps } from '@xyflow/react'
import { WorkflowNodeWrapper } from './WorkflowNodeWrapper'
import { IfNode } from '../../../pages/blueprint/components/nodes/IfNode'
import { HttpNode } from '../../../pages/blueprint/components/nodes/HttpNode'
import { FilterNode } from '../../../pages/blueprint/components/nodes/FilterNode'
import { SwitchNode } from '../../../pages/blueprint/components/nodes/SwitchNode'
import { ApprovalNode } from '../../../pages/blueprint/components/nodes/ApprovalNode'
import { DefaultNode } from '../../../pages/blueprint/components/nodes/DefaultNode'
import { ManualTriggerNode } from '../../../pages/blueprint/components/nodes/ManualTriggerNode'
import { ScheduledTriggerNode } from '../../../pages/blueprint/components/nodes/ScheduledTriggerNode'
import { WebhookTriggerNode } from '../../../pages/blueprint/components/nodes/WebhookTriggerNode'
import { GithubTriggerNode } from '../../../pages/blueprint/components/nodes/GithubTriggerNode'
import { SemaphoreTriggerNode } from '../../../pages/blueprint/components/nodes/SemaphoreTriggerNode'

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

export const WorkflowSwitchNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={SwitchNode}
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

export const WorkflowManualTriggerNode = (props: NodeProps) => (
  <WorkflowNodeWrapper
    {...props}
    BaseNodeComponent={ManualTriggerNode}
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
