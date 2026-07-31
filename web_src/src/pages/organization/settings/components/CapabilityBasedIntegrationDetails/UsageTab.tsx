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
    return <span className="text-sm text-content-secondary">—</span>;
  }

  const shown = labels.slice(0, MAX_COMPONENT_LABELS_SHOWN);
  const restCount = labels.length - shown.length;

  return (
    <span className="inline leading-relaxed text-sm text-content-primary">
      {shown.map((label, index) => (
        <Fragment key={label}>
          {index > 0 ? <span className="text-content-secondary">, </span> : null}
          <code className={INTEGRATION_INLINE_CODE_CLASSES}>{label}</code>
        </Fragment>
      ))}
      {restCount > 0 ? <span className="text-content-secondary"> + {restCount}</span> : null}
    </span>
  );
}

export function UsageTab({ organizationId, workflowGroups }: UsageTabProps) {
  return (
    <div className="rounded-lg border border-edge-default bg-surface-raised p-6">
      {workflowGroups.length > 0 ? (
        <>
          <p className="mb-3 text-sm text-content-secondary">
            This integration is currently used in the following canvases:
          </p>
          <div className="overflow-hidden rounded-md border border-edge-default">
            <div className="overflow-x-auto">
              <table className="table-fixed w-full min-w-[520px] divide-y divide-edge-subtle">
                <colgroup>
                  <col className="w-48 min-w-0" />
                  <col className="min-w-0" />
                  <col className="w-12 min-w-0" />
                </colgroup>
                <tbody className="divide-y divide-edge-subtle bg-surface-raised">
                  {workflowGroups.map((group) => {
                    const uniqueNames = Array.from(
                      new Set(group.nodes.map((node) => workflowComponentDisplayName(node.nodeName))),
                    ).sort((left, right) => left.localeCompare(right));

                    return (
                      <tr
                        key={group.canvasId}
                        className="cursor-pointer transition-colors hover:bg-action-neutral-hover/50"
                        onClick={() => window.open(appPath(organizationId, group.canvasId), "_blank")}
                        onKeyDown={(event) => {
                          if (event.key !== "Enter" && event.key !== " ") return;
                          event.preventDefault();
                          window.open(appPath(organizationId, group.canvasId), "_blank");
                        }}
                        tabIndex={0}
                        role="link"
                      >
                        <td className="max-w-48 min-w-0 px-4 py-3 align-middle text-sm font-medium text-content-primary">
                          <span className="line-clamp-2">{group.canvasName}</span>
                        </td>
                        <td className="min-w-0 px-4 py-3 align-middle">
                          <div className="min-w-0 break-words">
                            <UsesSummary labels={uniqueNames} />
                          </div>
                        </td>
                        <td className="align-middle px-2 py-3">
                          <ExternalLink className="mx-auto h-4 w-4 shrink-0 text-content-muted" aria-hidden />
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
        <p className="text-sm text-content-secondary">This integration is not used in any workflow yet.</p>
      )}
    </div>
  );
}
