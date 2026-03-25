import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import {
  ArrowRight,
  ChevronRight,
  Clock,
  Database,
  Globe,
  LayoutTemplate,
  Loader2,
  Mail,
  Monitor,
  Plug,
  Plus,
  Sparkles,
  Terminal,
  Timer,
} from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Heading } from "@/components/Heading/heading";
import { PermissionTooltip } from "@/components/PermissionGate";
import { canvasKeys, useCanvasTemplates } from "@/hooks/useCanvasData";
import {
  canvasesCreateCanvas,
  canvasesDescribeCanvas,
  canvasesEmitNodeEvent,
  canvasesListCanvasEvents,
  canvasesListEventExecutions,
  canvasesUpdateCanvasVersion2,
} from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { showErrorToast } from "@/utils/toast";
import { useAccount } from "@/contexts/AccountContext";
import { useMe } from "@/hooks/useMe";
import { CLIPanel } from "@/components/CanvasCreation/CLIPanel";
import { AgentPanel } from "@/components/CanvasCreation/AgentPanel";

const QUICK_START_TEMPLATE_NAME = "Health Check Monitor";

/** Quick start runs one tick against server1, then saves server2 before opening the canvas. */
const QUICK_START_HTTP_URL_SERVER1 = "https://app.superplane.com/server1";
const QUICK_START_HTTP_URL_SERVER2 = "https://app.superplane.com/server2";

type Mode = "ui" | "cli" | "agent";

const PERSONAS = [
  {
    mode: "ui" as Mode,
    icon: Monitor,
    title: "Point & Click",
    subtitle: "Visual drag-and-drop builder",
  },
  {
    mode: "cli" as Mode,
    icon: Terminal,
    title: "Terminal Warrior",
    subtitle: "CLI power user",
  },
  {
    mode: "agent" as Mode,
    icon: Sparkles,
    title: "Let AI Cook",
    subtitle: "Prompt your problems away",
    badge: "Coming soon",
  },
];

