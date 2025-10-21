import { CanvasMembersSection } from '../../CanvasMembersSection/canvas-members-section'

interface CanvasMembersProps {
  canvasId: string
  organizationId: string
}

export function CanvasMembers({ canvasId, organizationId }: CanvasMembersProps) {
  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-4xl mx-auto py-6">
        <CanvasMembersSection
          canvasId={canvasId}
          organizationId={organizationId}
          title="Canvas Members"
          description="Manage members and invitations for this canvas"
        />
      </div>
    </div>
  )
}