import { useQuery } from "@tanstack/react-query";
import { actionsListActions, actionsDescribeAction } from "../api-client/sdk.gen";
import { withOrganizationHeader } from "../lib/withOrganizationHeader";

export const componentKeys = {
  all: ["components"] as const,
  lists: () => [...componentKeys.all, "list"] as const,
  list: (orgId: string) => [...componentKeys.lists(), orgId] as const,
  details: () => [...componentKeys.all, "detail"] as const,
  detail: (orgId: string, name: string) => [...componentKeys.details(), orgId, name] as const,
};

export const useComponents = (organizationId: string) => {
  return useQuery({
    queryKey: componentKeys.list(organizationId),
    queryFn: async () => {
      const response = await actionsListActions(withOrganizationHeader({}));
      return response.data?.actions || [];
    },
    enabled: !!organizationId,
  });
};

export const useComponent = (organizationId: string, componentName: string) => {
  return useQuery({
    queryKey: componentKeys.detail(organizationId, componentName),
    queryFn: async () => {
      const response = await actionsDescribeAction(
        withOrganizationHeader({
          path: { name: componentName },
        }),
      );
      return response.data?.action;
    },
    enabled: !!organizationId && !!componentName,
  });
};
