import { useQuery } from "@tanstack/react-query";
import { actionsListActions, actionsDescribeAction } from "../api-client/sdk.gen";
import { withOrganizationHeader } from "../lib/withOrganizationHeader";
import { useExperimentalFeature } from "./useExperimentalFeature";
import { useExperimentalFeaturesRegistry } from "./useExperimentalFeatures";

export const componentKeys = {
  all: ["components"] as const,
  lists: () => [...componentKeys.all, "list"] as const,
  list: (orgId: string) => [...componentKeys.lists(), orgId] as const,
  details: () => [...componentKeys.all, "detail"] as const,
  detail: (orgId: string, name: string) => [...componentKeys.details(), orgId, name] as const,
};

export const useComponents = (organizationId: string) => {
  const { enabledExperimentalFeatures } = useExperimentalFeature(organizationId);
  const { data: expRegistry } = useExperimentalFeaturesRegistry();
  const experimentalFeatures =
    expRegistry?.features.filter((feature) => !feature.released).map((feature) => feature.id) || [];

  return useQuery({
    queryKey: componentKeys.list(organizationId),
    queryFn: async () => {
      const response = await actionsListActions(withOrganizationHeader({}));
      return response.data?.actions || [];
    },
    select: (data) => {
      return (data || []).filter((action) => {
        const featureID = action.name || "";
        const isExperimental = experimentalFeatures.includes(featureID);
        const isEnabled = enabledExperimentalFeatures.includes(featureID);

        return !isExperimental || isEnabled;
      });
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
