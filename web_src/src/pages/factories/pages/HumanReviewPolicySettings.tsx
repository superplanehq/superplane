import type { SuperplaneComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { getApiErrorMessage } from "@/lib/errors";
import { useState } from "react";

import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";

/** Policy stays on the built-in entrypoint so reset and generic saves share one contract. */
export function HumanReviewPolicySettings({
  node,
  onSave,
}: {
  node: SuperplaneComponentsNode;
  onSave: (update: NodeConfigurationUpdate) => Promise<void> | void;
}) {
  const [enabled, setEnabled] = useState(node.configuration?.enabled === true);
  const [fallbackTeam, setFallbackTeam] = useState(String(node.configuration?.fallbackTeam ?? ""));
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string>();
  const fallbackMissing = enabled && !fallbackTeam.trim();

  async function save() {
    setPending(true);
    setError(undefined);
    try {
      await onSave({
        nodeId: node.id ?? "",
        name: node.name ?? "Human review",
        integration: node.integration,
        concurrency: node.concurrency,
        configuration: { ...node.configuration, enabled, fallbackTeam: fallbackTeam.trim() },
      });
    } catch (cause) {
      setError(getApiErrorMessage(cause, "SuperPlane could not save Human review settings. Try again."));
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="min-h-0 flex-1 space-y-6 overflow-y-auto px-10 py-6">
      <p className="text-sm text-muted-foreground">
        Only organization administrators can change Human review settings.
      </p>
      <div className="flex items-center justify-between gap-4">
        <Label htmlFor="human-review-enabled">Enable Human review</Label>
        <Switch id="human-review-enabled" checked={enabled} disabled={pending} onCheckedChange={setEnabled} />
      </div>
      <p className="text-sm text-muted-foreground">
        After merge confidence completes, an agent selects reviewers and decides whether approval is required.
      </p>
      <div className="space-y-2">
        <Label htmlFor="human-review-fallback-team">Fallback GitHub team</Label>
        <Input
          id="human-review-fallback-team"
          value={fallbackTeam}
          placeholder="organization/team"
          disabled={pending}
          onChange={(event) => setFallbackTeam(event.target.value)}
          aria-describedby="human-review-fallback-team-help"
          aria-invalid={fallbackMissing}
        />
        <p id="human-review-fallback-team-help" className="text-sm text-muted-foreground">
          Use this team when ownership and contribution history provide no suitable reviewer. Select a team before
          enablement.
        </p>
      </div>
      {error ? (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      ) : null}
      <Button disabled={pending || fallbackMissing} onClick={() => void save()}>
        {pending ? "Saving…" : "Save Human review settings"}
      </Button>
    </div>
  );
}
