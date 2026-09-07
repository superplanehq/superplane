import { factoriesUpdateFactoryRepository, type FactoriesUpdateFactoryRepositoryBody } from "@/api-client";
import { factoryQueryKeys } from "@/hooks/useFactoryData";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { useMutation, useQueryClient } from "@tanstack/react-query";

export function useFactoryRepository(organizationId: string, factoryId: string) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (input: FactoriesUpdateFactoryRepositoryBody) => {
      const response = await factoriesUpdateFactoryRepository(
        withOrganizationHeader({
          organizationId,
          path: { id: factoryId },
          body: input,
        }),
      );
      if (!response.data?.factory) {
        throw new Error("Failed to update workspace repository");
      }
      return response.data.factory;
    },
    onSuccess: (factory) => {
      queryClient.setQueryData(factoryQueryKeys.detail(organizationId, factoryId), factory);
      void queryClient.invalidateQueries({ queryKey: factoryQueryKeys.list(organizationId) });
    },
  });
}
