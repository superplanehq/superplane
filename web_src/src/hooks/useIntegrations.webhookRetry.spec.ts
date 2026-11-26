import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, describe, expect, it } from "bun:test";

import { client } from "@/api-client/client.gen";
import { integrationKeys, useRetryIntegrationWebhooks } from "./useIntegrations";

describe("retrying project webhooks", () => {
  const previousConfig = client.getConfig();

  afterEach(() => {
    cleanup();
    client.setConfig({ ...previousConfig, fetch: previousConfig.fetch });
  });

  it.each([200, 403])("attempts every webhook and refreshes integration status after HTTP %s", async (status) => {
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
    const queryKey = integrationKeys.connected("org-1");
    queryClient.setQueryData(queryKey, []);
    const requests: Request[] = [];
    let finishLastRequest: (response: Response) => void = () => {};
    const lastResponse = new Promise<Response>((resolve) => {
      finishLastRequest = resolve;
    });
    const fetchWebhooks: typeof fetch = async (input) => {
      const request = input instanceof Request ? input : new Request(input);
      requests.push(request);
      if (request.url.includes("/checks/retry")) return lastResponse;
      const responseStatus = request.url.includes("/issues/retry") ? status : 200;
      return new Response(JSON.stringify(responseStatus === 200 ? {} : { message: "Webhook retry was denied" }), {
        status: responseStatus,
        headers: { "Content-Type": "application/json" },
      });
    };
    client.setConfig({ baseUrl: "http://localhost", fetch: fetchWebhooks, throwOnError: false });

    function Wrapper({ children }: { children: ReactNode }) {
      return createElement(QueryClientProvider, { client: queryClient }, children);
    }
    const { result } = renderHook(() => useRetryIntegrationWebhooks("org-1"), { wrapper: Wrapper });
    const targets = [
      { integrationId: "github-1", webhookId: "issues" },
      { integrationId: "github-1", webhookId: "reviews" },
      { integrationId: "github-2", webhookId: "checks" },
    ];
    const outcome = result.current.mutateAsync(targets).then(
      () => undefined,
      (error: unknown) => error,
    );

    await waitFor(() => expect(requests).toHaveLength(targets.length));
    expect(
      requests.map((request) => ({
        path: new URL(request.url).pathname,
        method: request.method,
        organizationId: request.headers.get("x-organization-id"),
      })),
    ).toEqual(
      targets.map(({ integrationId, webhookId }) => ({
        path: `/api/v1/organizations/org-1/integrations/${integrationId}/webhooks/${webhookId}/retry`,
        method: "POST",
        organizationId: "org-1",
      })),
    );
    expect(queryClient.getQueryState(queryKey)?.isInvalidated).toBe(false);

    let error: unknown;
    await act(async () => {
      finishLastRequest(new Response("{}", { headers: { "Content-Type": "application/json" } }));
      error = await outcome;
    });
    expect(error).toEqual(status === 200 ? undefined : { message: "Webhook retry was denied" });
    await waitFor(() => expect(result.current.status).toBe(status === 200 ? "success" : "error"));
    expect(queryClient.getQueryState(queryKey)?.isInvalidated).toBe(true);
  });
});
