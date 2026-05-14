import { LayoutDashboard } from "lucide-react";

interface CanvasDashboardOverlayProps {
  canvasName: string;
  description?: string;
}

export function CanvasDashboardOverlay({ canvasName, description }: CanvasDashboardOverlayProps) {
  return (
    <div className="flex h-full w-full items-center justify-center p-6">
      <div className="w-full max-w-xl rounded-lg border border-slate-200 bg-white p-8 text-center shadow-sm">
        <div className="mx-auto mb-4 flex h-10 w-10 items-center justify-center rounded-md bg-slate-100 text-slate-600">
          <LayoutDashboard className="h-5 w-5" />
        </div>
        <h2 className="text-lg font-semibold text-slate-900">Dashboard</h2>
        <p className="mt-2 text-sm text-slate-600">
          Minimal dashboard mode for <span className="font-medium text-slate-800">{canvasName}</span>.
        </p>
        {description ? <p className="mt-1 text-sm text-slate-500">{description}</p> : null}
        <p className="mt-4 text-xs text-slate-500">More dashboard widgets and layout controls will be added next.</p>
      </div>
    </div>
  );
}
