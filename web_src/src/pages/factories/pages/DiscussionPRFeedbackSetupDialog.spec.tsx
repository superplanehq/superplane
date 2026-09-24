import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { DiscussionPRFeedbackSetupDialog } from "./DiscussionPRFeedbackSetupDialog";
import { PR_FEEDBACK_SOURCES } from "./prFeedbackSettingsModel";
import { discussionBotSettings } from "./useDiscussionPRFeedbackSetup";

const mocks = vi.hoisted(() => ({
  createHandler: vi.fn(),
  fetching: false,
  catalog: [
    { type: "review_bot", id: "coderabbitai", name: "coderabbitai[bot]" },
    { type: "review_bot", id: "bugbot", name: "bugbot[bot]" },
  ],
}));

vi.mock("@/hooks/useFactoryPRFeedbackData", () => ({
  useCreateFactoryPRFeedbackHandler: () => ({ mutateAsync: mocks.createHandler, isPending: false }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: mocks.catalog,
    isPending: false,
    isFetching: mocks.fetching,
    isError: false,
  }),
}));

vi.mock("@/ui/componentSidebar/integrationIcons", () => ({
  IntegrationIcon: () => <span />,
}));

Element.prototype.scrollIntoView ??= () => undefined;

const discussionSource = PR_FEEDBACK_SOURCES.find((source) => source.id === "discussion")!;

const defaultCatalog = [
  { type: "review_bot", id: "coderabbitai", name: "coderabbitai[bot]" },
  { type: "review_bot", id: "bugbot", name: "bugbot[bot]" },
];

async function openBotsStep(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByTestId("discussion-setup-continue"));
}

async function chooseAddressBots(user: ReturnType<typeof userEvent.setup>) {
  await openBotsStep(user);
  await user.click(screen.getByRole("radio", { name: /Address bot comments/ }));
}

describe("discussionBotSettings", () => {
  it("maps each bot mode to ignore and allowlist settings", () => {
    expect(discussionBotSettings("ignore", ["coderabbitai"])).toEqual({
      ignoreBots: true,
      allowedBots: [],
    });
    expect(discussionBotSettings("address", ["coderabbitai"])).toEqual({
      ignoreBots: true,
      allowedBots: ["coderabbitai"],
    });
  });
});

function stubReducedMotion() {
  vi.stubGlobal("matchMedia", (query: string) => ({
    matches: query.includes("prefers-reduced-motion"),
    media: query,
    onchange: null,
    addListener() {},
    removeListener() {},
    addEventListener() {},
    removeEventListener() {},
    dispatchEvent: () => false,
  }));
}

