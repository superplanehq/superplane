import type { ConfigurationField, IntegrationSetupStepDefinition, OrganizationsIntegration } from "@/api-client";

function isMissingValue(value: unknown): boolean {
  if (value === null || value === undefined) {
    return true;
  }

  if (typeof value === "string") {
    return value.trim() === "";
  }

  if (Array.isArray(value)) {
    return value.length === 0;
  }

  return false;
}

export function getMissingRequiredFields(
  fields: Array<ConfigurationField> | undefined,
  values: Record<string, unknown>,
): Set<string> {
  const missing = new Set<string>();
  if (!fields) {
    return missing;
  }

  fields.forEach((field) => {
    if (!field.name || !field.required) {
      return;
    }

    if (isMissingValue(values[field.name])) {
      missing.add(field.name);
    }
  });

  return missing;
}

export function getNextIntegrationName(baseName: string, existingNames: Set<string>): string {
  const normalizedBaseName = baseName.trim() || "integration";
  if (!existingNames.has(normalizedBaseName)) {
    return normalizedBaseName;
  }

  let suffix = 2;
  let candidate = `${normalizedBaseName}-${suffix}`;
  while (existingNames.has(candidate)) {
    suffix += 1;
    candidate = `${normalizedBaseName}-${suffix}`;
  }

  return candidate;
}

export function getCurrentSetupStep(
  integration: OrganizationsIntegration | null,
): IntegrationSetupStepDefinition | null {
  return integration?.status?.setupState?.currentStep ?? null;
}

export function canRevertSetupStep(integration: OrganizationsIntegration | null): boolean {
  const previousSteps = integration?.status?.setupState?.previousSteps ?? [];
  return previousSteps.length > 0;
}

/** Used for grouped capability checkbox UI and for clearing/selecting groups in sync. */
export function getGroupToggleState(capabilityNames: string[], selected: ReadonlySet<string>): "all" | "some" | "none" {
  if (capabilityNames.length === 0) {
    return "none";
  }

  let count = 0;
  for (const name of capabilityNames) {
    if (selected.has(name)) {
      count++;
    }
  }

  if (count === 0) {
    return "none";
  }
  if (count === capabilityNames.length) {
    return "all";
  }
  return "some";
}

/** Key that bumps when resumed describe timestamps or setup step advances. */
export function getResumeDescribeStateKey(describe: OrganizationsIntegration): string {
  const setupStepName = describe.status?.setupState?.currentStep?.name ?? "";
  return `${describe.metadata?.updatedAt ?? ""}:${setupStepName}`;
}

/**
 * Applies resume syncing when describe matches integration id and the state key differs from {@link lastKeyRef}.
 */
export function applyResumeDescribeIfChanged(
  setupIntegrationId: string | undefined,
  resumeDescribe: OrganizationsIntegration | null | undefined,
  lastKeyRef: { current: string | null },
  onApply: (describe: OrganizationsIntegration) => void,
): void {
  if (setupIntegrationId == null || setupIntegrationId === "") {
    return;
  }

  if (!resumeDescribe) {
    return;
  }

  if (resumeDescribe.metadata?.id !== setupIntegrationId) {
    return;
  }

  const resumeKey = getResumeDescribeStateKey(resumeDescribe);
  if (lastKeyRef.current === resumeKey) {
    return;
  }

  lastKeyRef.current = resumeKey;
  onApply(resumeDescribe);
}
