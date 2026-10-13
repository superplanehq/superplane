import { useSecrets } from "@/hooks/useSecrets";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

interface SecretsCountProps {
  organizationId: string;
}

/**
 * Total number of organization secrets, shown next to the Secrets heading.
 * Shares the query key with the Secrets list, so it adds no extra request.
 */
export function SecretsCount({ organizationId }: SecretsCountProps) {
  const { data: secrets, isLoading, isError } = useSecrets(organizationId, "DOMAIN_TYPE_ORGANIZATION");

  if (isLoading || isError || !secrets) {
    return null;
  }

  return (
    <span
      className={cn("text-sm font-medium tabular-nums text-gray-500", appDarkModeClasses.textSecondary)}
      data-testid="secrets-count"
    >
      {secrets.length}
    </span>
  );
}
