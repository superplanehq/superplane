import { useParams } from "react-router-dom";

export const useOrganizationId = (): string | null => {
  const { organizationId } = useParams<{ organizationId: string }>();
  return organizationId || null;
};
