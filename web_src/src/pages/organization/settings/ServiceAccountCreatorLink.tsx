import { Link } from "@/components/Link/link";

interface ServiceAccountCreatorLinkProps {
  organizationId: string;
  createdBy?: { id?: string; name?: string } | null;
}

export function ServiceAccountCreatorLink({ organizationId, createdBy }: ServiceAccountCreatorLinkProps) {
  if (!createdBy?.id || !createdBy?.name?.trim()) {
    return <span className="text-sm text-gray-500 dark:text-gray-400">—</span>;
  }

  return (
    <Link
      href={`/${organizationId}/settings/members#member-${createdBy.id}`}
      className="cursor-pointer text-sm text-gray-800 !underline underline-offset-2 dark:text-gray-200"
      data-testid="sa-created-by-link"
    >
      {createdBy.name}
    </Link>
  );
}
