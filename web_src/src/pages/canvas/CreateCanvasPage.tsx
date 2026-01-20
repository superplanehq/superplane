import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Field, Label } from "../../components/Fieldset/fieldset";
import { Heading } from "../../components/Heading/heading";
import { Input } from "../../components/Input/input";
import { Text } from "../../components/Text/text";
import { Textarea } from "../../components/ui/textarea";
import { Button } from "../../components/ui/button";
import { useCreateWorkflow, useWorkflowTemplates } from "../../hooks/useWorkflowData";
import { showErrorToast } from "../../utils/toast";
import type { ComponentsEdge, ComponentsNode } from "@/api-client";
import { Rainbow, FileUp } from "lucide-react";
import * as yaml from "js-yaml";

const MAX_CANVAS_NAME_LENGTH = 50;
const MAX_CANVAS_DESCRIPTION_LENGTH = 200;

export function CreateCanvasPage() {
  usePageTitle(["Create New Canvas"]);
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [nameError, setNameError] = useState("");
  const [yamlContent, setYamlContent] = useState("");
  const [yamlError, setYamlError] = useState("");
  const [useYamlMode, setUseYamlMode] = useState(false);

  const createMutation = useCreateWorkflow(organizationId || "");
  const { data: workflowTemplates = [] } = useWorkflowTemplates(organizationId || "");

  const handleSubmit = async () => {
    setNameError("");
    setYamlError("");

    if (useYamlMode) {
      // Parse and validate YAML
      if (!yamlContent.trim()) {
        setYamlError("YAML content is required");
        return;
      }

      try {
        const parsed = yaml.load(yamlContent) as any;

        if (!parsed) {
          setYamlError("Invalid YAML format");
          return;
        }

        if (parsed.kind !== "Canvas") {
          setYamlError("YAML must be of kind: Canvas");
          return;
        }

        if (!parsed.metadata?.name) {
          setYamlError("Canvas name is required in metadata");
          return;
        }

        if (!organizationId) {
          showErrorToast("Organization ID is missing");
          return;
        }

        const result = await createMutation.mutateAsync({
          name: parsed.metadata.name,
          description: parsed.metadata.description || undefined,
          nodes: parsed.spec?.nodes || [],
          edges: parsed.spec?.edges || [],
        });

        if (result?.data?.workflow?.metadata?.id) {
          navigate(`/${organizationId}/workflows/${result.data.workflow.metadata.id}`);
        }
      } catch (error) {
        console.error("Error creating canvas from YAML:", error);
        const errorMessage = (error as Error)?.message || error?.toString() || "Failed to parse YAML";
        setYamlError(errorMessage);
        showErrorToast(errorMessage);
      }
      return;
    }

    // Regular form submission
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

      if (result?.data?.workflow?.metadata?.id) {
        navigate(`/${organizationId}/workflows/${result.data.workflow.metadata.id}`);
      }
    } catch (error) {
      console.error("Error creating canvas:", error);
      const errorMessage = (error as Error)?.message || error?.toString() || "Failed to create canvas";

      showErrorToast(errorMessage);

      if (errorMessage.toLowerCase().includes("already") || errorMessage.toLowerCase().includes("exists")) {
        setNameError("A canvas with this name already exists");
      }
    }
  };

  const handleCancel = () => {
    navigate(`/${organizationId}`);
  };

  return (
    <div className="min-h-screen flex flex-col bg-gray-100 dark:bg-gray-900">
      <header className="bg-white border-b border-border px-4 h-12 flex items-center">
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

            <div className="bg-white dark:bg-gray-950 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-6">
              <div className="flex justify-between items-center mb-4">
                <div>
                  <Label className="text-sm font-medium text-gray-700 dark:text-gray-300">Creation mode</Label>
                </div>
                <div className="flex gap-2">
                  <Button
                    variant={!useYamlMode ? "default" : "outline"}
                    size="sm"
                    onClick={() => {
                      setUseYamlMode(false);
                      setYamlError("");
                    }}
                    type="button"
                  >
                    Form
                  </Button>
                  <Button
                    variant={useYamlMode ? "default" : "outline"}
                    size="sm"
                    onClick={() => {
                      setUseYamlMode(true);
                      setNameError("");
                    }}
                    type="button"
                  >
                    <FileUp className="w-4 h-4 mr-1" />
                    Import YAML
                  </Button>
                </div>
              </div>

              {useYamlMode ? (
                <Field>
                  <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    Canvas YAML *
                  </Label>
                  <Textarea
                    data-testid="canvas-yaml-input"
                    value={yamlContent}
                    onChange={(e) => {
                      setYamlContent(e.target.value);
                      if (yamlError) {
                        setYamlError("");
                      }
                    }}
                    placeholder={`apiVersion: v1
kind: Canvas
metadata:
  name: my-canvas
  description: Optional description
spec:
  nodes: []
  edges: []`}
                    rows={15}
                    className={`w-full font-mono text-sm ${yamlError ? "border-red-500" : ""}`}
                  />
                  {yamlError && <div className="text-xs text-red-600 mt-1">{yamlError}</div>}
                  <div className="text-xs text-gray-500 dark:text-gray-400 mt-2">
                    Paste your Canvas YAML definition here.{" "}
                    <a
                      href="/docs/examples/canvas-with-nodes.yml"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="underline hover:text-gray-700 dark:hover:text-gray-200"
                    >
                      See example
                    </a>
                  </div>
                </Field>
              ) : (
                <>
                  <Field>
                    <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Canvas name *
                    </Label>
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
                    <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                      Description
                    </Label>
                    <Textarea
                      value={description}
                      onChange={(e) => {
                        if (e.target.value.length <= MAX_CANVAS_DESCRIPTION_LENGTH) {
                          setDescription(e.target.value);
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
                </>
              )}

              <div className="flex justify-start gap-3">
                <Button
                  onClick={handleSubmit}
                  disabled={
                    (useYamlMode && !yamlContent.trim()) ||
                    (!useYamlMode && !name.trim()) ||
                    createMutation.isPending ||
                    !!nameError ||
                    !!yamlError
                  }
                  data-testid="create-canvas-button"
                >
                  {createMutation.isPending ? "Creating Canvas..." : useYamlMode ? "Import Canvas" : "Create Canvas"}
                </Button>
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
                  <TemplateCard
                    key={template.metadata?.id}
                    template={template}
                    organizationId={organizationId || ""}
                    navigate={navigate}
                  />
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
  template: any;
  organizationId: string;
  navigate: any;
}

function TemplateCard({ template, organizationId, navigate }: TemplateCardProps) {
  const previewNodes = (template.spec?.nodes || []) as ComponentsNode[];
  const previewEdges = (template.spec?.edges || []) as ComponentsEdge[];
  const templateId = template.metadata?.id;

  if (!templateId) return null;

  const handleNavigate = () => navigate(`/${organizationId}/templates/${templateId}`);

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={(event) => {
        if (event.defaultPrevented) return;
        handleNavigate();
      }}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          handleNavigate();
        }
      }}
      className="min-h-48 bg-white dark:bg-gray-950 rounded-md outline outline-slate-950/10 hover:shadow-md transition-shadow cursor-pointer group"
    >
      <div className="flex flex-col h-full">
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

        <div className="p-4 border-t border-gray-200">
          <Heading
            level={3}
            className="!text-base font-medium text-gray-800 transition-colors mb-1 !leading-6 line-clamp-2"
          >
            {template.metadata?.name || "Untitled template"}
          </Heading>

          {template.metadata?.description ? (
            <div>
              <Text className="text-[13px] !leading-normal text-left text-gray-800 dark:text-gray-400 line-clamp-3">
                {template.metadata.description}
              </Text>
            </div>
          ) : null}
        </div>
      </div>
    </div>
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
