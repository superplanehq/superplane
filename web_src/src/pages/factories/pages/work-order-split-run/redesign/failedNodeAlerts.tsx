import { Link } from "@/components/Link/link";
import { Alert, AlertAction, AlertDescription, AlertTitle } from "@/components/reui/alert";
import { Button } from "@/components/ui/button";
import { Bug } from "lucide-react";

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
  return (
    <Alert variant="destructive" data-testid="redesign-run-node-error">
      <NodeIcon iconSlug={node.iconSlug} />
      <AlertTitle>{node.name}</AlertTitle>
      <AlertDescription>
        <pre className="max-h-40 w-full overflow-auto font-mono text-[12px] leading-5 whitespace-pre-wrap text-destructive">
          {node.message}
        </pre>
      </AlertDescription>
      {runHref ? (
        <AlertAction>
          <Button asChild size="sm" variant="outline">
            <Link href={runHref}>
              <Bug className="size-3.5" aria-hidden />
              Debug
            </Link>
          </Button>
        </AlertAction>
      ) : null}
    </Alert>
  );
}
