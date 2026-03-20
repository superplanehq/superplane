import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { ArrowRight, ChevronRight, Clock, Globe, LayoutTemplate, Loader2, Mail, Plug, Plus, Timer } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Heading } from "@/components/Heading/heading";
import { PermissionTooltip } from "@/components/PermissionGate";
import { useCreateCanvas, useCanvasTemplates } from "@/hooks/useCanvasData";
import { canvasesUpdateCanvasVersion2 } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { showErrorToast } from "@/utils/toast";
import { useAccount } from "@/contexts/AccountContext";

const QUICK_START_TEMPLATE_NAME = "Health Check Monitor";

const FLOW_STEPS = [
  { icon: Clock, label: "Schedule", bg: "bg-indigo-100 dark:bg-indigo-900/50", iconColor: "text-indigo-600 dark:text-indigo-400" },
  { icon: Globe, label: "HTTP Check", bg: "bg-sky-100 dark:bg-sky-900/50", iconColor: "text-sky-600 dark:text-sky-400" },
  { icon: Mail, label: "Email Alert", bg: "bg-amber-100 dark:bg-amber-900/50", iconColor: "text-amber-600 dark:text-amber-400" },
];

interface OnboardingWelcomeProps {
  organizationId: string;
  canCreateCanvases: boolean;
  permissionsLoading: boolean;
}

