import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { SuperplaneUsersUser } from "@/api-client";
import { MentionChipFromLink } from "./MentionChip";

const { useOrganizationUsersMock } = vi.hoisted(() => ({ useOrganizationUsersMock: vi.fn() }));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: useOrganizationUsersMock,
}));

function buildUser(id: string, displayName: string, avatarUrl?: string): SuperplaneUsersUser {
  return {
    metadata: { id, email: `${id}@example.com` },
    spec: { displayName },
    status: avatarUrl ? { accountProviders: [{ avatarUrl }] } : undefined,
  } as SuperplaneUsersUser;
}

describe("MentionChipFromLink", () => {
  it("renders the live org member's name and avatar when found", () => {
    useOrganizationUsersMock.mockReturnValue({
      data: [buildUser("user-1", "Ada Lovelace", "https://example.com/ada.png")],
    });

    render(<MentionChipFromLink userId="user-1" rawLabel="Stale Name" organizationId="org-1" />);

    const chip = screen.getByTestId("mention-chip");
    expect(chip).toHaveTextContent("@Ada Lovelace");
    expect(chip.querySelector("img")).toHaveAttribute("src", "https://example.com/ada.png");
  });

  it("falls back to the label captured in the link when the member isn't found", () => {
    useOrganizationUsersMock.mockReturnValue({ data: [] });

    render(<MentionChipFromLink userId="user-404" rawLabel="Departed Member" organizationId="org-1" />);

    expect(screen.getByTestId("mention-chip")).toHaveTextContent("@Departed Member");
  });

  it("falls back to a generic label when there's no captured name and no match", () => {
    useOrganizationUsersMock.mockReturnValue({ data: [] });

    render(<MentionChipFromLink userId="user-404" organizationId="org-1" />);

    expect(screen.getByTestId("mention-chip")).toHaveTextContent("@Unknown member");
  });
});
