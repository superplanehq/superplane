import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import type { SuperplaneUsersUser } from "@/api-client";
import { WorkOrderCommentComposer } from "./WorkOrderCommentComposer";

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: vi.fn(() => ({ data: mockUsers, isLoading: false })),
}));

function buildUser(id: string, displayName: string): SuperplaneUsersUser {
  return {
    metadata: { id, email: `${id}@example.com` },
    spec: { displayName },
  } as SuperplaneUsersUser;
}

const ALICE_ID = "11111111-1111-1111-1111-111111111111";
const BOB_ID = "22222222-2222-2222-2222-222222222222";

const mockUsers: SuperplaneUsersUser[] = [buildUser(ALICE_ID, "Alice Anderson"), buildUser(BOB_ID, "Bob Brown")];

async function findMentionOption(name: string) {
  const options = await screen.findAllByTestId("work-order-mention-option");
  const match = options.find((option) => option.textContent?.includes(name));
  if (!match) throw new Error(`No mention option found for "${name}"`);
  return match;
}

function renderComposer(overrides: Partial<Parameters<typeof WorkOrderCommentComposer>[0]> = {}) {
  const onSubmit = vi.fn().mockResolvedValue(undefined);
  render(
    <WorkOrderCommentComposer organizationId="org-1" canComment isSubmitting={false} onSubmit={onSubmit} {...overrides} />,
  );
  return { onSubmit };
}

describe("WorkOrderCommentComposer", () => {
  it("submits the trimmed body with no mentions when nobody is tagged", async () => {
    const { onSubmit } = renderComposer();
    const textarea = screen.getByLabelText("Add a comment");

    await userEvent.type(textarea, "  Just a note  ");
    await userEvent.click(screen.getByTestId("work-order-comment-submit"));

    expect(onSubmit).toHaveBeenCalledWith("Just a note", []);
  });

  it("opens the mention menu on @ and filters as you type", async () => {
    renderComposer();
    const textarea = screen.getByLabelText("Add a comment");

    await userEvent.type(textarea, "hey @al");

    expect(await screen.findByTestId("work-order-mention-menu")).toBeInTheDocument();
    const options = screen.getAllByTestId("work-order-mention-option");
    expect(options).toHaveLength(1);
    expect(options[0]).toHaveTextContent("Alice Anderson");
  });

  it("inserts a mention token on click and includes the id on submit", async () => {
    const { onSubmit } = renderComposer();
    const textarea = screen.getByLabelText<HTMLTextAreaElement>("Add a comment");

    await userEvent.type(textarea, "hey @alice");
    await userEvent.click(await findMentionOption("Alice Anderson"));

    await waitFor(() => {
      expect(textarea.value).toBe(`hey @[Alice Anderson](user:${ALICE_ID}) `);
    });
    expect(screen.queryByTestId("work-order-mention-menu")).not.toBeInTheDocument();

    await userEvent.type(textarea, "please take a look");
    await userEvent.click(screen.getByTestId("work-order-comment-submit"));

    expect(onSubmit).toHaveBeenCalledWith(
      `hey @[Alice Anderson](user:${ALICE_ID}) please take a look`,
      [ALICE_ID],
    );
  });

  it("selects a highlighted option with the keyboard (arrow + Enter)", async () => {
    const { onSubmit } = renderComposer();
    const textarea = screen.getByLabelText<HTMLTextAreaElement>("Add a comment");

    await userEvent.type(textarea, "@");
    await screen.findByTestId("work-order-mention-menu");

    // Down arrow moves the highlight from Alice to Bob, Enter selects Bob.
    await userEvent.keyboard("{ArrowDown}{Enter}");

    await waitFor(() => {
      expect(textarea.value).toBe(`@[Bob Brown](user:${BOB_ID}) `);
    });

    await userEvent.click(screen.getByTestId("work-order-comment-submit"));
    expect(onSubmit).toHaveBeenCalledWith(`@[Bob Brown](user:${BOB_ID})`, [BOB_ID]);
  });

  it("dismisses the menu on Escape without inserting anything", async () => {
    renderComposer();
    const textarea = screen.getByLabelText<HTMLTextAreaElement>("Add a comment");

    await userEvent.type(textarea, "@al");
    await screen.findByTestId("work-order-mention-menu");

    await userEvent.keyboard("{Escape}");

    expect(screen.queryByTestId("work-order-mention-menu")).not.toBeInTheDocument();
    expect(textarea.value).toBe("@al");
  });

  it("drops a stale mention id once its token is edited away", async () => {
    const { onSubmit } = renderComposer();
    const textarea = screen.getByLabelText<HTMLTextAreaElement>("Add a comment");

    await userEvent.type(textarea, "hey @alice");
    await userEvent.click(await findMentionOption("Alice Anderson"));
    await waitFor(() => expect(textarea.value).toBe(`hey @[Alice Anderson](user:${ALICE_ID}) `));

    // Break the token's markup by deleting its closing paren + trailing space.
    await userEvent.type(textarea, "{Backspace}{Backspace}");

    await userEvent.click(screen.getByTestId("work-order-comment-submit"));

    expect(onSubmit).toHaveBeenCalledWith(expect.any(String), []);
  });

  it("does not allow submitting an empty comment", async () => {
    const { onSubmit } = renderComposer();

    expect(screen.getByTestId("work-order-comment-submit")).toBeDisabled();
    await userEvent.click(screen.getByTestId("work-order-comment-submit"));
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("disables the textarea and submit button without comment permission", () => {
    renderComposer({ canComment: false });

    expect(screen.getByLabelText("Add a comment")).toBeDisabled();
    expect(screen.getByTestId("work-order-comment-submit")).toBeDisabled();
  });
});
