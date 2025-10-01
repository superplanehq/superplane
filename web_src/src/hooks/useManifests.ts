import { useQuery } from '@tanstack/react-query';
import { manifestServiceGetManifests } from '../api-client';
import type { SuperplaneTypeManifest } from '../api-client';

export type ManifestCategory = 'executor' | 'event_source';

export function useManifests(category: ManifestCategory) {
  return useQuery({
    queryKey: ['manifests', category],
    queryFn: async () => {
      const response = await manifestServiceGetManifests({
        query: { category },
      });
      return response.data?.manifests || [];
    },
    staleTime: 5 * 60 * 1000, // 5 minutes - manifests don't change often
  });
}

export function useManifestByType(category: ManifestCategory, type: string | undefined) {
  const { data: manifests, ...rest } = useManifests(category);

  const manifest = manifests?.find((m: SuperplaneTypeManifest) => m.type === type);

  return {
    manifest,
    manifests,
    ...rest,
  };
}
