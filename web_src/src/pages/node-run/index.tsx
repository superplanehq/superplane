import { useParams, useNavigate } from 'react-router-dom'
import { Header, type BreadcrumbItem } from '@/ui/CanvasPage/Header'
import { useWorkflow } from '@/hooks/useWorkflowData'
import { useBlueprints, useComponents } from '@/hooks/useBlueprintData'
import { useNodeExecutions } from '@/hooks/useWorkflowData'
import { getTriggerRenderer } from '@/pages/workflowv2/renderers'
import { getBackgroundColorClass, getColorClass } from '@/utils/colors'

export function NodeRunPage() {
  const { organizationId, workflowId, nodeId } = useParams()
  const { data: workflow } = useWorkflow(organizationId || '', workflowId || '')
  const { data: blueprints = [] } = useBlueprints(organizationId || '')
  const { data: components = [] } = useComponents(organizationId || '')

  // Derive icon from metadata similar to canvas
  const node = workflow?.nodes?.find(n => n.id === nodeId)
  let iconSlug: string | undefined
  let color: string | undefined
  if (node?.blueprint?.id) {
    const bp = blueprints.find(b => b.id === node.blueprint?.id)
    iconSlug = bp?.icon || undefined
    color = bp?.color || undefined
  } else if (node?.component?.name) {
    const comp = components.find(c => c.name === node.component?.name)
    iconSlug = comp?.icon || undefined
    color = comp?.color || undefined
  } else if (node?.trigger?.name) {
    // triggers not fetched here; fall back to default
    iconSlug = 'bolt'
    color = 'blue'
  }

export function NodeRunPage() {
  const { data: blueprints = [] } = useBlueprints(organizationId || '')
  const { data: components = [] } = useComponents(organizationId || '')
  const nodeName = workflow?.nodes?.find(n => n.id === nodeId)?.name || 'Component'
  const { data: nodeExecs } = useNodeExecutions(workflowId || '', nodeId || '')
  const latestExecution = nodeExecs?.executions?.[0]
  const latestRunTitle = (() => {
    if (!latestExecution) return undefined
    const rootNode = workflow?.nodes?.find(n => n.id === latestExecution.rootEvent?.nodeId)
    const renderer = getTriggerRenderer(rootNode?.trigger?.name || '')
    if (latestExecution.rootEvent) {
      return renderer.getTitleAndSubtitle(latestExecution.rootEvent).title
    }
    return 'Execution'
  })()

  // Derive icon from metadata similar to canvas
  const node = workflow?.nodes?.find(n => n.id === nodeId)
  let iconSlug: string | undefined
  let color: string | undefined
  if (node?.blueprint?.id) {
    const bp = blueprints.find(b => b.id === node.blueprint?.id)
    iconSlug = bp?.icon || undefined
    color = bp?.color || undefined
  } else if (node?.component?.name) {
    const comp = components.find(c => c.name === node.component?.name)
    iconSlug = comp?.icon || undefined
    color = comp?.color || undefined
  } else if (node?.trigger?.name) {
    // triggers not fetched here; fall back to default
    iconSlug = 'bolt'
    color = 'blue'
  }

  const navigate = useNavigate()

  const breadcrumbs: BreadcrumbItem[] = [
    { label: 'Canvases', href: `/${organizationId}` },
    { label: workflow?.name || 'Workflow', href: `/${organizationId}/workflows/${workflowId}` },
    { label: nodeName, iconSlug: iconSlug || 'boxes', iconColor: getColorClass(color || 'indigo'), iconBackground: getBackgroundColorClass(color || 'indigo') },
    ...(latestRunTitle ? [{ label: latestRunTitle }] as BreadcrumbItem[] : []),
    { label: 'Build/Test/Deploy Stage', iconSlug: 'git-branch', iconColor: 'text-purple-500' },
  ]

  return (
    <div className="h-screen w-screen bg-slate-50">
      <Header
        breadcrumbs={breadcrumbs}
        organizationId={organizationId}
        onLogoClick={() => navigate(`/${organizationId}`)}
      />
    </div>
  )
}

export default NodeRunPage;