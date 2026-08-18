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
    await user.click(screen.getByTestId("work-order-comment-submit"));

    expect(onSubmit).toHaveBeenCalledWith("Hello @Alice Anderson", ["alice"]);
  });
});
