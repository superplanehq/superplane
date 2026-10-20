import type { SuperplaneComponentsNode } from "@/api-client";

export function dependabotRepositoryValue(
  node: SuperplaneComponentsNode | undefined,
  fallback?: string,
): string | undefined {
  const configured = node?.configuration?.repository;
  if (typeof configured === "string" && configured.trim() !== "") {
    return configured;
  }
  return fallback;
}
