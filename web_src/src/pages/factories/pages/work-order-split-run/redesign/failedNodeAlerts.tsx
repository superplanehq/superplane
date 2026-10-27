import { Link } from "@/components/Link/link";
import { Alert, AlertAction, AlertDescription, AlertTitle } from "@/components/reui/alert";
import { Button } from "@/components/ui/button";

import { NodeIcon } from "./redesignShared";
import type { FailedNodeError } from "./failedNodeErrors";

export function FailedNodeAlerts({ nodes, runHref }: { nodes: FailedNodeError[]; runHref?: string | null }) {
  return (
    <div className="space-y-2">
      {nodes.map((node) => (
        <FailedNodeAlert key={node.id} node={node} runHref={runHref} />
      ))}
    </div>
  );
}

function FailedNodeAlert({ node, runHref }: { node: FailedNodeError; runHref?: string | null }) {
  const href =
    runHref && node.executionId
      ? `${runHref}${runHref.includes("?") ? "&" : "?"}activity=${encodeURIComponent(node.executionId)}`
      : runHref;
  return (
    <Alert variant="destructive" data-testid="redesign-run-node-error">
      <NodeIcon iconSlug={node.iconSlug} />
      <AlertTitle>{node.name}</AlertTitle>
      <AlertDescription>
        <pre className="max-h-40 w-full overflow-auto font-mono text-[12px] leading-5 whitespace-pre-wrap text-destructive">
          {node.message}
        </pre>
      </AlertDescription>
      {href ? (
        <AlertAction>
          <Button asChild size="sm" variant="outline">
            <Link href={href}>View run details</Link>
          </Button>
        </AlertAction>
      ) : null}
    </Alert>
  );
}
