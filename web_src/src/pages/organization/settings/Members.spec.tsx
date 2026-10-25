import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "bun:test";
import { client } from "@/api-client/client.gen";
import type { OrganizationsListInvitationsResponse } from "@/api-client";
import { TooltipProvider } from "@/ui/tooltip";
import { Members } from "./Members";

vi.mock("@/hooks/usePageTitle", () => ({ usePageTitle: vi.fn() }));
vi.mock("@/hooks/useReportPageReady", () => ({ useReportPageReady: vi.fn() }));
vi.mock("@/hooks/useMe", () => ({ useMe: () => ({ data: { id: "user-viewer" } }) }));
vi.mock("@/lib/toast", () => ({ showSuccessToast: vi.fn(), showErrorToast: vi.fn() }));

let manage = true;
let assignRoles = true;
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (_resource: string, action: string) => action === "read" || (action === "update" ? assignRoles : manage),
    isLoading: false,
  }),
}));

const LONG_EMAIL = "very.long.email.address.that.should.not.truncate@example.com";
const acceptanceUrl = "https://example.com/invitations/recipient-token";
const originalConfig = client.getConfig();
const writeText = vi.fn(async (_text: string) => {});
let seatLimit: number | undefined;
let listError = false;
let sendError = false;
let resendError = false;
let requests: Request[];
let invitationList: OrganizationsListInvitationsResponse;
let queryClient: QueryClient;

beforeEach(() => {
  manage = true;
  assignRoles = true;
  seatLimit = undefined;
  listError = false;
  sendError = false;
  resendError = false;
  requests = [];
  invitationList = { invitations: [], seats: { perSeat: false, occupied: "1", pendingInvitations: "0" } };
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  writeText.mockReset();
  writeText.mockResolvedValue(undefined);
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
  client.setConfig({ baseUrl: "http://localhost", fetch: apiFetch });
});

afterEach(() => {
  queryClient.clear();
  client.setConfig(originalConfig);
});

const apiFetch: typeof fetch = async (input, init) => {
  const request = input instanceof Request ? input : new Request(input, init);
  const path = new URL(request.url).pathname;
  if (path === "/api/v1/users") {
    return Response.json({
      users: [
        {
          metadata: { id: "user-1", email: LONG_EMAIL },
          spec: { displayName: "Ada Lovelace" },
          status: { isOwner: true, roles: [{ roleName: "org_admin", roleDisplayName: "Admin" }] },
        },
      ],
      seatLimit,
    });
  }
  if (path === "/api/v1/roles") {
    return Response.json({
      roles: [
        { metadata: { name: "org_admin" }, spec: { displayName: "Admin" } },
        { metadata: { name: "org_operator" }, spec: { displayName: "Operator" } },
      ],
    });
  }
  if (path === "/api/v1/organizations/org-1/invitations" && request.method === "GET") {
    return listError
      ? Response.json({ message: "Invitations unavailable" }, { status: 503 })
      : Response.json(invitationList);
  }
  requests.push(request.clone());
  if (path === "/api/v1/organizations/org-1/invitations" && request.method === "POST") {
    if (sendError)
      return Response.json({ message: "A pending invitation already exists for this email address." }, { status: 409 });
    const body = await request.json();
    const invitation = {
      id: "invitation-1",
      email: body.email,
      role: body.role,
      state: "pending",
      expiresAt: new Date(Date.now() + 86400000).toISOString(),
    };
    invitationList = { ...invitationList, invitations: [invitation] };
    return Response.json({ invitation, acceptanceUrl });
  }
  if (path === "/api/v1/organizations/org-1/invitations/invitation-1/resend") return invitationResendResponse();
  if (path === "/api/v1/organizations/org-1/invitations/invitation-1" && request.method === "DELETE") {
    invitationList = {
      ...invitationList,
      invitations: [{ ...invitationList.invitations?.[0], state: "canceled" }],
      seats: { ...invitationList.seats, invitable: "1", pendingInvitations: "0" },
    };
    return Response.json({});
  }
  return Response.json({ message: "Unexpected request: " + path }, { status: 404 });
};

function invitationResendResponse() {
  if (resendError) return Response.json({ message: "Could not resend invitation." }, { status: 500 });
  return Response.json({ invitation: invitationList.invitations?.[0], acceptanceUrl: acceptanceUrl + "-renewed" });
}

function renderMembers() {
  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <Members organizationId="org-1" />
      </TooltipProvider>
    </QueryClientProvider>,
  );
}

async function enterRecipient() {
  if (!screen.queryByLabelText("Email address")) {
    fireEvent.click(await screen.findByRole("button", { name: "Invite member" }));
  }
  const email = await screen.findByLabelText("Email address");
  fireEvent.change(email, { target: { value: "invitee@example.com" } });
  const sendInvitation = screen.getByRole("button", { name: "Send invitation" });
  await waitFor(() => expect(sendInvitation.matches(":disabled")).toBe(false));
}

