import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { CodexActionContext } from "../../codex/codexActionContext";
import { FactoriesLayoutContext } from "../../layout/factoriesLayoutContext";
import { AgentActivityView } from "./AgentActivityView";
import type { AgentActivity } from "./agentActivity";

function activityWithItems(items: AgentActivity["items"]): AgentActivity {
  return { id: "activity-1", provider: "codex", status: "running", sequence: 1, items, truncated: false };
}

describe("AgentActivityView authentication recovery", () => {
  it("opens Codex reconnection for a permanent refresh failure", async () => {
    const connect = vi.fn(async () => true);
    render(
      <FactoriesLayoutContext.Provider
        value={{
          organizationId: "org-1",
          factoryId: "factory-id",
          factoryKey: "EXAMPLE",
          factory: null,
          factories: [],
          openCreateWorkOrder: () => {},
        }}
      >
        <CodexActionContext.Provider value={connect}>
          <AgentActivityView
            activity={activityWithItems([
              {
                type: "notice",
                id: "failure",
                code: "chatgpt_reconnect_required",
                text: "SuperPlane could not refresh your ChatGPT connection. Reconnect ChatGPT to continue.",
              },
            ])}
          />
        </CodexActionContext.Provider>
      </FactoriesLayoutContext.Provider>,
    );

    await userEvent.setup().click(screen.getByRole("button", { name: "Reconnect ChatGPT" }));
    expect(connect).toHaveBeenCalledWith("org-1", "factory-id", "setup");
  });

  it("shows one recovery outcome for repeated historical authentication notices", () => {
    const previousPath = window.location.pathname;
    window.history.replaceState(null, "", "/org-1/workspaces/example/work-orders");
    try {
      render(
        <MemoryRouter>
          <AgentActivityView
            activity={activityWithItems([
              { type: "notice", id: "retry-1", code: "codex_error", text: "Reconnecting… 1/5" },
              { type: "notice", id: "retry-2", code: "codex_error", text: "Reconnecting… 2/5" },
              {
                type: "notice",
                id: "failure",
                code: "chatgpt_reconnect_required",
                text: "SuperPlane could not refresh your ChatGPT connection. Reconnect ChatGPT to continue.",
              },
              { type: "notice", id: "other", code: "codex_error", text: "A tool failed to read a file." },
            ])}
          />
        </MemoryRouter>,
      );
      expect(screen.getAllByRole("alert")).toHaveLength(2);
      expect(screen.getByRole("link", { name: "Reconnect ChatGPT" })).toHaveAttribute(
        "href",
        "/org-1/workspaces/example/settings/account/codex-usage",
      );
      expect(screen.queryByText(/Reconnecting…/)).not.toBeInTheDocument();
      expect(screen.getByText("A tool failed to read a file.")).toBeInTheDocument();
    } finally {
      window.history.replaceState(null, "", previousPath);
    }
  });

  it("hides refresh progress after the agent starts work", () => {
    const refreshing: AgentActivity["items"][number] = {
      type: "notice",
      id: "refreshing",
      code: "chatgpt_refreshing",
      text: "Refreshing your ChatGPT connection…",
    };
    const view = render(<AgentActivityView live activity={activityWithItems([refreshing])} />);
    expect(screen.getByRole("status")).toHaveTextContent("Refreshing your ChatGPT connection…");

    view.rerender(
      <AgentActivityView
        live
        activity={activityWithItems([
          refreshing,
          {
            type: "content",
            id: "answer",
            kind: "assistant",
            text: "I read the task.",
            status: "running",
            truncated: false,
          },
        ])}
      />,
    );
    expect(screen.queryByText("Refreshing your ChatGPT connection…")).not.toBeInTheDocument();
    expect(screen.getByText("I read the task.")).toBeInTheDocument();
  });
});
