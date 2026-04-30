import { Link } from "@/components/Link/link";

interface ServiceAccountCreatorLinkProps {
  organizationId: string;
  creator?: { id?: string; name?: string };
}

export function ServiceAccountCreatorLink({ organizationId, creator }: ServiceAccountCreatorLinkProps) {
  const membersHref = `/${organizationId}/settings/members`;

  if (creator?.id && creator?.name) {
    return (
      <Link
        href={membersHref}
        className="text-sm text-gray-800 underline underline-offset-2 dark:text-gray-200"
        data-testid="sa-created-by-link"
      >
        {creator.name}
      </Link>
    );
  }

  return <span className="text-sm text-gray-500 dark:text-gray-400">—</span>;
}
