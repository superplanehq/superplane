export type PendingStartTemplateRun = {
  nodeId: string;
  templateName: string;
  initialData: string;
};

let pending: PendingStartTemplateRun | null = null;

export function setPendingStartTemplateRun(args: PendingStartTemplateRun): void {
  pending = args;
}

/**
 * Returns pending template run context for this node and clears the slot if it matches.
 */
export function takePendingStartTemplateRunForNode(
  nodeId: string,
): Pick<PendingStartTemplateRun, "templateName" | "initialData"> | undefined {
  if (!pending || pending.nodeId !== nodeId) {
    return undefined;
  }
  const { templateName, initialData } = pending;
  pending = null;
  return { templateName, initialData };
}
