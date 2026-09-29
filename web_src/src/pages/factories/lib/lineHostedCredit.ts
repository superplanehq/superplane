const AGENT_RUNNER_COMPONENTS = new Set(["runnerSuperPlane", "runnerClaudeCode", "runnerCodex", "runnerOpenRouter"]);

export type LineRunnerNode = {
  component?: string;
  configuration?: { credentials?: { source?: string } } | Record<string, unknown>;
};

/**
 * Whether one agent node spends SuperPlane hosted credit.
 * SuperPlane Agent always does. Other runners do only when credentials are hosted.
 * A node that is not an agent runner returns null.
 */
export function runnerUsesHostedCredit(node: LineRunnerNode): boolean | null {
  const component = node.component?.trim() ?? "";
  if (!AGENT_RUNNER_COMPONENTS.has(component)) {
    return null;
  }
  if (component === "runnerSuperPlane") {
    return true;
  }
  const source = credentialSource(node.configuration);
  if (source === "secret" || source === "integration") {
    return false;
  }
  return true;
}

/**
 * True when any agent on the line spends hosted credit.
 * A line with only BYOK runners, or with no agent runner, returns false.
 */
export function lineUsesHostedCredit(nodes: readonly LineRunnerNode[]): boolean {
  return nodes.some((node) => runnerUsesHostedCredit(node) === true);
}

export function lineAppIds(line: { steps?: Array<{ app?: { app?: string } }> } | undefined): string[] {
  const ids = (line?.steps ?? []).flatMap((step) => {
    const appId = step.app?.app?.trim() ?? "";
    return appId ? [appId] : [];
  });
  return [...new Set(ids)];
}

function credentialSource(configuration: LineRunnerNode["configuration"]): string {
  if (!configuration || typeof configuration !== "object" || !("credentials" in configuration)) {
    return "";
  }
  const credentials = configuration.credentials;
  if (!credentials || typeof credentials !== "object" || !("source" in credentials)) {
    return "";
  }
  const source = credentials.source;
  return typeof source === "string" ? source.trim() : "";
}
