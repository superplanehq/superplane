import { useParams, useNavigate } from 'react-router-dom'
import { Header, type BreadcrumbItem } from '@/ui/CanvasPage/Header'

export function NodeRunPage() {
  const { organizationId, workflowId } = useParams()
  const navigate = useNavigate()

  const breadcrumbs: BreadcrumbItem[] = [
    { label: 'Canvases', href: `/${organizationId}` },
    { label: workflowId || 'Workflow' },
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

export default NodeRunPage
