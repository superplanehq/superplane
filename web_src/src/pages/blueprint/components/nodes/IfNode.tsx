import { memo } from 'react'
import { NodeProps, Node } from '@xyflow/react'
import { If } from 'ui'


type IfNodeData = Node<{
  label?: string
  component?: string
  channels?: string[]
  configuration?: Record<string, any>
}>

export const IfNode = memo(({ data, selected }: NodeProps<IfNodeData>) => {
  return (
    <If
      data={data}
      selected={selected}
    />
  )
})

IfNode.displayName = 'IfNode'