describe("DiscussionPRFeedbackSetupDialog", () => {
  beforeEach(() => {
    stubReducedMotion();
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
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/app"
        source={discussionSource}
        onClose={onClose}
        onCreated={onCreated}
      />,
    );

    expect(screen.getByRole("radio", { name: /Require @superplaneagent/ })).toBeChecked();
    expect(screen.getByTestId("discussion-setup-preview-comment")).toHaveAttribute(
      "data-comment",
      "@superplaneagent fix this and make no mistakes",
    );
    expect(screen.getByTestId("discussion-setup-preview-comment")).toHaveTextContent("@superplaneagent");
    expect(screen.getByTestId("discussion-setup-preview-caption")).toHaveTextContent(
      "A mention starts pull request feedback.",
    );
    await user.click(screen.getByRole("radio", { name: /Start from any human comment/ }));
    expect(screen.getByRole("radio", { name: /Start from any human comment/ })).toBeChecked();
    expect(screen.getByRole("radio", { name: /Require @superplaneagent/ })).not.toBeChecked();
    expect(screen.getByTestId("discussion-setup-preview-comment")).toHaveAttribute(
      "data-comment",
      "fix this and make no mistakes",
    );
    expect(screen.getByTestId("discussion-setup-preview-comment")).not.toHaveTextContent("@superplaneagent");
    expect(screen.getByTestId("discussion-setup-preview-caption")).toHaveTextContent(
      "Any human comment starts pull request feedback.",
    );
    await openBotsStep(user);
    expect(screen.getByRole("radio", { name: /Ignore bot comments/ })).toBeChecked();
    expect(screen.queryByTestId("discussion-setup-bots-list")).not.toBeInTheDocument();
    expect(screen.getByTestId("discussion-setup-preview-comment")).toHaveTextContent(
      "This test does not cover the new error path.",
    );
    expect(screen.getByTestId("discussion-setup-preview-bot-outcome")).toHaveTextContent("Ignored");
    expect(screen.getByTestId("discussion-setup-preview-caption")).toHaveTextContent(
      "Bot comments do not start pull request feedback.",
    );
    await user.click(screen.getByTestId("discussion-setup-finish"));

    expect(mocks.createHandler).toHaveBeenCalledWith({
      source: "SOURCE_PULL_REQUEST_DISCUSSION",
      name: "Address PR feedback",
      settings: {
        subject: { repository: "acme/app" },
        discussion: {
          mention: "",
          ignoreBots: true,
          allowedBots: [],
        },
      },
    });
    expect(onCreated).toHaveBeenCalledWith("handler-1");
    expect(onClose).toHaveBeenCalled();
  });

  it("shows review bots only when addressing bot comments", async () => {
    const user = userEvent.setup();
    render(
      <DiscussionPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/app"
        source={discussionSource}
        onClose={vi.fn()}
        onCreated={vi.fn()}
      />,
    );

    await chooseAddressBots(user);
    expect(screen.getByTestId("discussion-setup-preview-comment")).toHaveAttribute("data-author", "coderabbitai[bot]");
    expect(screen.getByTestId("discussion-setup-preview-bot-outcome")).toHaveTextContent("Fixing...");
    expect(screen.getByTestId("discussion-setup-preview-caption")).toHaveTextContent(
      "Selected bots start pull request feedback.",
    );
    expect(screen.getByTestId("discussion-setup-bots-found")).toBeInTheDocument();
    expect(screen.getByTestId("discussion-setup-bot-coderabbitai")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("discussion-setup-bot-bugbot")).toHaveAttribute("aria-selected", "true");
    await user.click(screen.getByTestId("discussion-setup-bot-coderabbitai"));
    expect(screen.getByTestId("discussion-setup-preview-comment")).toHaveAttribute("data-author", "bugbot[bot]");
    await user.click(screen.getByTestId("discussion-setup-bot-bugbot"));
    expect(screen.getByTestId("discussion-setup-preview-comment")).toHaveAttribute("data-author", "coderabbitai[bot]");
    expect(screen.getByTestId("discussion-setup-preview-bot-outcome")).toHaveTextContent("Ignored");
    await user.click(screen.getByTestId("discussion-setup-finish"));

    expect(mocks.createHandler).toHaveBeenCalledWith(
      expect.objectContaining({
        settings: expect.objectContaining({
          discussion: expect.objectContaining({
            mention: "@superplaneagent",
            ignoreBots: true,
            allowedBots: [],
          }),
        }),
      }),
    );
  });

  it("shows a step question for human comments and AI comments", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
      <DiscussionPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/app"
        source={discussionSource}
        onClose={onClose}
        onCreated={vi.fn()}
      />,
    );

    expect(screen.getByTestId("discussion-setup-back")).toHaveTextContent("Back to board");
    expect(screen.getByRole("heading", { name: "How to handle human comments?" })).toBeInTheDocument();
    await openBotsStep(user);
    expect(screen.getByRole("heading", { name: "How to handle AI comments?" })).toBeInTheDocument();
    expect(
      screen.getByText(
        "AI review bots often leave comments on pull requests. Choose whether SuperPlane should handle that feedback.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /Ignore bot comments/ })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /Address bot comments/ })).toBeInTheDocument();
    expect(screen.getByTestId("discussion-setup-back")).toHaveTextContent("Back");
    await user.click(screen.getByTestId("discussion-setup-back"));
    expect(screen.getByRole("heading", { name: "How to handle human comments?" })).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });

  it("shows an empty catalog and lets the user add a bot login manually", async () => {
    mocks.catalog.splice(0);
    const user = userEvent.setup();
    const onCreated = vi.fn();
    render(
      <DiscussionPRFeedbackSetupDialog
        organizationId="org-1"
        factoryId="factory-1"
        githubIntegrationId="gh-1"
        repository="acme/app"
        source={discussionSource}
        onClose={vi.fn()}
        onCreated={onCreated}
      />,
    );

    await chooseAddressBots(user);
    expect(screen.getByTestId("discussion-setup-bots-empty")).toBeInTheDocument();
    expect(screen.getByTestId("discussion-setup-preview-comment")).toHaveAttribute("data-author", "coderabbitai");
    expect(screen.getByTestId("discussion-setup-preview-bot-outcome")).toHaveTextContent("Ignored");
    expect(screen.getByTestId("discussion-setup-finish")).toBeEnabled();

    await user.click(screen.getByTestId("discussion-setup-bot-add-manually"));
    await user.type(screen.getByTestId("discussion-setup-bot-manual"), "coderabbitai");
    await user.click(screen.getByTestId("discussion-setup-bot-add"));
    expect(screen.getByTestId("discussion-setup-bot-coderabbitai")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByTestId("discussion-setup-preview-comment")).toHaveAttribute("data-author", "coderabbitai");
    expect(screen.getByTestId("discussion-setup-preview-bot-outcome")).toHaveTextContent("Fixing...");
    await user.click(screen.getByTestId("discussion-setup-finish"));

    expect(mocks.createHandler).toHaveBeenCalledWith(
      expect.objectContaining({
        settings: expect.objectContaining({
          discussion: expect.objectContaining({
            ignoreBots: true,
            allowedBots: ["coderabbitai"],
          }),
        }),
      }),
    );
    expect(onCreated).toHaveBeenCalledWith("handler-1");
  });
});
