import { AutomationChatContext } from "./useAutomationChatSessions";
import { useState } from "react";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it } from "vitest";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AutomationChatNavigation, AutomationTimelineChat } from "./AutomationChatNavigation";
import type { SplitRunPhase } from "./splitRunMocks";

const chats: SplitRunPhase[] = ["Strategy", "Review"].map((name) => ({
  id: name,
  runId: `run-${name}`,
  name,
  componentName: name,
  status: "passed",
  duration: "1m",
  artifacts: [],
  stream: [],
  canvasSteps: [],
}));

function TaskChats() {
  const [choice, setChoice] = useState<string | undefined>("run-Strategy");
  const selected = chats.find((chat) => chat.runId === choice);
  return (
    <TooltipProvider>
      <AutomationChatContext.Provider
        value={{ chats, hasHistory: true, choice: choice ?? "activity", selected, setChoice }}
      >
        <AutomationChatNavigation />
        {selected ? (
          <p>{selected.componentName} conversation</p>
        ) : (
          <section aria-label="Timeline cards">
            <AutomationTimelineChat phaseIds={["Strategy"]} />
            <AutomationTimelineChat phaseIds={["Review"]} />
            <AutomationTimelineChat phaseIds={["Bash"]} />
          </section>
        )}
      </AutomationChatContext.Provider>
    </TooltipProvider>
  );
}

it("shows the chat count and switches between named automation conversations", async () => {
  const user = userEvent.setup();
  render(<TaskChats />);
  await user.click(screen.getByRole("button", { name: "Strategy (2 active chats)" }));
  expect(screen.getByRole("menuitemradio", { name: "Strategy" })).toHaveAttribute("aria-checked", "true");
  await user.click(screen.getByRole("menuitemradio", { name: "Review" }));
  expect(screen.getByText("Review conversation")).toBeVisible();
  expect(screen.getByRole("button", { name: "Review (2 active chats)" })).toBeVisible();
});

it("opens the matching conversation from a timeline chat button", async () => {
  const user = userEvent.setup();
  render(<TaskChats />);
  await user.click(screen.getByRole("button", { name: "Timeline" }));
  const timeline = within(screen.getByRole("region", { name: "Timeline cards" }));
  expect(timeline.getAllByRole("button")).toHaveLength(2);
  await user.click(timeline.getByRole("button", { name: "Chat with Review" }));
  expect(screen.getByText("Review conversation")).toBeVisible();
});