export function OnboardingWelcome({ organizationId, canCreateCanvases, permissionsLoading }: OnboardingWelcomeProps) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { account } = useAccount();
  const permissionAllowed = canCreateCanvases || permissionsLoading;
  const createCanvasMutation = useCreateCanvas(organizationId);
  const { data: templates = [] } = useCanvasTemplates(organizationId);

  const firstName = account?.name?.split(" ")[0] || "";

  const handleQuickStart = async () => {
    const template = templates.find((t: any) => t.metadata?.name === QUICK_START_TEMPLATE_NAME);
    if (!template) {
      showErrorToast("Quick start template not found. Please try again later.");
      return;
    }

    try {
      const nodes = template.spec?.nodes || [];
      const edges = template.spec?.edges || [];
      const description = template.metadata?.description || "";

      const result = await createCanvasMutation.mutateAsync({
        name: QUICK_START_TEMPLATE_NAME,
        description,
        nodes,
        edges,
      });

      const canvasId = result?.data?.canvas?.metadata?.id;
      if (!canvasId) return;

      // Trigger a save so the backend runs Setup() validation on each node,
      // surfacing warnings for incomplete configuration (e.g. empty HTTP URL).
      await canvasesUpdateCanvasVersion2(
        withOrganizationHeader({
          path: { canvasId },
          body: {
            canvas: {
              metadata: { name: QUICK_START_TEMPLATE_NAME, description },
              spec: { nodes, edges },
            },
          },
        }),
      );

      await queryClient.invalidateQueries({ queryKey: ["canvases"] });
      navigate(`/${organizationId}/canvases/${canvasId}`);
    } catch (error) {
      const message = (error as Error)?.message || "Failed to create canvas";
      showErrorToast(message);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-[calc(100vh-8rem)]">
      <div className="max-w-2xl w-full mx-auto px-4">
        {/* Greeting */}
        <div className="text-center mb-8 animate-in fade-in-0 duration-700">
          <Heading level={1} className="!text-3xl mb-2 tracking-tight">
            {firstName ? `Hey ${firstName}, welcome!` : "Welcome to SuperPlane!"}
          </Heading>
          <p className="text-base text-gray-500 dark:text-gray-400 max-w-md mx-auto">
            Your first workflow is two clicks away. Seriously.
          </p>
        </div>

        {/* Quick Start hero card */}
        <div
          className="animate-in fade-in-0 slide-in-from-bottom-3 duration-600"
          style={{ animationDelay: "200ms", animationFillMode: "backwards" }}
        >
          <PermissionTooltip allowed={permissionAllowed} message="You don't have permission to create canvases.">
            <button
              type="button"
              disabled={!canCreateCanvases || createCanvasMutation.isPending}
              onClick={handleQuickStart}
              className="group w-full text-left bg-white dark:bg-gray-800 rounded-xl outline outline-slate-950/10 dark:outline-gray-700 p-5 mb-5 hover:shadow-lg hover:outline-primary/30 transition-all duration-200 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <div className="flex items-start justify-between gap-3 mb-3">
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <Heading level={3} className="!text-[15px]">
                      Health Check Monitor
                    </Heading>
                    <Badge variant="secondary" className="text-[10px] font-medium">
                      Quick Start
                    </Badge>
                  </div>
                  <p className="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed">
                    Pings your endpoint every minute. If it goes down, you get an email.
                  </p>
                </div>
                {createCanvasMutation.isPending ? (
                  <Loader2 size={18} className="mt-0.5 text-gray-400 shrink-0 animate-spin" />
                ) : (
                  <ArrowRight
                    size={18}
                    className="mt-0.5 text-gray-300 dark:text-gray-600 shrink-0 group-hover:text-primary group-hover:translate-x-0.5 transition-all"
                  />
                )}
              </div>

              <div className="flex items-center gap-1.5 mb-3">
                {FLOW_STEPS.map((step, i) => (
                  <div key={step.label} className="flex items-center gap-1.5">
                    <span className={`inline-flex items-center gap-1.5 rounded-full ${step.bg} px-2.5 py-1`}>
                      <step.icon size={12} className={step.iconColor} />
                      <span className="text-[11px] font-medium text-gray-600 dark:text-gray-300">{step.label}</span>
                    </span>
                    {i < FLOW_STEPS.length - 1 && (
                      <ChevronRight size={12} className="text-gray-400 dark:text-gray-500 shrink-0" />
                    )}
                  </div>
                ))}
              </div>

              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="inline-flex items-center gap-1 text-[11px] text-gray-500 dark:text-gray-400">
                    <Timer size={12} />
                    1 min setup
                  </span>
                  <span className="text-gray-300 dark:text-gray-600">|</span>
                  <span className="inline-flex items-center gap-1 text-[11px] text-gray-500 dark:text-gray-400">
                    <Plug size={12} />
                    No integrations
                  </span>
                </div>
                {createCanvasMutation.isPending ? (
                  <span className="text-[11px] text-gray-400">Creating...</span>
                ) : (
                  <span className="text-[11px] text-primary group-hover:underline">Get started</span>
                )}
              </div>
            </button>
          </PermissionTooltip>
        </div>

        {/* Divider with "or" */}
        <div
          className="flex items-center gap-3 mb-5 animate-in fade-in-0 duration-500"
          style={{ animationDelay: "500ms", animationFillMode: "backwards" }}
        >
          <div className="flex-1 h-px bg-gray-200 dark:bg-gray-700" />
          <span className="text-xs text-gray-400 dark:text-gray-500 font-medium uppercase tracking-wider">
            or pick your own path
          </span>
          <div className="flex-1 h-px bg-gray-200 dark:bg-gray-700" />
        </div>

        {/* Secondary cards */}
        <div
          className="grid grid-cols-1 sm:grid-cols-2 gap-4 animate-in fade-in-0 slide-in-from-bottom-2 duration-500"
          style={{ animationDelay: "600ms", animationFillMode: "backwards" }}
        >
          <PermissionTooltip allowed={permissionAllowed} message="You don't have permission to create canvases.">
            <button
              type="button"
              disabled={!canCreateCanvases}
              onClick={() => navigate(`/${organizationId}/templates`)}
              className="w-full text-left bg-white dark:bg-gray-800 rounded-xl outline outline-slate-950/10 dark:outline-gray-700 p-5 hover:shadow-md transition-shadow cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <div className="flex items-start gap-3">
                <div className="mt-0.5 rounded-lg bg-gray-100 dark:bg-gray-700 p-2">
                  <LayoutTemplate size={18} className="text-gray-600 dark:text-gray-300" />
                </div>
                <div>
                  <Heading level={3} className="!text-sm mb-1">
                    Browse Templates
                  </Heading>
                  <p className="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed">
                    Incident routing, CI/CD, rollbacks, and more. Ready to go.
                  </p>
                </div>
              </div>
            </button>
          </PermissionTooltip>

          <PermissionTooltip allowed={permissionAllowed} message="You don't have permission to create canvases.">
            <button
              type="button"
              disabled={!canCreateCanvases}
              onClick={() => navigate(`/${organizationId}/canvases/new`)}
              className="w-full text-left bg-white dark:bg-gray-800 rounded-xl outline outline-slate-950/10 dark:outline-gray-700 p-5 hover:shadow-md transition-shadow cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <div className="flex items-start gap-3">
                <div className="mt-0.5 rounded-lg bg-gray-100 dark:bg-gray-700 p-2">
                  <Plus size={18} className="text-gray-600 dark:text-gray-300" />
                </div>
                <div>
                  <Heading level={3} className="!text-sm mb-1">
                    Blank Canvas
                  </Heading>
                  <p className="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed">
                    Start from scratch. You know what you&rsquo;re doing.
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
