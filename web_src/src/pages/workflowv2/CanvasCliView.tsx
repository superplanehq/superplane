import { CliCommandsPanel } from "@/ui/CanvasPage/CliCommandsPanel";

interface CanvasCliViewProps {
  canvasId?: string;
  organizationId?: string;
}

export function CanvasCliView({ canvasId, organizationId }: CanvasCliViewProps) {
  return (
    <div className="px-4 py-6">
      <div className="mx-auto w-full max-w-3xl">
        <div className="overflow-hidden rounded-lg border border-slate-950/15 bg-white">
          <div className="px-4 py-3">
            <h2 className="text-sm font-semibold text-gray-900">CLI</h2>
            <p className="text-[13px] text-gray-500">
              SuperPlane CLI commands for this canvas and how to install the CLI.
            </p>
          </div>
          <CliCommandsPanel canvasId={canvasId} organizationId={organizationId} />
        </div>
      </div>
    </div>
  );
}
