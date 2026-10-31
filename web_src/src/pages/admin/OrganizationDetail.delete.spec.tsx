import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes } from "react-router";
import OrganizationDetail from "./OrganizationDetail";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "account-1", name: "Admin" } }),
}));

function renderPage() {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter initialEntries={["/admin/organizations/org-1"]}>
        <Routes>
          <Route path="/admin/organizations/:orgId" element={<OrganizationDetail />} />
          <Route path="/admin" element={<p>Organization list</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("OrganizationDetail deletion", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("requires the name, prevents dismissal while deleting, and returns to the list", async () => {
    const user = userEvent.setup();
    let completeDelete: (response: Response) => void = () => {};
    const pendingDelete = new Promise<Response>((resolve) => {
      completeDelete = resolve;
    });
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "DELETE") return pendingDelete;
      return Response.json({
        id: "org-1",
        name: "Acme",
        slug: "acme",
        description: "",
        canvas_count: 0,
        task_count: 0,
        done_task_count: 0,
        member_count: 0,
        workspace_count: 0,
        repository_count: 0,
      });
    });
    vi.stubGlobal("fetch", fetchMock);
    renderPage();

    await user.click(await screen.findByRole("button", { name: "Delete organization" }));
    const dialog = within(screen.getByRole("dialog"));
    const confirm = dialog.getByRole("button", { name: "Delete organization" });
    expect(confirm).toBeDisabled();
    await user.type(dialog.getByLabelText("Organization name"), "Other");
    expect(confirm).toBeDisabled();
    await user.clear(dialog.getByLabelText("Organization name"));
    await user.type(dialog.getByLabelText("Organization name"), "Acme");
    await user.click(confirm);
    expect(dialog.getByRole("button", { name: "Deleting..." })).toBeDisabled();
    expect(dialog.getByRole("button", { name: "Keep organization" })).toBeDisabled();
    await user.keyboard("{Escape}");
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith("/admin/api/organizations/org-1", {
      method: "DELETE",
      credentials: "include",
    });
    completeDelete(Response.json({ status: "deleted" }));
    expect(await screen.findByText("Organization list")).toBeInTheDocument();
  });

  it("shows a billing rejection and allows retry or cancellation", async () => {
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "DELETE") {
          return Response.json({ code: 10, message: "Cancel the subscription before deletion." }, { status: 409 });
        }
        return Response.json({
          id: "org-1",
          name: "Acme",
          slug: "acme",
          description: "",
          canvas_count: 0,
          task_count: 0,
          done_task_count: 0,
          member_count: 0,
          workspace_count: 0,
          repository_count: 0,
        });
      }),
    );
    renderPage();
    await user.click(await screen.findByRole("button", { name: "Delete organization" }));
    const dialog = within(screen.getByRole("dialog"));
    await user.type(dialog.getByLabelText("Organization name"), "Acme");
    await user.click(dialog.getByRole("button", { name: "Delete organization" }));
    expect(await dialog.findByRole("alert")).toHaveTextContent("Cancel the subscription before deletion.");
    expect(dialog.getByRole("button", { name: "Delete organization" })).toBeEnabled();
    expect(screen.queryByText("Organization list")).not.toBeInTheDocument();
    await user.click(dialog.getByRole("button", { name: "Keep organization" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Delete organization" }));
    expect(within(screen.getByRole("dialog")).getByLabelText("Organization name")).toHaveValue("");
  });
});
