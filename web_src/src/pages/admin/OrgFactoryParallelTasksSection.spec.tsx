import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { OrgFactoryParallelTasksSection } from "./OrgFactoryParallelTasksSection";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("OrgFactoryParallelTasksSection", () => {
  it("shows the installation default when the organization has no override", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => {
        return new Response(
          JSON.stringify({
            max_parallel_factory_tasks: null,
            installation_default: 50,
            effective: 50,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }),
    );

    render(<OrgFactoryParallelTasksSection orgId="org-1" />);

    const input = await screen.findByTestId("admin-org-max-parallel-factory-tasks");
    expect(input).toHaveValue("");
    expect(input).toHaveAttribute("placeholder", "Installation default (50)");
    expect(screen.getByText("Effective limit: 50")).toBeInTheDocument();
  });

  it("sends null when the override is cleared", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "PATCH") {
        return new Response(
          JSON.stringify({
            max_parallel_factory_tasks: null,
            installation_default: 50,
            effective: 50,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }
      return new Response(
        JSON.stringify({
          max_parallel_factory_tasks: 3,
          installation_default: 50,
          effective: 3,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<OrgFactoryParallelTasksSection orgId="org-1" />);

    const input = await screen.findByTestId("admin-org-max-parallel-factory-tasks");
    expect(input).toHaveValue("3");
    await userEvent.clear(input);
    await userEvent.click(screen.getByTestId("admin-org-max-parallel-factory-tasks-save"));

    const patchCall = fetchMock.mock.calls.find((call) => {
      const init = call[1] as RequestInit | undefined;
      return init?.method === "PATCH";
    });
    expect(patchCall).toBeTruthy();
    expect(JSON.parse(String((patchCall?.[1] as RequestInit).body))).toEqual({
      max_parallel_factory_tasks: null,
    });
  });
});
