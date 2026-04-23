import {
  canvasesCreateCanvas,
  canvasesCreateCanvasVersion,
  canvasesDescribeCanvas,
  canvasesEmitNodeEvent,
  canvasesListCanvasEvents,
  canvasesListEventExecutions,
  canvasesPublishCanvasVersion,
  canvasesUpdateCanvasVersion,
} from "@/api-client/sdk.gen";
import type { CanvasesCanvas, SuperplaneComponentsEdge, SuperplaneComponentsNode } from "@/api-client/types.gen";
import { AgentPanel } from "@/components/CanvasCreation/AgentPanel";
import { CLIPanel } from "@/components/CanvasCreation/CLIPanel";
import { Heading } from "@/components/Heading/heading";
import { Badge } from "@/components/ui/badge";
import { useAccount } from "@/contexts/AccountContext";
import { canvasKeys, useCanvasTemplates } from "@/hooks/useCanvasData";
import { useMe } from "@/hooks/useMe";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { QuickStartUiPanel } from "@/pages/home/QuickStartUiPanel";
import { useQueryClient } from "@tanstack/react-query";
import { Monitor, Sparkles, Terminal } from "lucide-react";
import { useState } from "react";
import { useNavigate } from "react-router-dom";

const QUICK_START_TEMPLATE_NAME = "Health Check Monitor";

/** Quick start runs one tick against status/200, then saves status/500 before opening the canvas. */
const QUICK_START_HTTP_URL_SERVER1 = "https://httpbin.org/status/200";
const QUICK_START_HTTP_URL_SERVER2 = "https://httpbin.org/status/500";

type Mode = "ui" | "cli" | "agent";

function normalizeTemplateName(name: string) {
  return name.trim().toLowerCase().replace(/ /g, "-");
}

function findQuickStartTemplate(templates: CanvasesCanvas[]) {
  const normalizedQuickStartName = normalizeTemplateName(QUICK_START_TEMPLATE_NAME);

  return templates.find((template) => {
    const templateName = template.metadata?.name;
    if (!templateName) {
      return false;
    }

    return normalizeTemplateName(templateName) === normalizedQuickStartName;
  });
}

async function publishCanvasGraphUpdate(params: {
  canvasId: string;
  name: string;
  description: string;
  nodes: SuperplaneComponentsNode[];
  edges: SuperplaneComponentsEdge[];
}) {
  const createVersionResponse = await canvasesCreateCanvasVersion(
    withOrganizationHeader({
      path: { canvasId: params.canvasId },
      body: {},
    }),
  );

  const versionId = createVersionResponse.data?.version?.metadata?.id;
  if (!versionId) {
    throw new Error("Failed to create draft version");
  }

  await canvasesUpdateCanvasVersion(
    withOrganizationHeader({
      path: { canvasId: params.canvasId, versionId },
      body: {
        canvas: {
          metadata: {
            name: params.name,
            description: params.description,
          },
          spec: {
            nodes: params.nodes,
            edges: params.edges,
          },
        },
      },
    }),
  );

  await canvasesPublishCanvasVersion(
    withOrganizationHeader({
      path: { canvasId: params.canvasId, versionId },
      body: {},
    }),
  );
}

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
  const { data: templates = [], isLoading: templatesLoading } = useCanvasTemplates(organizationId);
  const [mode, setMode] = useState<Mode>("ui");
  const [isLaunchingQuickStart, setIsLaunchingQuickStart] = useState(false);
  const [isCreatingBlankCanvas, setIsCreatingBlankCanvas] = useState(false);

  const firstName = account?.name?.split(" ")[0] || "";

  const handleQuickStart = async () => {
    if (templatesLoading) {
      showErrorToast("Templates are still loading. Please try again in a moment.");
      return;
    }

    const template = findQuickStartTemplate(templates);
    if (!template) {
      showErrorToast("Quick start template not found. Please try again later.");
      return;
    }

    setIsLaunchingQuickStart(true);
    try {
      const nodes = (template.spec?.nodes || []).map((node) => {
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
      const nodesWithServer2Url = nodes.map((node) =>
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

      await publishCanvasGraphUpdate({
        canvasId,
        name: QUICK_START_TEMPLATE_NAME,
        description,
        nodes: nodesWithServer2Url,
        edges,
      });

      const triggerNode = nodes.find((node) => node.trigger?.name === "schedule");
      const httpNode = nodes.find((node) => node.component?.name === "http");
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
          // Best-effort; the regular schedule will fire within ten minutes.
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

      await publishCanvasGraphUpdate({
        canvasId,
        name: QUICK_START_TEMPLATE_NAME,
        description,
        nodes,
        edges,
      });

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

        <div key={mode} className="min-h-[520px] animate-in fade-in-0 slide-in-from-bottom-1 duration-300">
          {mode === "ui" && (
            <QuickStartUiPanel
              canCreateCanvases={canCreateCanvases}
              permissionAllowed={permissionAllowed}
              templatesLoading={templatesLoading}
              isLaunchingQuickStart={isLaunchingQuickStart}
              isCreatingBlankCanvas={isCreatingBlankCanvas}
              onQuickStart={handleQuickStart}
              onBrowseTemplates={() => navigate(`/${organizationId}/templates`)}
              onCreateBlankCanvas={handleCreateBlankCanvas}
            />
          )}

          {mode === "cli" && <CLIPanel organizationId={organizationId} />}

          {mode === "agent" && <AgentPanel />}
        </div>
      </div>
    </div>
  );
}
