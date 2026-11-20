import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type { CustomAutomationColumn } from "@/pages/factories/lib/customAutomation";
import { factoryCustomAutomationKey } from "./useFactoryCustomAutomationData";
import { factoryAppsKey } from "./useFactoryData";

const collectionPath = "/api/v2/automations";

type AutomationSource = {
  id: string;
  factory_id: string;
  stage: CustomAutomationColumn;
  name: string;
  yaml: string;
  revision: number;
};

type AutomationSourceDiagnostic = {
  path: string;
  line: number;
  column: number;
  message: string;
};

export class AutomationSourceError extends Error {
  constructor(
    message: string,
    readonly formVersion?: string,
  ) {
    super(message);
  }
}

type AutomationSourceProblem = {
  can_edit_in_form?: boolean;
  live_version_id?: string;
  message?: string;
  diagnostics?: AutomationSourceDiagnostic[];
};

type SaveInput = { stage: CustomAutomationColumn; yaml: string };

function automationSourceKey(organizationId: string, factoryId: string, automationId?: string) {
  return ["automation-source", organizationId, factoryId, automationId] as const;
}

export function useCustomAutomationSource(organizationId: string, factoryId: string, automationId?: string) {
  return useQuery({
    queryKey: automationSourceKey(organizationId, factoryId, automationId),
    queryFn: ({ signal }) => automationSourceRequest<AutomationSource>(organizationId, automationId, { signal }),
    enabled: Boolean(organizationId && factoryId && automationId),
    staleTime: 0,
  });
}

export function useSaveCustomAutomationSource(organizationId: string, factoryId: string, automationId?: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ stage, yaml }: SaveInput) =>
      automationSourceRequest<AutomationSource>(organizationId, automationId, {
        method: automationId ? "PUT" : "POST",
        body: JSON.stringify(automationId ? { yaml } : { factory_id: factoryId, stage, yaml }),
      }),
    onSuccess: (source) => {
      client.setQueryData(automationSourceKey(organizationId, factoryId, source.id), source);
      void client.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
      void client.invalidateQueries({ queryKey: factoryCustomAutomationKey(organizationId, factoryId, source.id) });
    },
  });
}

export function useDeleteCustomAutomationSource(organizationId: string, factoryId: string, automationId?: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: () => automationSourceRequest<void>(organizationId, automationId, { method: "DELETE" }),
    onSuccess: () => {
      client.removeQueries({ queryKey: automationSourceKey(organizationId, factoryId, automationId) });
      void client.invalidateQueries({ queryKey: factoryAppsKey(organizationId, factoryId) });
    },
  });
}

async function automationSourceRequest<T>(organizationId: string, automationId: string | undefined, init: RequestInit) {
  const path = automationId ? `${collectionPath}/${encodeURIComponent(automationId)}` : collectionPath;
  const response = await fetch(
    path,
    withOrganizationHeader({
      ...init,
      organizationId,
      credentials: "include" as const,
      headers: init.body ? { "Content-Type": "application/json" } : undefined,
    }),
  );
  if (!response.ok) {
    const problem = (await response.json().catch(() => null)) as AutomationSourceProblem | null;
    const details = problem?.diagnostics?.map(({ path, line, column, message }) => {
      const location = line > 0 && column > 0 ? `Line ${line}, column ${column}` : "";
      return [path, location, message].filter(Boolean).join(" · ");
    });
    throw new AutomationSourceError(
      details?.length ? details.join("\n") : getApiErrorMessage(problem, "SuperPlane could not complete the request."),
      problem?.can_edit_in_form === true ? problem.live_version_id : undefined,
    );
  }
  return init.method === "DELETE" ? (undefined as T) : ((await response.json()) as T);
}
