import { useParams } from "react-router";

export const useOrganizationId = (): string | null => {
  const { organizationId } = useParams<{ organizationId: string }>();
  return organizationId || null;
};
