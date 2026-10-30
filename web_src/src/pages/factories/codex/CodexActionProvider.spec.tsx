import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, render, renderHook, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { useState } from "react";

import { meCancelCodexConnection, meDescribeCodexConnection, type FactoriesFactoryCodexConnection } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getApiErrorMessage } from "@/lib/errors";
import { unmockedSrc } from "@/test/unmockedModule";
import { factoryCodexConnectionKey, useFactoryCodexConnection } from "../pages/onboarding/useFactoryCodexConnection";
import { CodexActionProvider } from "./CodexActionProvider";
import { CodexConnectionSetupDialog } from "./CodexConnectionSetupDialog";
import { isCodexActionCancelled, useCodexActionGuard } from "./codexActionContext";

vi.mock("@/api-client", () => ({
  ...unmockedSrc<Record<string, unknown>>("api-client/index.ts"),
  meDescribeCodexConnection: vi.fn(),
  meCancelCodexConnection: vi.fn(),
}));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, currentUserId: "me", isLoading: false }),
}));
vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({ isLoading: false, has: () => true }),
}));

const connectionRequired = { details: [{ domain: "superplane.com", reason: "CODEX_CONNECTION_REQUIRED" }] };
let connection: FactoriesFactoryCodexConnection;

function TaskDraft({ save }: { save: (title: string) => Promise<void> }) {
  const [title, setTitle] = useState("My task draft");
  const [status, setStatus] = useState("");
  const guard = useCodexActionGuard("org", "factory");
  return (
    <>
      <Input aria-label="Title" value={title} onChange={(event) => setTitle(event.target.value)} />
      <Button
        onClick={() => {
          void guard(() => save(title))
            .then(() => setStatus("Saved"))
            .catch((error: unknown) => {
              if (!isCodexActionCancelled(error)) setStatus(getApiErrorMessage(error));
            });
        }}
      >
        Create task
      </Button>
      <p role="status">{status}</p>
    </>
  );
}

function DismissibleCodexSetup({ onConnected }: { onConnected: () => void }) {
  const [open, setOpen] = useState(true);
  return (
    <CodexConnectionSetupDialog
      organizationId="org"
      factoryId="factory"
      open={open}
      onOpenChange={setOpen}
      onConnected={onConnected}
    />
  );
}

function mount(save: (title: string) => Promise<void>) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <CodexActionProvider>
        <TaskDraft save={save} />
      </CodexActionProvider>
    </QueryClientProvider>,
  );
  return client;
}