const FLOW_STEPS = [
  {
    icon: Clock,
    label: "Schedule",
    bg: "bg-indigo-100 dark:bg-indigo-900/50",
    iconColor: "text-indigo-600 dark:text-indigo-400",
  },
  {
    icon: Globe,
    label: "HTTP Check",
    bg: "bg-sky-100 dark:bg-sky-900/50",
    iconColor: "text-sky-600 dark:text-sky-400",
  },
  {
    icon: Database,
    label: "Memory Check",
    bg: "bg-violet-100 dark:bg-violet-900/50",
    iconColor: "text-violet-600 dark:text-violet-400",
  },
  {
    icon: Mail,
    label: "Email Alert",
    bg: "bg-amber-100 dark:bg-amber-900/50",
    iconColor: "text-amber-600 dark:text-amber-400",
  },
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
  const { data: me } = useMe();
  const permissionAllowed = canCreateCanvases || permissionsLoading;
  const { data: templates = [] } = useCanvasTemplates(organizationId);
  const [mode, setMode] = useState<Mode>("ui");
  const [isLaunchingQuickStart, setIsLaunchingQuickStart] = useState(false);
  const [isCreatingBlankCanvas, setIsCreatingBlankCanvas] = useState(false);

  const firstName = account?.name?.split(" ")[0] || "";

  const handleQuickStart = async () => {
    const template = templates.find((t: any) => t.metadata?.name === QUICK_START_TEMPLATE_NAME);
    if (!template) {
      showErrorToast("Quick start template not found. Please try again later.");
      return;
    }

    setIsLaunchingQuickStart(true);
    try {
      const nodes = (template.spec?.nodes || []).map((node: any) => {
        if (node.component?.name === "sendEmail" && me?.id) {
          return {
            ...node,
            configuration: {
              ...node.configuration,
              recipients: [{ type: "user", user: me.id }],
            },
          };
        }
        if (node.component?.name === "http") {
          return {
            ...node,
            configuration: {
              ...node.configuration,
              url: QUICK_START_HTTP_URL_SERVER1,
            },
          };
        }
        return node;
      });
      const nodesWithServer2Url = nodes.map((node: any) =>
        node.component?.name === "http"
          ? {
              ...node,
              configuration: {
                ...node.configuration,
                url: QUICK_START_HTTP_URL_SERVER2,
              },
            }
          : node,
      );
      const edges = template.spec?.edges || [];
      const description = template.metadata?.description || "";

      const result = await canvasesCreateCanvas(
        withOrganizationHeader({
          body: {
            canvas: {
              metadata: { name: QUICK_START_TEMPLATE_NAME, description },
              spec: { nodes, edges },
            },
          },
        }),
      );

      const canvasId = result.data?.canvas?.metadata?.id;
      if (!canvasId) return;

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

      const triggerNode = nodes.find((n: any) => n.trigger?.name === "schedule");
      const httpNode = nodes.find((n: any) => n.component?.name === "http");
      let emittedEventId: string | undefined;

      if (triggerNode) {
        const now = new Date();
        try {
          const emitResponse = await canvasesEmitNodeEvent(
            withOrganizationHeader({
              path: { canvasId, nodeId: triggerNode.id },
              body: {
                channel: "default",
                data: {
                  type: "scheduler.tick",
                  timestamp: now.toISOString(),
                  data: {
                    calendar: {
                      year: String(now.getFullYear()),
                      month: now.toLocaleString("en-US", { month: "long" }),
                      day: String(now.getDate()),
                      hour: String(now.getHours()).padStart(2, "0"),
                      minute: String(now.getMinutes()).padStart(2, "0"),
                      second: String(now.getSeconds()).padStart(2, "0"),
                      week_day: now.toLocaleString("en-US", { weekday: "long" }),
                    },
                  },
                },
              },
            }),
          );
          emittedEventId = emitResponse.data?.eventId;
        } catch {
          // Best-effort; the regular schedule will fire within a minute.
        }
      }

      if (emittedEventId && httpNode?.id) {
        for (let i = 0; i < 15; i++) {
          await new Promise((resolve) => setTimeout(resolve, 1000));
          try {
            const execResp = await canvasesListEventExecutions(
              withOrganizationHeader({
                path: { canvasId, eventId: emittedEventId },
              }),
            );
            const httpDone = (execResp.data?.executions ?? []).some(
              (e) => e.nodeId === httpNode.id && e.state === "STATE_FINISHED",
            );
            if (httpDone) break;
          } catch {
            break;
          }
        }
      } else {
        await new Promise((resolve) => setTimeout(resolve, 2000));
      }

      await canvasesUpdateCanvasVersion2(
        withOrganizationHeader({
          path: { canvasId },
          body: {
            canvas: {
              metadata: { name: QUICK_START_TEMPLATE_NAME, description },
              spec: { nodes: nodesWithServer2Url, edges },
            },
          },
        }),
      );

      const [canvasResponse, eventsResponse] = await Promise.all([
        canvasesDescribeCanvas(withOrganizationHeader({ path: { id: canvasId } })),
        canvasesListCanvasEvents(withOrganizationHeader({ path: { canvasId }, query: { limit: 50 } })),
      ]);

      if (canvasResponse.data?.canvas) {
        queryClient.setQueryData(canvasKeys.detail(organizationId, canvasId), canvasResponse.data.canvas);
      }
      if (eventsResponse.data) {
        queryClient.setQueryData(canvasKeys.eventList(canvasId, 50), eventsResponse.data);
      }

      navigate(`/${organizationId}/canvases/${canvasId}`, { replace: true });

      queryClient.invalidateQueries({ queryKey: canvasKeys.lists() });
    } catch (error) {
      const message = (error as Error)?.message || "Failed to create canvas";
      showErrorToast(message);
    } finally {
      setIsLaunchingQuickStart(false);
    }
  };

  const handleCreateBlankCanvas = async () => {
    setIsCreatingBlankCanvas(true);
    try {
      const result = await canvasesCreateCanvas(
        withOrganizationHeader({
          body: {
            canvas: {
              metadata: { name: "Untitled Canvas", description: "" },
              spec: { nodes: [], edges: [] },
            },
          },
        }),
      );

      const canvasId = result.data?.canvas?.metadata?.id;
      if (!canvasId) return;

      queryClient.invalidateQueries({ queryKey: canvasKeys.lists() });
      navigate(`/${organizationId}/canvases/${canvasId}`, { replace: true });
    } catch (error) {
      const message = (error as Error)?.message || "Failed to create canvas";
      showErrorToast(message);
    } finally {
      setIsCreatingBlankCanvas(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-[calc(100vh-8rem)]">
      <div className="max-w-2xl w-full mx-auto px-4">
        {/* Greeting */}
        <div className="text-center mb-6 animate-in fade-in-0 duration-700">
          <Heading level={1} className="!text-3xl mb-2 tracking-tight">
            {firstName ? `Hey ${firstName}, welcome!` : "Welcome to SuperPlane!"}
          </Heading>
          <p className="text-base text-gray-500 dark:text-gray-400">How do you roll?</p>
        </div>

        {/* Persona tiles */}
        <div
          className="grid grid-cols-3 gap-3 mb-6 animate-in fade-in-0 slide-in-from-bottom-2 duration-500"
          style={{ animationDelay: "150ms", animationFillMode: "backwards" }}
        >
          {PERSONAS.map((persona) => {
            const isSelected = mode === persona.mode;
            return (
              <button
                key={persona.mode}
                type="button"
                onClick={() => setMode(persona.mode)}
                className={`relative text-center rounded-xl p-4 transition-all duration-200 cursor-pointer ${
                  isSelected
                    ? "bg-white dark:bg-gray-800 outline outline-2 outline-primary shadow-md"
                    : "bg-white/60 dark:bg-gray-800/60 outline outline-slate-950/10 dark:outline-gray-700 hover:bg-white hover:dark:bg-gray-800 hover:shadow-sm"
                }`}
              >
                <div
                  className={`mx-auto mb-2 rounded-lg p-2 w-fit transition-colors duration-200 ${
                    isSelected ? "bg-primary/10 dark:bg-primary/20" : "bg-gray-100 dark:bg-gray-700"
                  }`}
                >
                  <persona.icon
                    size={20}
                    className={`transition-colors duration-200 ${
                      isSelected ? "text-primary" : "text-gray-500 dark:text-gray-400"
                    }`}
                  />
                </div>
                <div
                  className={`text-sm font-semibold mb-0.5 transition-colors duration-200 ${
                    isSelected ? "text-gray-900 dark:text-white" : "text-gray-600 dark:text-gray-300"
                  }`}
                >
                  {persona.title}
                </div>
                <div className="text-[11px] text-gray-400 dark:text-gray-500">{persona.subtitle}</div>
                {persona.badge && (
                  <Badge variant="secondary" className="absolute top-2 right-2 text-[9px] px-1.5 py-0">
                    {persona.badge}
                  </Badge>
                )}
              </button>
            );
          })}
        </div>

        {/* Content area with crossfade */}
        <div key={mode} className="min-h-[520px] animate-in fade-in-0 slide-in-from-bottom-1 duration-300">
          {mode === "ui" && (
            <>
              {/* Quick Start hero card */}
              <PermissionTooltip allowed={permissionAllowed} message="You don't have permission to create canvases.">
                <button
                  type="button"
                  disabled={!canCreateCanvases || isLaunchingQuickStart}
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
                        Pings your endpoint every minute and alerts only on healthy-to-failing transitions, including
                        approximately how long it stayed healthy.
                      </p>
                    </div>
                    {isLaunchingQuickStart ? (
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
                        <Timer size={12} />1 min setup
                      </span>
                      <span className="text-gray-300 dark:text-gray-600">|</span>
                      <span className="inline-flex items-center gap-1 text-[11px] text-gray-500 dark:text-gray-400">
                        <Plug size={12} />
                        No integrations
                      </span>
                    </div>
                    {isLaunchingQuickStart ? (
                      <span className="text-[11px] text-gray-400">Setting up...</span>
                    ) : (
                      <span className="text-[11px] text-primary group-hover:underline">Get started</span>
                    )}
                  </div>
                </button>
              </PermissionTooltip>

              {/* Divider */}
              <div className="flex items-center gap-3 mb-5">
                <div className="flex-1 h-px bg-gray-200 dark:bg-gray-700" />
                <span className="text-xs text-gray-400 dark:text-gray-500 font-medium uppercase tracking-wider">
                  or pick your own path
                </span>
                <div className="flex-1 h-px bg-gray-200 dark:bg-gray-700" />
              </div>

              {/* Secondary cards */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
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
                    disabled={!canCreateCanvases || isCreatingBlankCanvas}
                    onClick={handleCreateBlankCanvas}
                    className="w-full text-left bg-white dark:bg-gray-800 rounded-xl outline outline-slate-950/10 dark:outline-gray-700 p-5 hover:shadow-md transition-shadow cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <div className="flex items-start gap-3">
                      <div className="mt-0.5 rounded-lg bg-gray-100 dark:bg-gray-700 p-2">
                        {isCreatingBlankCanvas ? (
                          <Loader2 size={18} className="text-gray-600 dark:text-gray-300 animate-spin" />
                        ) : (
                          <Plus size={18} className="text-gray-600 dark:text-gray-300" />
                        )}
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
            </>
          )}

          {mode === "cli" && <CLIPanel organizationId={organizationId} />}

          {mode === "agent" && <AgentPanel />}
        </div>
      </div>
    </div>
  );
}
