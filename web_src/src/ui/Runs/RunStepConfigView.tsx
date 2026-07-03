import { useMemo } from "react";
import type { SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { DetailBox } from "./RunStepAccordion";
import { RUN_NODE_ICON_SIZE, RunNodeIcon } from "./RunNodeIcon";
import { getMockStepConfig } from "./runStepConfigMock";

const noop = () => {};

/**
 * Read-only rendition of the node editing form, shown when the run panel is in
 * "Step configuration" mode. It renders the same field controls as the edit
 * sidebar (`ConfigurationFieldRenderer`) but non-interactive (`pointer-events-none`
 * + no-op `onChange`), so the configuration reads exactly like the edit form.
 * Schema/values come from the prototype mock catalog since the run panel has no
 * live catalog access.
 */
export function RunStepConfigView({
  nodeId,
  workflowNodes,
  componentIconMap = {},
}: {
  nodeId: string;
  workflowNodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
}) {
  const workflowNode = useMemo(() => workflowNodes.find((node) => node.id === nodeId), [workflowNodes, nodeId]);

  const { fields, values } = useMemo(() => {
    const mock = getMockStepConfig(workflowNode?.component);
    const nodeConfiguration = (workflowNode?.configuration as Record<string, unknown> | undefined) ?? {};
    return { fields: mock.fields, values: { ...mock.values, ...nodeConfiguration } };
  }, [workflowNode]);

  const title = workflowNode?.name || "Step";
  const visibleFields = fields.filter((field) => field.name && field.name !== "customName");

  return (
    <div className="bg-slate-50 px-3 py-3">
      <DetailBox
        title="Configuration"
        actions={
          <span className="flex items-center gap-1.5 text-[11px] font-medium normal-case tracking-normal text-slate-500">
            <RunNodeIcon
              iconSrc={getHeaderIconSrc(workflowNode?.component)}
              iconSlug={workflowNode?.component ? componentIconMap[workflowNode.component] : undefined}
              alt={title}
              size={RUN_NODE_ICON_SIZE}
              className="h-3.5 w-3.5"
            />
            <span className="max-w-[12rem] truncate">{title}</span>
          </span>
        }
      >
        {visibleFields.length > 0 ? (
          <div className="pointer-events-none space-y-4" aria-disabled="true">
            {visibleFields.map((field) => (
              <ConfigurationFieldRenderer
                key={field.name}
                field={field}
                value={values[field.name!]}
                onChange={noop}
                allValues={values}
                allowExpressions={false}
              />
            ))}
          </div>
        ) : (
          <p className="text-[13px] text-slate-400">No configuration for this step.</p>
        )}
      </DetailBox>
    </div>
  );
}
