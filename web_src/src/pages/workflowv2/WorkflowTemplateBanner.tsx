import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { countNodesByType, extractIntegrations, getTemplateTags } from "@/pages/canvas/templateMetadata";
import { getIntegrationIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { ArrowLeft } from "lucide-react";
import { Link } from "react-router-dom";

type WorkflowTemplateBannerProps = {
  canvasName?: string;
  canvasDescription?: string;
  canvasNodes: ComponentsNode[];
  organizationId?: string;
  hasUnsavedChanges: boolean;
  onUseTemplate: () => void;
};

export function WorkflowTemplateBanner({
  canvasName,
  canvasDescription,
  canvasNodes,
  organizationId,
  hasUnsavedChanges,
  onUseTemplate,
}: WorkflowTemplateBannerProps) {
  const templateIntegrations = extractIntegrations(canvasNodes);
  const templateTags = getTemplateTags(canvasName);
  const templateNodeCounts = countNodesByType(canvasNodes);

  return (
    <div className="border-b border-orange-200 bg-orange-50 px-4 py-3">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 flex-1">
          <p className="mb-1 truncate text-sm font-medium text-gray-900">{canvasName || "Template"}</p>

          {canvasDescription ? <p className="mb-2 max-w-2xl text-[13px] text-gray-600">{canvasDescription}</p> : null}

          <div className="flex flex-wrap items-center gap-x-4 gap-y-1.5">
            {templateTags.length > 0 ? (
              <div className="flex flex-wrap gap-1">
                {templateTags.map((tag) => (
                  <Badge key={tag} variant="outline" className="bg-white px-1.5 py-0 text-[11px] text-gray-600">
                    {tag}
                  </Badge>
                ))}
              </div>
            ) : null}

            <div className="flex items-center gap-2">
              <span className="text-xs font-medium text-gray-500">Requires:</span>
              {templateIntegrations.length > 0 ? (
                <div className="flex items-center gap-2.5">
                  {templateIntegrations.map((name) => (
                    <TemplateIntegration key={name} name={name} />
                  ))}
                </div>
              ) : (
                <span className="text-xs text-gray-500">No integrations needed</span>
              )}
            </div>

            <span className="text-xs text-gray-500">
              {formatTemplateNodeCounts(templateNodeCounts.components, templateNodeCounts.triggers)}
            </span>
          </div>
        </div>

        <div className="flex shrink-0 flex-col items-end gap-2 self-start">
          <Link
            to={`/${organizationId}/templates`}
            className="flex items-center gap-1 text-sm text-gray-600 transition-colors hover:text-gray-900"
          >
            <ArrowLeft size={14} />
            <span>Back to templates</span>
          </Link>
          <Button size="sm" onClick={onUseTemplate}>
            {hasUnsavedChanges ? "Save changes to new canvas" : "Use template"}
          </Button>
        </div>
      </div>
    </div>
  );
}

function TemplateIntegration({ name }: { name: string }) {
  const iconSrc = getIntegrationIconSrc(name);
  if (!iconSrc) return null;

  return (
    <span className="inline-flex items-center gap-1">
      <span className="inline-block h-4 w-4 shrink-0">
        <img src={iconSrc} alt={name} className="h-full w-full object-contain" />
      </span>
      <span className="text-xs capitalize text-gray-600">{name}</span>
    </span>
  );
}

function formatTemplateNodeCounts(components: number, triggers: number) {
  const counts = [];
  if (components > 0) counts.push(`${components} ${components === 1 ? "component" : "components"}`);
  if (triggers > 0) counts.push(`${triggers} ${triggers === 1 ? "trigger" : "triggers"}`);
  return counts.join(" · ");
}