it("creates an email invitation and copies its recipient-specific link", async () => {
  renderMembers();
  await enterRecipient();
  expect(screen.queryByRole("switch", { name: "Toggle invite link" })).not.toBeInTheDocument();
  expect(screen.queryByText("generate a new link")).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Send invitation" }));
  const link = await screen.findByLabelText("Invitation link for invitee@example.com");
  expect(link).toHaveValue(acceptanceUrl);
  const request = requests[0];
  expect(await request.json()).toEqual({ email: "invitee@example.com", role: "org_operator" });
  expect(request.headers.get("x-organization-id")).toBe("org-1");
  fireEvent.click(screen.getByRole("button", { name: "Copy link" }));
  await waitFor(() => expect(writeText).toHaveBeenCalledWith(acceptanceUrl));
  fireEvent.click(screen.getByRole("button", { name: "Done" }));
  fireEvent.keyDown(screen.getByRole("button", { name: "Invitation actions for invitee@example.com" }), {
    key: "ArrowDown",
  });
  fireEvent.click(await screen.findByRole("menuitem", { name: "Copy link" }));
  await waitFor(() => expect(writeText).toHaveBeenCalledTimes(2));
  expect(requests).toHaveLength(1);
});

it("resends and copies a saved invitation, then releases its seat on cancel", async () => {
  invitationList = {
    invitations: [{ id: "invitation-1", email: "invitee@example.com", role: "org_operator", state: "pending" }],
    seats: { perSeat: true, invitable: "0", pendingInvitations: "1" },
  };
  renderMembers();
  fireEvent.keyDown(await screen.findByRole("button", { name: "Invitation actions for invitee@example.com" }), {
    key: "ArrowDown",
  });
  fireEvent.click(await screen.findByRole("menuitem", { name: "Resend and copy link" }));
  await waitFor(() => expect(writeText).toHaveBeenCalledWith(acceptanceUrl + "-renewed"));
  fireEvent.keyDown(screen.getByRole("button", { name: "Invitation actions for invitee@example.com" }), {
    key: "ArrowDown",
  });
  fireEvent.click(await screen.findByRole("menuitem", { name: "Cancel invitation" }));
  await waitFor(() => expect(screen.queryByText("invitee@example.com")).not.toBeInTheDocument());
  expect(requests.map((request) => request.method)).toEqual(["POST", "DELETE"]);
  await enterRecipient();
});

for (const seats of [
  { perSeat: true, purchased: "2", occupied: "1", invitable: "0", pendingInvitations: "1" },
  { perSeat: false, memberLimit: "1", occupied: "1" },
]) {
  it(`blocks new invitations when ${seats.perSeat ? "pending invitations reserve the last paid seat" : "the member limit is reached"}`, async () => {
    invitationList.seats = seats;
    renderMembers();
    fireEvent.click(await screen.findByRole("button", { name: "Invite member" }));
    fireEvent.change(await screen.findByLabelText("Email address"), { target: { value: "another@example.com" } });
    expect(screen.getByRole("button", { name: "Send invitation" })).toBeDisabled();
    expect(requests).toHaveLength(0);
  });
}

it("keeps the form blocked after a list failure and recovers on refresh", async () => {
  listError = true;
  renderMembers();
  fireEvent.click(await screen.findByRole("button", { name: "Invite member" }));
  fireEvent.change(await screen.findByLabelText("Email address"), { target: { value: "invitee@example.com" } });
  expect(await screen.findByText("Invitations unavailable")).toBeInTheDocument();
  const sendInvitation = screen.getByRole("button", { name: "Send invitation" });
  expect(sendInvitation.matches(":disabled")).toBe(true);
  listError = false;
  await act(async () => {
    await queryClient.invalidateQueries({ queryKey: ["organization", "invitations", "org-1"] });
  });
  await waitFor(() => expect(sendInvitation.matches(":disabled")).toBe(false));
});

it("shows send failures without offering a link or losing the recipient", async () => {
  sendError = true;
  renderMembers();
  await enterRecipient();
  fireEvent.click(screen.getByRole("button", { name: "Send invitation" }));
  expect(await screen.findByText("A pending invitation already exists for this email address.")).toBeInTheDocument();
  expect(screen.getByLabelText("Email address")).toHaveValue("invitee@example.com");
  expect(screen.queryByRole("button", { name: "Copy link" })).not.toBeInTheDocument();
});

it("keeps the manual link available when the clipboard is unavailable", async () => {
  renderMembers();
  await enterRecipient();
  fireEvent.click(screen.getByRole("button", { name: "Send invitation" }));
  await screen.findByLabelText("Invitation link for invitee@example.com");
  writeText.mockRejectedValueOnce(new Error("Clipboard unavailable"));
  fireEvent.click(screen.getByRole("button", { name: "Copy link" }));
  expect(await screen.findByText("Could not copy the link. Select and copy it manually.")).toBeInTheDocument();
  expect(screen.getByLabelText("Invitation link for invitee@example.com")).toHaveValue(acceptanceUrl);
});

it("shows member seats without invitation controls for read-only members", async () => {
  manage = false;
  assignRoles = false;
  seatLimit = 4;
  renderMembers();
  expect(await screen.findByText("1 of 4 seats used")).toBeInTheDocument();
  expect(screen.queryByRole("button", { name: "Send invitation" })).not.toBeInTheDocument();
  expect(screen.queryByLabelText("Email address")).not.toBeInTheDocument();
});

