import type { FactoriesFactoryRepositoryStatusCheck } from "@/api-client";

export type ChecksPRFeedbackSetupStep = "checks" | "tools";

export const CHECKS_HANDLER_SKIP_INTEGRATIONS = new Set(["github", "gitlab"]);

export function requiredStatusCheckNames(catalog: FactoriesFactoryRepositoryStatusCheck[]): string[] {
  return catalog.flatMap((check) => (check.required && check.name ? [check.name] : []));
}

export function catalogStatusCheckNames(catalog: FactoriesFactoryRepositoryStatusCheck[]): string[] {
  return catalog.flatMap((check) => (check.name?.trim() ? [check.name.trim()] : []));
}

export function suggestedIntegrationsForChecks(
  catalog: FactoriesFactoryRepositoryStatusCheck[],
  selectedNames: string[],
): string[] {
  const selected = new Set(selectedNames.map((name) => name.toLowerCase()));
  const names = new Set<string>();
  for (const check of catalog) {
    if (!check.name || !selected.has(check.name.toLowerCase())) {
      continue;
    }
    const integration = check.suggestedIntegration?.trim().toLowerCase();
    if (!integration || CHECKS_HANDLER_SKIP_INTEGRATIONS.has(integration)) {
      continue;
    }
    names.add(integration);
  }
  return [...names];
}
