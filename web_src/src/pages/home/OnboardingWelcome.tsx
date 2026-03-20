import { useNavigate } from "react-router-dom";
import { Activity, ArrowRight, LayoutTemplate, Loader2, Plus } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Heading } from "@/components/Heading/heading";
import { PermissionTooltip } from "@/components/PermissionGate";
import { useCreateCanvas } from "@/hooks/useCanvasData";
import { showErrorToast } from "@/utils/toast";

interface OnboardingWelcomeProps {
  organizationId: string;
  canCreateCanvases: boolean;
  permissionsLoading: boolean;
}

export function OnboardingWelcome({ organizationId, canCreateCanvases, permissionsLoading }: OnboardingWelcomeProps) {
  const navigate = useNavigate();
  const permissionAllowed = canCreateCanvases || permissionsLoading;
  const createCanvasMutation = useCreateCanvas(organizationId);

  const handleQuickStart = async () => {
    try {
      const result = await createCanvasMutation.mutateAsync({
        name: "Health Check Monitor",
        description: "Monitor an endpoint and get notified when it goes down.",
      });
      const canvasId = result?.data?.canvas?.metadata?.id;
      if (canvasId) {
        navigate(`/${organizationId}/canvases/${canvasId}`);
      }
    } catch (error) {
      const message = (error as Error)?.message || "Failed to create canvas";
      showErrorToast(message);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-[calc(100vh-8rem)]">
      <div className="max-w-2xl w-full mx-auto">
        {/* Hero */}
        <div className="text-center mb-10">
          <Heading level={1} className="!text-2xl mb-3">
            Welcome to SuperPlane
          </Heading>
          <p className="text-sm text-gray-500 dark:text-gray-400 max-w-lg mx-auto leading-relaxed">
            Connect your DevOps tools into automated workflows. Trigger actions across GitHub, Slack, PagerDuty, AWS,
            and 35+ more integrations &mdash; or start with zero setup.
          </p>
        </div>

        {/* Quick Start — primary card */}
        <PermissionTooltip allowed={permissionAllowed} message="You don't have permission to create canvases.">
          <button
            type="button"
            disabled={!canCreateCanvases || createCanvasMutation.isPending}
            onClick={handleQuickStart}
            className="w-full text-left bg-white dark:bg-gray-800 rounded-md outline outline-slate-950/10 dark:outline-gray-700 p-6 mb-4 hover:shadow-md transition-shadow cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-start gap-4">
                <div className="mt-0.5 rounded-md bg-primary/10 p-2">
                  <Activity size={20} className="text-primary" />
                </div>
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <Heading level={3} className="!text-base">
                      Quick Start: Health Check Monitor
                    </Heading>
                    <Badge variant="secondary" className="text-[11px]">
                      Recommended
                    </Badge>
                  </div>
                  <p className="text-sm text-gray-500 dark:text-gray-400 leading-relaxed">
                    Monitor any endpoint and get notified when it goes down. Takes 2 minutes, no integrations required.
                  </p>
                </div>
              </div>
              {createCanvasMutation.isPending ? (
                <Loader2 size={18} className="mt-1.5 text-gray-400 shrink-0 animate-spin" />
              ) : (
                <ArrowRight size={18} className="mt-1.5 text-gray-400 shrink-0" />
              )}
            </div>
          </button>
        </PermissionTooltip>

        {/* Secondary cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          {/* Templates */}
          <PermissionTooltip allowed={permissionAllowed} message="You don't have permission to create canvases.">
            <button
              type="button"
              disabled={!canCreateCanvases}
              onClick={() => navigate(`/${organizationId}/templates`)}
              className="w-full text-left bg-white dark:bg-gray-800 rounded-md outline outline-slate-950/10 dark:outline-gray-700 p-5 hover:shadow-md transition-shadow cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <div className="flex items-start gap-3">
                <div className="mt-0.5 rounded-md bg-gray-100 dark:bg-gray-700 p-2">
                  <LayoutTemplate size={18} className="text-gray-600 dark:text-gray-300" />
                </div>
                <div>
                  <Heading level={3} className="!text-sm mb-1">
                    Start from a Template
                  </Heading>
                  <p className="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed">
                    Pre-built workflows for incident routing, CI/CD, rollbacks, and more.
                  </p>
                </div>
              </div>
            </button>
          </PermissionTooltip>

          {/* Blank Canvas */}
          <PermissionTooltip allowed={permissionAllowed} message="You don't have permission to create canvases.">
            <button
              type="button"
              disabled={!canCreateCanvases}
              onClick={() => navigate(`/${organizationId}/canvases/new`)}
              className="w-full text-left bg-white dark:bg-gray-800 rounded-md outline outline-slate-950/10 dark:outline-gray-700 p-5 hover:shadow-md transition-shadow cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <div className="flex items-start gap-3">
                <div className="mt-0.5 rounded-md bg-gray-100 dark:bg-gray-700 p-2">
                  <Plus size={18} className="text-gray-600 dark:text-gray-300" />
                </div>
                <div>
                  <Heading level={3} className="!text-sm mb-1">
                    Blank Canvas
                  </Heading>
                  <p className="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed">
                    Build a workflow from scratch with full control.
                  </p>
                </div>
              </div>
            </button>
          </PermissionTooltip>
        </div>
      </div>
    </div>
  );
}
