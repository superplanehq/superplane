import type { SuperplaneComponentsNode } from "@/api-client";
import { Switch } from "@/components/ui/switch";
import { useId, useRef, useState } from "react";

import { AgentRunnerSettings } from "./AgentRunnerSettings";
import { ImplementSettingRow } from "./ImplementSettingRow";
import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";

type SaveSettings = (update: NodeConfigurationUpdate) => Promise<void> | void;

export function ImplementConfiguration({
  agent,
  nodes,
  organizationId,
  factoryId,
  factoryKey,
  onSave,
}: {
  agent: SuperplaneComponentsNode;
  nodes: SuperplaneComponentsNode[];
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  onSave: SaveSettings;
}) {
  const saveRef = useRef(onSave);
  saveRef.current = onSave;
  const saveQueue = useRef(Promise.resolve());
  const save: SaveSettings = (update) => {
    const next = saveQueue.current.catch(() => undefined).then(() => saveRef.current(update));
    saveQueue.current = next;
    return next;
  };
  const pullRequest = nodes.find((node) =>
    ["github.createPullRequest", "bitbucket.createPullRequest"].includes(node.component ?? ""),
  );

  return (
    <section aria-label="Implement configuration" className="min-h-0 flex-1 overflow-y-auto bg-background">
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-6 py-8 sm:px-10">
        <AgentRunnerSettings
          key={agent.id}
          node={agent}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          onSave={save}
          view="implementation"
        />
        {pullRequest ? <PullRequestSettings key={pullRequest.id} node={pullRequest} onSave={save} /> : null}
      </div>
    </section>
  );
}

function PullRequestSettings({ node, onSave }: { node: SuperplaneComponentsNode; onSave: SaveSettings }) {
  const id = useId();
  const [pending, setPending] = useState<boolean | null>(null);
  const [failed, setFailed] = useState(false);
  const save = async (draft: boolean) => {
    setPending(draft);
    setFailed(false);
    try {
      await onSave({
        nodeId: node.id ?? "",
        name: node.name ?? "",
        configuration: { ...node.configuration, draft },
        integration: node.integration,
        concurrency: node.concurrency,
      });
    } catch {
      setFailed(true);
    } finally {
      setPending(null);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <ImplementSettingRow
        label="Create draft pull requests"
        description="Pull requests stay in draft until you mark them ready."
        htmlFor={id}
      >
        <Switch
          id={id}
          aria-describedby={`${id}-help`}
          checked={pending ?? node.configuration?.draft === true}
          disabled={pending !== null}
          onCheckedChange={(checked) => void save(checked)}
        />
      </ImplementSettingRow>
      {failed ? (
        <p role="alert" className="text-sm text-destructive">
          SuperPlane could not save this setting. Try again.
        </p>
      ) : null}
    </div>
  );
}
