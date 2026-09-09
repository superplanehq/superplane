import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DiscussionPRFeedbackSetupDialog } from "./DiscussionPRFeedbackSetupDialog";
import { PR_FEEDBACK_SOURCES } from "./prFeedbackSettingsModel";

const mocks = vi.hoisted(() => ({
  createHandler: vi.fn(),
  fetching: false,
  catalog: [
    { login: "coderabbitai", displayName: "coderabbitai[bot]" },
    { login: "bugbot", displayName: "bugbot[bot]" },
  ],
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useFactoryRepositoryReviewBots: () => ({
    data: mocks.catalog,
    isPending: false,
    isFetching: mocks.fetching,
    isError: false,
  }),
  useCreateFactoryPRFeedbackHandler: () => ({ mutateAsync: mocks.createHandler, isPending: false }),
}));

vi.mock("@/ui/componentSidebar/integrationIcons", () => ({
  IntegrationIcon: () => <span />,
}));

Element.prototype.scrollIntoView ??= () => undefined;

const discussionSource = PR_FEEDBACK_SOURCES.find((source) => source.id === "discussion")!;

const defaultCatalog = [
  { login: "coderabbitai", displayName: "coderabbitai[bot]" },
  { login: "bugbot", displayName: "bugbot[bot]" },
];

describe("DiscussionPRFeedbackSetupDialog", () => {
  beforeEach(() => {
    mocks.createHandler.mockReset();
    mocks.createHandler.mockResolvedValue({ id: "handler-1" });
    mocks.fetching = false;
    mocks.catalog.splice(0, mocks.catalog.length, ...defaultCatalog);
  });

  it("requires a mention by default and sends an empty mention when that is turned off", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    const onClose = vi.fn();
    render(
      <DiscussionPRFeedbackSetupDialog
        open
        organizationId="org-1"
        factoryId="factory-1"
        repository="acme/app"
        source={discussionSource}
        onClose={onClose}
        onCreated={onCreated}
      />,
    );

    expect(screen.getByTestId("discussion-setup-mention")).toHaveAttribute("aria-selected", "true");
    await user.click(screen.getByTestId("discussion-setup-mention"));
    expect(screen.getByTestId("discussion-setup-mention")).toHaveAttribute("aria-selected", "false");
    await user.click(screen.getByTestId("discussion-setup-continue"));
    await user.click(screen.getByTestId("discussion-setup-finish"));

    expect(mocks.createHandler).toHaveBeenCalledWith({
      source: "SOURCE_PULL_REQUEST_DISCUSSION",
      name: "Address PR feedback",
      settings: {
        subject: { repository: "acme/app" },
        discussion: {
          mention: "",
          ignoreBots: true,
          allowedBots: ["coderabbitai", "bugbot"],
        },
      },
    });
    expect(onCreated).toHaveBeenCalledWith("handler-1");
    expect(onClose).toHaveBeenCalled();
  });

  it("preselects detected bots and still finishes when they are cleared", async () => {
    const user = userEvent.setup();
    render(
      <DiscussionPRFeedbackSetupDialog
        open
        organizationId="org-1"
        factoryId="factory-1"
        repository="acme/app"
        source={discussionSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    await user.click(screen.getByTestId("discussion-setup-continue"));
    expect(screen.getByTestId("discussion-setup-bot-coderabbitai")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("discussion-setup-bot-bugbot")).toHaveAttribute("aria-selected", "true");
    await user.click(screen.getByTestId("discussion-setup-bot-coderabbitai"));
    await user.click(screen.getByTestId("discussion-setup-bot-bugbot"));
    await user.click(screen.getByTestId("discussion-setup-finish"));

    expect(mocks.createHandler).toHaveBeenCalledWith(
      expect.objectContaining({
        settings: expect.objectContaining({
          discussion: expect.objectContaining({
            mention: "@superplaneagent",
            allowedBots: [],
          }),
        }),
      }),
    );
  });

  it("keeps the modal title stable when continuing to review bots", async () => {
    const user = userEvent.setup();
    render(
      <DiscussionPRFeedbackSetupDialog
        open
        organizationId="org-1"
        factoryId="factory-1"
        repository="acme/app"
        source={discussionSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    expect(screen.getByRole("heading", { name: "Address PR feedback" })).toBeInTheDocument();
    await user.click(screen.getByTestId("discussion-setup-continue"));
    expect(screen.getByRole("heading", { name: "Address PR feedback" })).toBeInTheDocument();
    expect(screen.getByText("Review bots")).toBeInTheDocument();
    expect(screen.getByTestId("discussion-setup-bots-list")).toBeInTheDocument();
  });

  it("shows an empty catalog and lets the user add a bot login manually", async () => {
    mocks.catalog.splice(0);
    const user = userEvent.setup();
    const onCreated = vi.fn();
    render(
      <DiscussionPRFeedbackSetupDialog
        open
        organizationId="org-1"
        factoryId="factory-1"
        repository="acme/app"
        source={discussionSource}
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    await user.click(screen.getByTestId("discussion-setup-continue"));
    expect(screen.getByTestId("discussion-setup-bots-empty")).toBeInTheDocument();
    expect(screen.getByTestId("discussion-setup-finish")).toBeEnabled();

    await user.type(screen.getByTestId("discussion-setup-bot-manual"), "coderabbitai");
    await user.click(screen.getByTestId("discussion-setup-bot-add"));
    expect(screen.getByTestId("discussion-setup-bot-coderabbitai")).toHaveAttribute("aria-selected", "true");
    await user.click(screen.getByTestId("discussion-setup-finish"));

    expect(mocks.createHandler).toHaveBeenCalledWith(
      expect.objectContaining({
        settings: expect.objectContaining({
          discussion: expect.objectContaining({
            allowedBots: ["coderabbitai"],
          }),
        }),
      }),
    );
    expect(onCreated).toHaveBeenCalledWith("handler-1");
  });
});
