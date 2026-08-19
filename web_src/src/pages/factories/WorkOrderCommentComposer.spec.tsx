import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { SuperplaneUsersUser } from "@/api-client";
import { WorkOrderCommentComposer } from "./WorkOrderCommentComposer";

function buildUser(id: string, displayName: string, email: string): SuperplaneUsersUser {
  return {
    metadata: { id, email },
    spec: { displayName },
  } as SuperplaneUsersUser;
}

const members: SuperplaneUsersUser[] = [
  buildUser("alice", "Alice Anderson", "alice@example.com"),
  buildUser("bob", "Bob Brown", "bob@example.com"),
];

function renderComposer(
  onSubmit: (body: string, mentionedUserIds: string[]) => Promise<void> = vi.fn(async () => undefined),
) {
  render(
    <WorkOrderCommentComposer
      organizationId="org-1"
      canComment
      isSubmitting={false}
      members={members}
      onSubmit={onSubmit}
    />,
  );
  return { onSubmit };
}

describe("WorkOrderCommentComposer", () => {
  it("shows matching members after typing @", async () => {
    const user = userEvent.setup();
    renderComposer();

    await user.type(screen.getByLabelText("Add a comment"), "@Ali");

    expect(screen.getByTestId("work-order-mention-menu")).toBeInTheDocument();
    expect(screen.getByTestId("work-order-mention-option-alice")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-mention-option-bob")).not.toBeInTheDocument();
  });

  it("inserts the selected mention and submits its user id", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn(async () => undefined);
    renderComposer(onSubmit);

    const textarea = screen.getByLabelText("Add a comment");
    await user.type(textarea, "Hello @Ali");
    await user.keyboard("{Enter}");
    expect(screen.getByTestId("work-order-mention")).toHaveTextContent("@Alice Anderson");
    await user.click(screen.getByTestId("work-order-comment-submit"));

    expect(onSubmit).toHaveBeenCalledWith("Hello @Alice Anderson", ["alice"]);
  });

  it("submits a typed complete @Name even when the picker is not used", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn(async () => undefined);
    renderComposer(onSubmit);

    const textarea = screen.getByLabelText("Add a comment");
    await user.type(textarea, "Thanks @Alice Anderson ");
    await user.click(screen.getByTestId("work-order-comment-submit"));

    expect(onSubmit).toHaveBeenCalledWith("Thanks @Alice Anderson", ["alice"]);
  });

  it("shows the mentioned member in a hover card", async () => {
    const user = userEvent.setup();
    renderComposer();

    const textarea = screen.getByLabelText("Add a comment");
    await user.type(textarea, "@Ali");
    await user.keyboard("{Enter}");
    await user.hover(screen.getByTestId("work-order-mention"));

    const card = await screen.findByTestId("work-order-mention-tooltip");
    expect(card).toHaveTextContent("Alice Anderson");
    expect(card).toHaveTextContent("alice@example.com");
  });

  it("submits with the send shortcut while the mention menu is open", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn(async () => undefined);
    renderComposer(onSubmit);

    const textarea = screen.getByLabelText("Add a comment");
    await user.type(textarea, "@Ali");
    expect(screen.getByTestId("work-order-mention-menu")).toBeInTheDocument();

    await user.keyboard("{Meta>}{Enter}{/Meta}");

    expect(onSubmit).toHaveBeenCalledWith("@Ali", []);
  });

  it("keeps the mention menu closed after Escape when the caret moves", async () => {
    const user = userEvent.setup();
    renderComposer();

    await user.type(screen.getByLabelText("Add a comment"), "@Ali");
    expect(screen.getByTestId("work-order-mention-menu")).toBeInTheDocument();

    await user.keyboard("{Escape}");
    expect(screen.queryByTestId("work-order-mention-menu")).not.toBeInTheDocument();

    await user.keyboard("{ArrowLeft}");
    expect(screen.queryByTestId("work-order-mention-menu")).not.toBeInTheDocument();
  });

  it("stacks the submit button above the mention overlay so a scrolled mention can't swallow the click", async () => {
    const user = userEvent.setup();
    const onSubmit = vi.fn(async () => undefined);
    renderComposer(onSubmit);

    // The mention overlay renders `@mention` pills with pointer-events
    // re-enabled (so hover cards work while composing) and it scrolls with
    // the textarea, so a pill can end up sitting right under the send
    // button. The button's wrapper must have a higher z-index than the
    // overlay or that pill would intercept clicks meant for the button.
    const submitButton = screen.getByTestId("work-order-comment-submit");
    const overlay = screen.getByTestId("work-order-comment-mention-overlay");
    expect(submitButton.closest(".z-\\[3\\]")).toBeInTheDocument();
    expect(overlay).toHaveClass("z-[2]");

    await user.type(screen.getByLabelText("Add a comment"), "cc @Ali");
    await user.keyboard("{Enter}");
    await user.click(submitButton);

    expect(onSubmit).toHaveBeenCalledWith("cc @Alice Anderson", ["alice"]);
  });
});
