import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { Field, Label } from "../../components/Fieldset/fieldset";
import { Heading } from "../../components/Heading/heading";
import { Input } from "../../components/Input/input";
import { Text } from "../../components/Text/text";
import { Textarea } from "../../components/ui/textarea";
import { Button } from "../../components/ui/button";
import { LoadingButton } from "../../components/ui/loading-button";
import { useCreateCanvas, useCanvasTemplates } from "../../hooks/useCanvasData";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { UsageLimitAlert } from "@/components/UsageLimitAlert";
import { getApiErrorMessage } from "@/utils/errors";
import { getUsageLimitNotice, getUsageLimitToastMessage } from "@/utils/usageLimits";
import { showErrorToast } from "../../utils/toast";
import type { CanvasesCanvas, ComponentsEdge, ComponentsNode } from "@/api-client";
import { Rainbow } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getIntegrationIconSrc } from "@/ui/componentSidebar/integrationIcons";
import { extractIntegrations, getTemplateTags, countNodesByType } from "./templateMetadata";

const MAX_CANVAS_NAME_LENGTH = 50;
const MAX_CANVAS_DESCRIPTION_LENGTH = 200;

export function CreateCanvasPage() {
  usePageTitle(["Create New Canvas"]);
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [nameError, setNameError] = useState("");
  const [submitError, setSubmitError] = useState<unknown>(null);

  const createMutation = useCreateCanvas(organizationId || "");
  const { data: workflowTemplates = [] } = useCanvasTemplates(organizationId || "");

  const handleSubmit = async () => {
    setNameError("");
    setSubmitError(null);

    if (!name.trim()) {
      setNameError("Name is required");
      return;
    }

    if (name.trim().length > MAX_CANVAS_NAME_LENGTH) {
      setNameError(`Name must be ${MAX_CANVAS_NAME_LENGTH} characters or less`);
      return;
    }

    if (!organizationId) {
      showErrorToast("Organization ID is missing");
      return;
    }

    try {
      const result = await createMutation.mutateAsync({
        name: name.trim(),
        description: description.trim() || undefined,
      });

      if (result?.data?.canvas?.metadata?.id) {
        navigate(`/${organizationId}/canvases/${result.data.canvas.metadata.id}`);
      }
    } catch (error) {
      const errorMessage = getApiErrorMessage(error, "Failed to create canvas");
      setSubmitError(error);
      showErrorToast(getUsageLimitToastMessage(error, "Failed to create canvas"));

      if (errorMessage.toLowerCase().includes("already") || errorMessage.toLowerCase().includes("exists")) {
        setNameError("A canvas with this name already exists");
      }
    }
  };

  const usageLimitNotice = submitError ? getUsageLimitNotice(submitError, organizationId) : null;

  const handleCancel = () => {
    navigate(`/${organizationId}`);
  };

  return (
    <div className="min-h-screen flex flex-col bg-slate-100 dark:bg-gray-900">
      <header className="bg-white border-b border-slate-950/15 px-4 h-12 flex items-center">
        <OrganizationMenuButton organizationId={organizationId || ""} />
      </header>
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className="w-full flex-grow-1">
          <div className="p-8 max-w-lg mx-auto">
            <div className="mb-6">
              <Heading level={2} className="!text-xl mb-1">
                Create New Canvas
              </Heading>
              <Text className="text-gray-800 dark:text-gray-400">
                Create a new canvas to orchestrate your DevOps work.
              </Text>
            </div>

            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-6">
              {usageLimitNotice ? <UsageLimitAlert notice={usageLimitNotice} /> : null}
              {submitError && !usageLimitNotice ? (
                <Alert variant="destructive">
                  <AlertTitle>Unable to create canvas</AlertTitle>
                  <AlertDescription>{getApiErrorMessage(submitError, "Failed to create canvas")}</AlertDescription>
                </Alert>
              ) : null}
              <Field>
                <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Canvas name *</Label>
                <Input
                  data-testid="canvas-name-input"
                  type="text"
                  autoComplete="off"
                  value={name}
                  onChange={(e) => {
                    if (e.target.value.length <= MAX_CANVAS_NAME_LENGTH) {
                      setName(e.target.value);
                    }
                    if (nameError) {
                      setNameError("");
                    }
                    if (submitError) {
                      setSubmitError(null);
                    }
                  }}
                  placeholder=""
                  className={`w-full ${nameError ? "border-red-500" : ""}`}
                  autoFocus
                  maxLength={MAX_CANVAS_NAME_LENGTH}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.shiftKey) {
                      e.preventDefault();
                      handleSubmit();
                    }
                  }}
                />
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {name.length}/{MAX_CANVAS_NAME_LENGTH} characters
                </div>
                {nameError && <div className="text-xs text-red-600 mt-1">{nameError}</div>}
              </Field>

              <Field>
                <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Description</Label>
                <Textarea
                  value={description}
                  onChange={(e) => {
                    if (e.target.value.length <= MAX_CANVAS_DESCRIPTION_LENGTH) {
                      setDescription(e.target.value);
                    }
                    if (submitError) {
                      setSubmitError(null);
                    }
                  }}
                  placeholder="Describe what it does (optional)"
                  rows={3}
                  className="w-full"
                  maxLength={MAX_CANVAS_DESCRIPTION_LENGTH}
                />
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {description.length}/{MAX_CANVAS_DESCRIPTION_LENGTH} characters
                </div>
              </Field>

              <div className="flex justify-start gap-3">
                <LoadingButton
                  onClick={handleSubmit}
                  disabled={!name.trim() || !!nameError}
                  loading={createMutation.isPending}
                  loadingText="Creating Canvas..."
                  data-testid="create-canvas-button"
                >
                  Create Canvas
                </LoadingButton>
                <Button variant="outline" onClick={handleCancel}>
                  Cancel
                </Button>
              </div>
            </div>
          </div>

          {workflowTemplates.length > 0 && (
            <div className="p-8 max-w-5xl mx-auto">
              <div className="relative flex items-center mb-4">
                <div
                  className="absolute left-0 right-0 top-1/2 -translate-y-1/2 border-t border-gray-300 dark:border-gray-600"
                  aria-hidden="true"
                />
                <Heading
                  level={3}
                  className="relative !text-sm pr-4 bg-gray-100 dark:bg-gray-900 text-gray-800 dark:text-white"
                >
                  Or start with an example
                </Heading>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {workflowTemplates.map((template) => (
                  <TemplateCard key={template.metadata?.id} template={template} organizationId={organizationId || ""} />
                ))}
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
interface TemplateCardProps {
  template: CanvasesCanvas;
  organizationId: string;
  showTags?: boolean;
}

function NodeCountLabel({ components, triggers }: { components: number; triggers: number }) {
  const parts: string[] = [];
  if (components > 0) parts.push(`${components} ${components === 1 ? "component" : "components"}`);
  if (triggers > 0) parts.push(`${triggers} ${triggers === 1 ? "trigger" : "triggers"}`);
  if (parts.length === 0) return null;
  return <div className="text-xs text-gray-500 dark:text-gray-500 mt-2">{parts.join(" · ")}</div>;
}

function IntegrationIcons({ integrations }: { integrations: string[] }) {
  if (integrations.length === 0) {
    return <span className="text-[11px] text-gray-400 dark:text-gray-500">No integrations needed</span>;
  }

  return (
    <div className="flex items-center gap-1.5 shrink-0">
      {integrations.map((name) => {
        const iconSrc = getIntegrationIconSrc(name);
        if (!iconSrc) return null;
        return (
          <Tooltip key={name}>
            <TooltipTrigger asChild>
              <span className="inline-block h-4 w-4 shrink-0">
                <img src={iconSrc} alt={name} className="h-full w-full object-contain" />
              </span>
            </TooltipTrigger>
            <TooltipContent side="bottom">
              <span className="capitalize">{name}</span>
            </TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}

function TagBadges({ tags }: { tags: string[] }) {
  if (tags.length === 0) return <div />;
  return (
    <div className="flex flex-wrap gap-1">
      {tags.map((tag) => (
        <Badge key={tag} variant="outline" className="text-[11px] px-1.5 py-0 text-gray-600 dark:text-gray-400">
          {tag}
        </Badge>
      ))}
    </div>
  );
}

export function TemplateCard({ template, organizationId, showTags = false }: TemplateCardProps) {
  const metadata = template.metadata;
  const nodes = template.spec?.nodes;
  const previewNodes = (nodes ?? []) as ComponentsNode[];
  const previewEdges = (template.spec?.edges ?? []) as ComponentsEdge[];
  const templateId = metadata?.id;

  if (!templateId) return null;

  const templateName = metadata?.name ?? "Untitled template";
  const description = metadata?.description ?? "";
  const tags = showTags ? getTemplateTags(metadata?.name) : [];
  const integrations = extractIntegrations(nodes);
  const { components, triggers } = countNodesByType(nodes);

  return (
    <Link
      to={`/${organizationId}/templates/${templateId}`}
      className="min-h-48 bg-white dark:bg-gray-800 rounded-md outline outline-slate-950/10 hover:shadow-md transition-shadow cursor-pointer group flex flex-col"
    >
      <div className="relative">
        <CanvasMiniMap nodes={previewNodes} edges={previewEdges} />
        <div
          className="absolute inset-0 flex items-center justify-center bg-white/80 rounded-t-md opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"
          aria-hidden
        >
          <span className="text-sm text-gray-800 dark:text-gray-900 bg-white/80 rounded-sm outline outline-1 outline-gray-400 dark:outline-gray-600 px-2 py-1">
            Preview
          </span>
        </div>
      </div>

      <div className="p-4 border-t border-gray-200 dark:border-gray-700 flex flex-col flex-1">
        <Heading
          level={3}
          className="!text-base font-medium text-gray-800 transition-colors mb-1 !leading-6 line-clamp-2"
        >
          {templateName}
        </Heading>

        {description ? (
          <Text className="text-[13px] !leading-normal text-left text-gray-800 dark:text-gray-400 line-clamp-3">
            {description}
          </Text>
        ) : null}

        <NodeCountLabel components={components} triggers={triggers} />

        <div className="mt-auto pt-3 flex items-end justify-between gap-2">
          <TagBadges tags={tags} />
          <IntegrationIcons integrations={integrations} />
        </div>
      </div>
    </Link>
  );
}

interface CanvasMiniMapProps {
  nodes?: ComponentsNode[];
  edges?: ComponentsEdge[];
}

function CanvasMiniMap({ nodes = [], edges = [] }: CanvasMiniMapProps) {
  const positionedNodes = nodes.filter(
    (node) => typeof node.position?.x === "number" && typeof node.position?.y === "number",
  ) as Array<ComponentsNode & { position: { x: number; y: number } }>;

  if (!positionedNodes.length) {
    return (
      <div className="p-4">
        <div className="h-28 w-full bg-transparent flex flex-col items-center justify-center pt-4 gap-1 text-[13px] text-gray-500">
          <Rainbow size={24} className="text-gray-500" />
          Canvas is empty
        </div>
      </div>
    );
  }

  const xs = positionedNodes.map((node) => node.position.x);
  const ys = positionedNodes.map((node) => node.position.y);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);
  const padding = 80;
  const width = Math.max(maxX - minX, 200) + padding * 2;
  const height = Math.max(maxY - minY, 200) + padding * 2;
  const viewBox = `${minX - padding} ${minY - padding} ${width} ${height}`;
  const nodeWidth = Math.min(Math.max(width * 0.08, 30), 80);
  const nodeHeight = nodeWidth * 0.45;

  const nodePositions = new Map<string, { x: number; y: number }>();
  positionedNodes.forEach((node) => {
    const id = node.id || node.name;
    if (!id) return;
    nodePositions.set(id, { x: node.position.x, y: node.position.y });
  });

  const drawableEdges =
    edges?.filter(
      (edge) => edge.sourceId && edge.targetId && nodePositions.has(edge.sourceId) && nodePositions.has(edge.targetId),
    ) || [];

  return (
    <div className="p-4 w-full overflow-hidden">
      <svg
        viewBox={viewBox}
        preserveAspectRatio="xMidYMid meet"
        className="w-full h-28 text-gray-500 dark:text-gray-400"
      >
        {drawableEdges.map((edge) => {
          const source = nodePositions.get(edge.sourceId!);
          const target = nodePositions.get(edge.targetId!);
          if (!source || !target) return null;
          return (
            <line
              key={`${edge.sourceId}-${edge.targetId}`}
              x1={source.x}
              y1={source.y}
              x2={target.x}
              y2={target.y}
              stroke="currentColor"
              strokeWidth={6}
              strokeLinecap="round"
              opacity={0.25}
            />
          );
        })}
        {positionedNodes.map((node) => {
          const id = node.id || node.name || `${node.position.x}-${node.position.y}`;
          return (
            <rect
              key={id}
              x={node.position.x - nodeWidth / 2}
              y={node.position.y - nodeHeight / 2}
              width={nodeWidth}
              height={nodeHeight}
              rx={8}
              ry={8}
              fill="#1f2937"
              opacity={1}
            />
          );
        })}
      </svg>
    </div>
  );
}
