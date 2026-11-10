import type { SuperplaneComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { getApiErrorMessage } from "@/lib/errors";
import { useState } from "react";

import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";

const NO_DEFAULT_REVIEWER = "__none__";

/** Policy stays on the built-in entrypoint so reset and generic saves share one contract. */
export function HumanReviewPolicySettings({
  node,
  reviewers = [],
  reviewersLoading = false,
  reviewersError,
  onRetryReviewers,
  onSave,
}: {
  node: SuperplaneComponentsNode;
  reviewers?: string[];
  reviewersLoading?: boolean;
  reviewersError?: string;
  onRetryReviewers?: () => void;
  onSave: (update: NodeConfigurationUpdate) => Promise<void> | void;
}) {
  const [enabled, setEnabled] = useState(node.configuration?.enabled === true);
  const [defaultReviewer, setDefaultReviewer] = useState(String(node.configuration?.defaultReviewer ?? "").trim());
  const [fallbackTeam, setFallbackTeam] = useState(String(node.configuration?.fallbackTeam ?? ""));
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string>();
  const fallbackMissing = enabled && !defaultReviewer && !fallbackTeam.trim();

  async function save() {
    setPending(true);
    setError(undefined);
    try {
      await onSave({
        nodeId: node.id ?? "",
        name: node.name ?? "Human review",
        integration: node.integration,
        concurrency: node.concurrency,
        configuration: { ...node.configuration, enabled, defaultReviewer, fallbackTeam: fallbackTeam.trim() },
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
        Only SuperPlane organization administrators can change Human review settings.
      </p>
      <div className="flex items-center justify-between gap-4">
        <Label htmlFor="human-review-enabled">Enable Human review</Label>
        <Switch id="human-review-enabled" checked={enabled} disabled={pending} onCheckedChange={setEnabled} />
      </div>
      <p className="text-sm text-muted-foreground">
        After merge confidence completes, an agent selects reviewers and decides whether approval is required.
      </p>
      <DefaultReviewerField
        value={defaultReviewer}
        onChange={setDefaultReviewer}
        reviewers={reviewers}
        loading={reviewersLoading}
        error={reviewersError}
        onRetry={onRetryReviewers}
        disabled={pending}
        missing={fallbackMissing}
      />
      {node.configuration?.fallbackTeam ? (
        <div className="space-y-2">
          <Label htmlFor="human-review-fallback-team">Fallback GitHub team (optional)</Label>
          <Input
            id="human-review-fallback-team"
            value={fallbackTeam}
            disabled={pending}
            onChange={(event) => setFallbackTeam(event.target.value)}
            aria-describedby="human-review-fallback-team-help"
          />
          <p id="human-review-fallback-team-help" className="text-sm text-muted-foreground">
            Use this team when no default reviewer is set and no suitable reviewer is found.
          </p>
        </div>
      ) : null}
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

function DefaultReviewerField({
  value,
  onChange,
  reviewers,
  loading,
  error,
  onRetry,
  disabled,
  missing,
}: {
  value: string;
  onChange: (value: string) => void;
  reviewers: string[];
  loading: boolean;
  error?: string;
  onRetry?: () => void;
  disabled: boolean;
  missing: boolean;
}) {
  // Keep the saved reviewer visible while GitHub access or the list changes.
  const choices = [...new Set([value, ...reviewers].filter(Boolean))].sort((left, right) => left.localeCompare(right));

  return (
    <div className="space-y-2">
      <Label htmlFor="human-review-default-reviewer">Default reviewer</Label>
      <Select
        value={value || NO_DEFAULT_REVIEWER}
        disabled={disabled || loading || choices.length === 0}
        onValueChange={(next) => onChange(next === NO_DEFAULT_REVIEWER ? "" : next)}
      >
        <SelectTrigger
          id="human-review-default-reviewer"
          className="w-full"
          aria-describedby="human-review-default-reviewer-help"
          aria-invalid={missing}
        >
          <SelectValue placeholder="Select a reviewer" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={NO_DEFAULT_REVIEWER}>No default reviewer</SelectItem>
          {choices.map((login) => (
            <SelectItem key={login} value={login}>
              @{login}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p id="human-review-default-reviewer-help" className="text-sm text-muted-foreground">
        Use this person when ownership and contribution history provide no suitable reviewer. A GitHub organization is
        not required. Pull request authors cannot approve their own pull requests.
      </p>
      <ReviewerListStatus
        loading={loading}
        error={error}
        empty={reviewers.length === 0}
        disabled={disabled}
        onRetry={onRetry}
      />
    </div>
  );
}

function ReviewerListStatus({
  loading,
  error,
  empty,
  disabled,
  onRetry,
}: {
  loading: boolean;
  error?: string;
  empty: boolean;
  disabled: boolean;
  onRetry?: () => void;
}) {
  if (loading) {
    return (
      <p role="status" className="text-sm text-muted-foreground">
        Loading reviewers…
      </p>
    );
  }
  if (error) {
    return (
      <div className="space-y-2">
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
        {onRetry ? (
          <Button type="button" variant="outline" size="sm" disabled={disabled} onClick={onRetry}>
            Try again
          </Button>
        ) : null}
      </div>
    );
  }
  if (!empty) {
    return null;
  }
  return (
    <div className="space-y-2">
      <p className="text-sm text-muted-foreground">
        No eligible reviewers are available. Give a collaborator write access to the repository, then try again.
      </p>
      {onRetry ? (
        <Button type="button" variant="outline" size="sm" disabled={disabled} onClick={onRetry}>
          Refresh reviewers
        </Button>
      ) : null}
    </div>
  );
}
