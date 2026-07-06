import { ExternalLink } from "lucide-react";
import { Fragment } from "react";
import { appPath } from "@/lib/appPaths";
import { INTEGRATION_INLINE_CODE_CLASSES } from "./lib";

export type CapabilityIntegrationUsageGroup = {
  canvasId: string;
  canvasName: string;
  nodes: Array<{ nodeId: string; nodeName: string }>;
};

export interface UsageTabProps {
  organizationId: string;
  workflowGroups: CapabilityIntegrationUsageGroup[];
}

const MAX_COMPONENT_LABELS_SHOWN = 3;

/** Drops a leading integration id prefix (e.g. `github.getIssue` → `getIssue`). */
function workflowComponentDisplayName(nodeName: string): string {
  const dot = nodeName.indexOf(".");
  if (dot === -1) return nodeName;
  const rest = nodeName.slice(dot + 1);
  return rest.length > 0 ? rest : nodeName;
}

function UsesSummary({ labels }: { labels: string[] }) {
  if (labels.length === 0) {
    return <span className="text-sm text-gray-500 dark:text-gray-400">—</span>;
  }

  const shown = labels.slice(0, MAX_COMPONENT_LABELS_SHOWN);
  const restCount = labels.length - shown.length;

  return (
    <span className="inline leading-relaxed text-sm text-gray-800 dark:text-gray-200">
      {shown.map((label, index) => (
        <Fragment key={label}>
          {index > 0 ? <span className="text-gray-500 dark:text-gray-400">, </span> : null}
          <code className={INTEGRATION_INLINE_CODE_CLASSES}>{label}</code>
        </Fragment>
      ))}
      {restCount > 0 ? <span className="text-gray-600 dark:text-gray-400"> + {restCount}</span> : null}
    </span>
  );
}

export function UsageTab({ organizationId, workflowGroups }: UsageTabProps) {
  return (
    <div className="rounded-lg border border-gray-300 bg-white p-6 dark:border-gray-800 dark:bg-gray-900">
      {workflowGroups.length > 0 ? (
        <>
          <p className="mb-3 text-sm text-gray-500 dark:text-gray-400">
            This integration is currently used in the following canvases:
          </p>
          <div className="overflow-hidden rounded-md border border-gray-300 dark:border-gray-600">
            <div className="overflow-x-auto">
              <table className="table-fixed w-full min-w-[520px] divide-y divide-gray-200 dark:divide-gray-800">
                <colgroup>
                  <col className="w-48 min-w-0" />
                  <col className="min-w-0" />
                  <col className="w-12 min-w-0" />
                </colgroup>
                <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-900">
                  {workflowGroups.map((group) => {
                    const uniqueNames = Array.from(
                      new Set(group.nodes.map((node) => workflowComponentDisplayName(node.nodeName))),
                    ).sort((left, right) => left.localeCompare(right));

                    return (
                      <tr
                        key={group.canvasId}
                        className="cursor-pointer transition-colors hover:bg-gray-50 dark:hover:bg-gray-800/50"
                        onClick={() => window.open(appPath(organizationId, group.canvasId), "_blank")}
                        onKeyDown={(event) => {
                          if (event.key !== "Enter" && event.key !== " ") return;
                          event.preventDefault();
                          window.open(appPath(organizationId, group.canvasId), "_blank");
                        }}
                        tabIndex={0}
                        role="link"
                      >
                        <td className="max-w-48 min-w-0 px-4 py-3 align-middle text-sm font-medium text-gray-800 dark:text-gray-100">
                          <span className="line-clamp-2">{group.canvasName}</span>
                        </td>
                        <td className="min-w-0 px-4 py-3 align-middle">
                          <div className="min-w-0 break-words">
                            <UsesSummary labels={uniqueNames} />
                          </div>
                        </td>
                        <td className="align-middle px-2 py-3">
                          <ExternalLink
                            className="mx-auto h-4 w-4 shrink-0 text-gray-400 dark:text-gray-500"
                            aria-hidden
                          />
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        </>
      ) : (
        <p className="text-sm text-gray-500 dark:text-gray-400">This integration is not used in any workflow yet.</p>
      )}
    </div>
  );
}