it("allows inviters without role-update permission to invite only Operators", async () => {
  assignRoles = false;
  renderMembers();
  await enterRecipient();
  expect(screen.getByRole("combobox", { name: "Role" })).toBeDisabled();
  fireEvent.click(screen.getByRole("button", { name: "Send invitation" }));
  await screen.findByLabelText("Invitation link for invitee@example.com");
  expect(await requests[0].json()).toMatchObject({ role: "org_operator" });
});

it("keeps full member emails readable in the member row", async () => {
  renderMembers();
  const email = await screen.findByText(LONG_EMAIL);
  const row = email.closest("tr")!;
  expect(within(row).getByText("Ada Lovelace")).toBeInTheDocument();
});

it("sends the role selected by an admin", async () => {
  Element.prototype.scrollIntoView = vi.fn();
  renderMembers();
  await enterRecipient();
  fireEvent.keyDown(screen.getByRole("combobox", { name: "Role" }), { key: "ArrowDown" });
  fireEvent.click(await screen.findByRole("option", { name: "Admin" }));
  fireEvent.click(screen.getByRole("button", { name: "Send invitation" }));
  await screen.findByLabelText("Invitation link for invitee@example.com");
  expect(await requests[0].json()).toMatchObject({ role: "org_admin" });
});

it("blocks new invitations while cached capacity refreshes", async () => {
  renderMembers();
  await enterRecipient();
  const sendInvitation = screen.getByRole("button", { name: "Send invitation" });
  let finishRefresh!: () => void;
  const pendingResponse = new Promise<void>((resolve) => {
    finishRefresh = resolve;
  });
  client.setConfig({
    fetch: async (input, init) => {
      await pendingResponse;
      return apiFetch(input, init);
    },
  });
  const refresh = queryClient.invalidateQueries({ queryKey: ["organization", "invitations", "org-1"] });
  try {
    await waitFor(() => expect(sendInvitation.matches(":disabled")).toBe(true));
  } finally {
    finishRefresh();
    await refresh;
  }
  await waitFor(() => expect(sendInvitation.matches(":disabled")).toBe(false));
});

it.each([1, 14])("shows the configured invitation duration of %i days", async (days) => {
  invitationList.invitationDurationDays = days;
  renderMembers();
  fireEvent.click(await screen.findByRole("button", { name: "Invite member" }));
  expect(
    await screen.findByText(`Invitations expire after ${days} ${days === 1 ? "day" : "days"}.`),
  ).toBeInTheDocument();
});

it("shows members and actionable invitations in one list", async () => {
  invitationList.invitations = [
    { id: "pending", email: "pending@example.com", role: "org_operator", state: "pending" },
    { id: "expired", email: "expired@example.com", role: "org_operator", state: "expired" },
    { id: "accepted", email: "accepted@example.com", state: "accepted" },
    { id: "canceled", email: "canceled@example.com", state: "canceled" },
  ];
  renderMembers();
  const member = await screen.findByText("Ada Lovelace");
  const pending = await screen.findByText("pending@example.com");
  expect(member.closest("table")).toBe(pending.closest("table"));
  expect(screen.getAllByRole("table")).toHaveLength(1);
  expect(screen.getByText("expired@example.com")).toBeInTheDocument();
  expect(screen.queryByText("accepted@example.com")).not.toBeInTheDocument();
  expect(screen.queryByText("canceled@example.com")).not.toBeInTheDocument();
  expect(screen.queryByLabelText("Email address")).not.toBeInTheDocument();
});

it("offers manual copying when resending a saved invitation cannot use the clipboard", async () => {
  invitationList.invitations = [
    { id: "invitation-1", email: "invitee@example.com", role: "org_operator", state: "pending" },
  ];
  writeText.mockRejectedValueOnce(new Error("Clipboard unavailable"));
  renderMembers();
  fireEvent.keyDown(await screen.findByRole("button", { name: "Invitation actions for invitee@example.com" }), {
    key: "ArrowDown",
  });
  fireEvent.click(await screen.findByRole("menuitem", { name: "Resend and copy link" }));
  expect(await screen.findByLabelText("Invitation link for invitee@example.com")).toHaveValue(
    acceptanceUrl + "-renewed",
  );
  expect(screen.getByText("Could not copy the link. Select and copy it manually.")).toBeInTheDocument();
});

it("shows a resend failure without offering an invalid invitation link", async () => {
  resendError = true;
  invitationList.invitations = [
    { id: "invitation-1", email: "invitee@example.com", role: "org_operator", state: "pending" },
  ];
  renderMembers();
  fireEvent.keyDown(await screen.findByRole("button", { name: "Invitation actions for invitee@example.com" }), {
    key: "ArrowDown",
  });
  fireEvent.click(await screen.findByRole("menuitem", { name: "Resend and copy link" }));
  expect(await screen.findByText("Could not resend invitation.")).toBeInTheDocument();
  expect(screen.queryByLabelText("Invitation link for invitee@example.com")).not.toBeInTheDocument();
  expect(writeText).not.toHaveBeenCalled();
});
