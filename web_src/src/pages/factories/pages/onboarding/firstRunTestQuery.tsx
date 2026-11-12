import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement } from "react";

import { organizationKeys } from "@/hooks/useOrganizationData";

// First-run setup tests do not call the organization API. Seed the query so
// GitHub Enterprise stays off unless a test asks for that feature.
export function withFirstRunQuery(ui: ReactElement, enterpriseFeatures: string[] = []) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  queryClient.setQueryData(organizationKeys.details("org-1"), {
    id: "org-1",
    enterpriseFeatures,
  });
  return <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>;
}