describe("Codex assignment continuation", () => {
  beforeEach(() => {
    connection = { status: "disconnected", eligible: true };
    vi.mocked(meCancelCodexConnection)
      .mockReset()
      .mockResolvedValue({ data: {} } as never);
    vi.mocked(meDescribeCodexConnection)
      .mockReset()
      .mockImplementation(async () => ({ data: { connection } }) as never);
  });

  it("preserves the action input and resumes once after connection", async () => {
    const save = vi.fn().mockRejectedValueOnce(connectionRequired).mockResolvedValue(undefined);
    const client = mount(save);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Create task" }));
    expect(await screen.findByRole("heading", { name: "Connect Codex to continue" })).toBeInTheDocument();
    expect(
      screen.getByText(
        "This workspace uses Codex subscriptions. Connect your Codex account to take ownership of this task.",
      ),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Connect Codex" }));
    for (const status of ["pending", "finishing", "ready"]) {
      connection = { status, eligible: true, reservationHeld: true, availableModels: ["gpt-5.4"] };
      await act(async () => client.setQueryData(factoryCodexConnectionKey("org", "factory", "me"), connection));
      expect(save).toHaveBeenCalledTimes(1);
      expect(screen.getByRole("dialog")).toBeInTheDocument();
    }
    connection = { status: "ready", eligible: true, reservationHeld: false, availableModels: ["gpt-5.4"] };
    await act(async () => client.setQueryData(factoryCodexConnectionKey("org", "factory", "me"), connection));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Saved"));
    expect(save.mock.calls).toEqual([["My task draft"], ["My task draft"]]);
    await act(async () => client.invalidateQueries({ queryKey: factoryCodexConnectionKey("org", "factory", "me") }));
    expect(save).toHaveBeenCalledTimes(2);
  });

  it.each(["prompt", "setup"] as const)("cancels from %s without losing the draft", async (stage) => {
    const save = vi.fn().mockRejectedValue(connectionRequired);
    mount(save);
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: "Create task" }));
    await screen.findByRole("heading", { name: "Connect Codex to continue" });
    if (stage === "setup") await user.click(screen.getByRole("button", { name: "Connect Codex" }));
    await user.click(screen.getByRole("button", { name: stage === "setup" ? /^Close setup$/ : /^Cancel$/ }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(screen.getByRole("textbox", { name: "Title" })).toHaveValue("My task draft");
    expect(save).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("status")).toBeEmptyDOMElement();
  });

  it.each(["close", "unmount"] as const)("does not continue after %s during the final refresh", async (dismissal) => {
    const onConnected = vi.fn();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = render(
      <QueryClientProvider client={client}>
        <DismissibleCodexSetup onConnected={onConnected} />
      </QueryClientProvider>,
    );
    // Another workspace keeps the shared status query active after this dialog closes.
    const observer = renderHook(() => useFactoryCodexConnection("org", "other-workspace", true), {
      wrapper: ({ children }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>,
    });
    const user = userEvent.setup();
    await waitFor(() => expect(screen.getByRole("button", { name: "Connect ChatGPT" })).toBeEnabled());
    connection = { status: "ready", eligible: true, availableModels: ["gpt-5.4"] };
    let completeRefresh!: (response: { data: { connection: FactoriesFactoryCodexConnection } }) => void;
    const refresh = new Promise<{ data: { connection: FactoriesFactoryCodexConnection } }>((resolve) => {
      completeRefresh = resolve;
    });
    vi.mocked(meDescribeCodexConnection).mockImplementation(() => refresh as never);
    await act(async () => client.setQueryData(factoryCodexConnectionKey("org", "factory", "me"), connection));
    await waitFor(() =>
      expect(client.getQueryState(factoryCodexConnectionKey("org", "factory", "me"))?.fetchStatus).toBe("fetching"),
    );
    if (dismissal === "unmount") view.unmount();
    else await user.click(screen.getByRole("button", { name: /^Close setup$/ }));
    await act(async () => completeRefresh({ data: { connection } }));
    await waitFor(() => expect(client.isFetching()).toBe(0));
    expect(onConnected).not.toHaveBeenCalled();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    observer.unmount();
  });

  it.each(["Close setup", "Cancel sign-in"])(
    "cancels the shared sign-in only through its explicit control: %s",
    async (action) => {
      connection = { status: "pending", eligible: true, reservationHeld: true };
      const onConnected = vi.fn();
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      render(
        <QueryClientProvider client={client}>
          <DismissibleCodexSetup onConnected={onConnected} />
        </QueryClientProvider>,
      );
      const button = await screen.findByRole("button", { name: action });
      await waitFor(() => expect(button).toBeEnabled());
      await userEvent.setup().click(button);
      if (action === "Cancel sign-in") await waitFor(() => expect(meCancelCodexConnection).toHaveBeenCalledTimes(1));
      else expect(meCancelCodexConnection).not.toHaveBeenCalled();
      expect(onConnected).not.toHaveBeenCalled();
    },
  );

  it("keeps an active-run ownership denial and does not offer a connection", async () => {
    mount(vi.fn().mockRejectedValue({ message: "Stop the active run before changing its owner." }));
    await userEvent.setup().click(screen.getByRole("button", { name: "Create task" }));
    await waitFor(() =>
      expect(screen.getByRole("status")).toHaveTextContent("Stop the active run before changing its owner."),
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
