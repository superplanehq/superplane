import { ExternalLink } from "lucide-react";

export type CapabilityIntegrationUsageGroup = {
  canvasId: string;
  canvasName: string;
  nodes: Array<{ nodeId: string; nodeName: string }>;
};

export interface UsageTabProps {
  organizationId: string;
  workflowGroups: CapabilityIntegrationUsageGroup[];
}

export function UsageTab({ organizationId, workflowGroups }: UsageTabProps) {
  return (
    <div className="rounded-lg border border-gray-300 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
      {workflowGroups.length > 0 ? (
        <>
          <p className="mb-3 text-sm text-gray-500 dark:text-gray-400">
            This integration is currently used in the following canvases:
          </p>
          <div className="space-y-2">
            {workflowGroups.map((group) => (
              <button
                key={group.canvasId}
                type="button"
                onClick={() => window.open(`/${organizationId}/canvases/${group.canvasId}`, "_blank")}
                className="flex w-full items-center gap-2 rounded-md border border-gray-300 bg-gray-50 p-3 text-left transition-colors hover:bg-gray-100 dark:border-gray-700 dark:bg-gray-800/50 dark:hover:bg-gray-800"
              >
                <div className="flex-1">
                  <p className="text-sm font-medium text-gray-800 dark:text-gray-100">Canvas: {group.canvasName}</p>
                  <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    Used in {group.nodes.length} node{group.nodes.length !== 1 ? "s" : ""}:{" "}
                    {group.nodes.map((node) => node.nodeName).join(", ")}
                  </p>
                </div>
                <ExternalLink className="h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500" />
              </button>
            ))}
          </div>
        </>
      ) : (
        <p className="text-sm text-gray-500 dark:text-gray-400">This integration is not used in any workflow yet.</p>
      )}
    </div>
  );
}
