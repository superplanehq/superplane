import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { expect, it, vi } from "bun:test";

const createHandler = vi.fn();
const updateHandler = vi.fn();
const deleteHandler = vi.fn();
vi.mock("@/api-client", () => ({
  factoriesCreateFactoryPrFeedbackHandler: createHandler,
  factoriesUpdateFactoryPrFeedbackHandler: updateHandler,
  factoriesDeleteFactoryPrFeedbackHandler: deleteHandler,
}));

import {
  useCreateFactoryPRFeedbackHandler,
  useUpdateFactoryPRFeedbackHandler,
  useDeleteFactoryPRFeedbackHandler,
} from "./useFactoryPRFeedbackData";

for (const change of ["create", "update", "delete"] as const) {
  it(`refreshes workspace warnings after PR feedback handler ${change}`, async () => {
    const client = new QueryClient();
    const key = ["integrations", "connected", "org-1", "workspace", "factory-1"];
    client.setQueryData(key, [{ metadata: { id: "github-1" } }]);
    createHandler.mockResolvedValue({ data: { handler: { id: "handler-1" } } });
    updateHandler.mockResolvedValue({ data: { handler: { id: "handler-1" } } });
    deleteHandler.mockResolvedValue({});
    const { result } = renderHook(
      () => ({
        create: useCreateFactoryPRFeedbackHandler("org-1", "factory-1"),
        update: useUpdateFactoryPRFeedbackHandler("org-1", "factory-1"),
        delete: useDeleteFactoryPRFeedbackHandler("org-1", "factory-1"),
      }),
      {
        wrapper: ({ children }: { children: ReactNode }) => createElement(QueryClientProvider, { client }, children),
      },
    );
    if (change === "create") await result.current.create.mutateAsync({ repository: "acme/api" });
    if (change === "update") await result.current.update.mutateAsync({ handlerId: "handler-1", settings: {} });
    if (change === "delete") await result.current.delete.mutateAsync("handler-1");
    expect(client.getQueryState(key)?.isInvalidated).toBe(true);
  });
}
