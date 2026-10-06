import type { SuperplaneComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useEffect, useState } from "react";

import { canvasNodeToPlanningReviewComponent, type CanvasSpecNode } from "../lib/columnCanvasAgent";
import type { NodeConfigurationUpdate } from "./nodeConfigurationCanvas";
import { PlanningReviewForm } from "./PlanningReviewForm";
import type { PlanningReviewComponent, PlanningReviewDraft } from "./planningReviewMockup";

const AGENT_RUNNER_COPY = {
  save: "Save Agent",
  saving: "Saving…",
} as const;

/** Agent settings for one runner step. Uses the same flat layout as the pull request step. */
export function AgentRunnerSettings({
  node,
  organizationId,
  factoryId,
  factoryKey,
  onSave,
}: {
  node: SuperplaneComponentsNode;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  onSave: (update: NodeConfigurationUpdate) => Promise<void> | void;
}) {
  const [draft, setDraft] = useState(() => draftFromNode(node));
  const [saving, setSaving] = useState(false);
  const nodeSignature = agentNodeSignature(node);

  useEffect(() => {
    setDraft(draftFromNode(node));
  }, [node, nodeSignature]);

  const component = draft.components[0];
  const dirty = component ? !sameAgent(node, component) : false;

  async function save() {
    if (!component || !dirty) {
      return;
    }
    setSaving(true);
    try {
      await onSave(updateFromComponent(node, component));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="min-h-0 flex-1 overflow-y-auto px-10 py-6">
      <PlanningReviewForm
        appearance="fields"
        showResources={false}
        draft={draft}
        onChange={setDraft}
        organizationId={organizationId}
        factoryId={factoryId}
        factoryKey={factoryKey}
      />
      <div className="mt-8 flex justify-end">
        <Button type="button" onClick={() => void save()} disabled={!dirty || saving} data-testid="agent-runner-save">
          {saving ? AGENT_RUNNER_COPY.saving : AGENT_RUNNER_COPY.save}
        </Button>
      </div>
    </div>
  );
}

function draftFromNode(node: SuperplaneComponentsNode): PlanningReviewDraft {
  const component = canvasNodeToPlanningReviewComponent(node as CanvasSpecNode);
  return { title: component.title, components: [component] };
}

function agentNodeSignature(node: SuperplaneComponentsNode) {
  return JSON.stringify({
    id: node.id,
    name: node.name,
    configuration: node.configuration,
    concurrency: node.concurrency,
  });
}

function sameAgent(node: SuperplaneComponentsNode, component: PlanningReviewComponent) {
  const current = canvasNodeToPlanningReviewComponent(node as CanvasSpecNode);
  return JSON.stringify(agentSnapshot(current)) === JSON.stringify(agentSnapshot(component));
}

function agentSnapshot(component: PlanningReviewComponent) {
  return {
    title: component.title,
    configuration: component.configuration,
    max: component.concurrency.max,
    key: component.concurrency.key,
  };
}

function updateFromComponent(
  node: SuperplaneComponentsNode,
  component: PlanningReviewComponent,
): NodeConfigurationUpdate {
  const next = nodeFromComponent(node, component);
  return {
    nodeId: node.id ?? "",
    name: next.name,
    configuration: next.configuration ?? {},
    concurrency: next.concurrency,
  };
}

function nodeFromComponent(
  node: SuperplaneComponentsNode,
  component: PlanningReviewComponent,
): SuperplaneComponentsNode {
  const parsedMax = Number.parseInt(component.concurrency.max, 10);
  const max = Number.isFinite(parsedMax) && parsedMax > 0 ? parsedMax : 1;
  const key = component.concurrency.key.trim();
  return {
    ...node,
    name: component.title,
    configuration: component.configuration,
    concurrency: key ? { max, key } : { max },
  };
}
