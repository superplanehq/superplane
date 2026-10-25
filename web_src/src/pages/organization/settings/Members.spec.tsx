import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "bun:test";
import { client } from "@/api-client/client.gen";
import type { OrganizationsListInvitationsResponse } from "@/api-client";
import { formatDate } from "@/lib/datetime";
import { TooltipProvider } from "@/ui/tooltip";
import { MemoryRouter } from "react-router";
import { Members } from "./Members";

vi.mock("@/hooks/usePageTitle", () => ({ usePageTitle: vi.fn() }));
vi.mock("@/hooks/useReportPageReady", () => ({ useReportPageReady: vi.fn() }));
vi.mock("@/hooks/useMe", () => ({ useMe: () => ({ data: { id: "user-viewer" } }) }));
vi.mock("@/lib/toast", () => ({ showSuccessToast: vi.fn(), showErrorToast: vi.fn() }));

let manage = true;
let assignRoles = true;
let manageBilling = true;
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (resource: string, action: string) => {
      if (resource === "org" && action === "update") return manageBilling;
      return action === "read" || (action === "update" ? assignRoles : manage);
    },
    isLoading: false,
  }),
}));

const LONG_EMAIL = "very.long.email.address.that.should.not.truncate@example.com";
const acceptanceUrl = "https://example.com/invitations/recipient-token";
const originalConfig = client.getConfig();
const writeText = vi.fn(async (_text: string) => {});
interface ApiFailure {
  status: number;
  message: string;
  reason?: string;
}

let seatLimit: number | undefined;
let listError = false;
let sendFailure: ApiFailure | undefined;
let resendFailure: ApiFailure | undefined;
let seatsAfterFailedSend: OrganizationsListInvitationsResponse["seats"];
let seatsAfterCancel: OrganizationsListInvitationsResponse["seats"];
let requests: Request[];
let invitationList: OrganizationsListInvitationsResponse;
let queryClient: QueryClient;

beforeEach(() => {
  manage = true;
  assignRoles = true;
  manageBilling = true;
  seatLimit = undefined;
  listError = false;
  sendFailure = undefined;
  resendFailure = undefined;
  seatsAfterFailedSend = undefined;
  seatsAfterCancel = undefined;
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
    return createInvitationResponse(request);
  }
  if (path === "/api/v1/organizations/org-1/invitations/invitation-1/resend") return invitationResendResponse();
  if (path === "/api/v1/organizations/org-1/invitations/invitation-1" && request.method === "DELETE") {
    invitationList = {
      ...invitationList,
      invitations: [{ ...invitationList.invitations?.[0], state: "canceled" }],
      seats: seatsAfterCancel ?? { ...invitationList.seats, invitable: "1", pendingInvitations: "0" },
    };
    return Response.json({});
  }
  return Response.json({ message: "Unexpected request: " + path }, { status: 404 });
};

async function createInvitationResponse(request: Request) {
  if (sendFailure) {
    if (seatsAfterFailedSend) invitationList = { ...invitationList, seats: seatsAfterFailedSend };
    return failureResponse(sendFailure);
  }
  const body = await request.json();
  const invitation = {
    id: "invitation-1",
    email: body.email,
    role: body.role,
    state: "pending",
    expiresAt: new Date(Date.now() + 86400000).toISOString(),
  };
  const seats = invitationList.seats;
  const reserved = seats?.perSeat ? { ...seats, pendingInvitations: "1", invitable: "0" } : seats;
  invitationList = { ...invitationList, invitations: [invitation], seats: reserved };
  return Response.json({ invitation, acceptanceUrl });
}

function failureResponse({ status, message, reason }: ApiFailure) {
  const details = reason
    ? [{ "@type": "type.googleapis.com/google.rpc.ErrorInfo", reason, domain: "superplane.com" }]
    : undefined;
  return Response.json({ message, details }, { status });
}

function invitationResendResponse() {
  if (resendFailure) return failureResponse(resendFailure);
  return Response.json({ invitation: invitationList.invitations?.[0], acceptanceUrl: acceptanceUrl + "-renewed" });
}

function renderMembers() {
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <TooltipProvider>
          <Members organizationId="org-1" />
        </TooltipProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

async function enterRecipient() {
  if (!screen.queryByLabelText("Email address")) {
    const invite = await screen.findByRole("button", { name: "Invite member" });
    await waitFor(() => expect(invite).toBeEnabled());
    fireEvent.click(invite);
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

it("keeps the invite button disabled and the form closed after a list failure, then recovers on refresh", async () => {
  listError = true;
  renderMembers();
  expect(await screen.findByText("Invitations unavailable")).toBeInTheDocument();
  const invite = screen.getByRole("button", { name: "Invite member" });
  expect(invite).toBeDisabled();
  listError = false;
  await act(async () => {
    await queryClient.invalidateQueries({ queryKey: ["organization", "invitations", "org-1"] });
  });
  await waitFor(() => expect(invite).toBeEnabled());
});

it("shows send failures without offering a link or losing the recipient", async () => {
  sendFailure = { status: 409, message: "A pending invitation already exists for this email address." };
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
  invitationList.seats = { perSeat: false, memberLimit: "4", occupied: "1" };
  renderMembers();
  expect(await screen.findByText("1 of 4 members")).toBeInTheDocument();
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
  resendFailure = { status: 500, message: "Could not resend invitation." };
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

function perSeatCapacity(overrides: NonNullable<OrganizationsListInvitationsResponse["seats"]>) {
  return { perSeat: true, purchased: "10", occupied: "9", pendingInvitations: "0", invitable: "1", ...overrides };
}

const fullSeats = (purchasable: string) => perSeatCapacity({ occupied: "10", invitable: "0", purchasable });
const capacityNotice = () => screen.queryByRole("status", { name: "Invitation capacity" });
const purchaseLink = () => screen.queryByRole("link", { name: "Purchase seats" });

it("uses the last purchased seat for one invitation, then reports no invitable seats", async () => {
  invitationList.seats = perSeatCapacity({ purchasable: "5" });
  renderMembers();
  expect(await screen.findByText("9 of 10 seats used · 0 pending · 1 invitable")).toBeInTheDocument();
  expect(capacityNotice()).not.toBeInTheDocument();
  await enterRecipient();
  fireEvent.click(screen.getByRole("button", { name: "Send invitation" }));
  await screen.findByLabelText("Invitation link for invitee@example.com");
  fireEvent.click(screen.getByRole("button", { name: "Done" }));
  expect(await screen.findByText("9 of 10 seats used · 1 pending · 0 invitable")).toBeInTheDocument();
  await waitFor(() => expect(screen.getByRole("button", { name: "Invite member" })).toBeDisabled());
});

it.each([
  {
    name: "payment is blocked",
    seats: perSeatCapacity({ paymentBlocked: true, purchasable: "5" }),
    notice: "Invitations are paused until the payment method is updated.",
    purchase: false,
  },
  {
    name: "members exceed paid seats",
    seats: perSeatCapacity({ occupied: "12", invitable: "0", purchasable: "5" }),
    notice: "12 members use 10 paid seats.",
    purchase: true,
  },
  {
    name: "the plan maximum is reached",
    seats: fullSeats("0"),
    notice: "Remove a member or cancel a pending invitation to free a seat.",
    purchase: false,
  },
  {
    name: "the member limit is reached",
    seats: { perSeat: false, memberLimit: "1", occupied: "1" },
    notice: "Remove a member or contact a site admin to increase the limit.",
    purchase: false,
  },
])("disables inviting and explains recovery when $name", async ({ seats, notice, purchase }) => {
  invitationList.seats = seats;
  renderMembers();
  expect(await screen.findByRole("button", { name: "Invite member" })).toBeDisabled();
  expect(await screen.findByRole("status", { name: "Invitation capacity" })).toHaveTextContent(notice);
  expect(purchaseLink() !== null).toBe(purchase);
});

it("blocks resending a pending invitation while payment is blocked", async () => {
  invitationList.seats = perSeatCapacity({ paymentBlocked: true, invitable: "0", pendingInvitations: "1" });
  invitationList.invitations = [
    { id: "invitation-1", email: "invitee@example.com", role: "org_operator", state: "pending" },
  ];
  renderMembers();
  fireEvent.keyDown(await screen.findByRole("button", { name: "Invitation actions for invitee@example.com" }), {
    key: "ArrowDown",
  });
  expect(await screen.findByRole("menuitem", { name: "Resend invitation" })).toHaveAttribute("aria-disabled", "true");
  expect(screen.getByRole("menuitem", { name: "Resend and copy link" })).toHaveAttribute("aria-disabled", "true");
});

const billingPath = "/org-1/organization/billing";
it.each([
  { link: "Open Billing", billing: true, href: billingPath, seats: perSeatCapacity({ paymentBlocked: true }) },
  { link: "Open Billing", billing: false, href: null, seats: perSeatCapacity({ paymentBlocked: true }) },
  { link: "Purchase seats", billing: true, href: `${billingPath}?buySeats=1`, seats: fullSeats("5") },
  { link: "Purchase seats", billing: false, href: null, seats: fullSeats("5") },
  { link: "Purchase seats", billing: true, href: null, seats: fullSeats("0") },
])("offers the $link link only when allowed: billing=$billing href=$href", async ({ link, billing, href, seats }) => {
  manageBilling = billing;
  invitationList.seats = seats;
  renderMembers();
  await screen.findByRole("status", { name: "Invitation capacity" });
  expect(screen.queryByRole("link", { name: link })?.getAttribute("href") ?? null).toBe(href);
});

it("shows no capacity notice and keeps inviting open when billing is off", async () => {
  invitationList.seats = { perSeat: false, occupied: "1" };
  renderMembers();
  expect(await screen.findByText("1 member")).toBeInTheDocument();
  await waitFor(() => expect(screen.getByRole("button", { name: "Invite member" })).toBeEnabled());
  expect(capacityNotice()).not.toBeInTheDocument();
});

it.each([
  { reason: "INVITATION_RATE_LIMITED", expected: "Too many invitations were sent. Wait, then try again." },
  {
    reason: "PENDING_INVITATION_LIMIT_REACHED",
    expected: "Too many invitations are pending. Cancel a pending invitation, then try again.",
  },
])("shows a specific message for $reason", async ({ reason, expected }) => {
  sendFailure = { status: 429, message: "Server wording that the page does not repeat.", reason };
  renderMembers();
  await enterRecipient();
  fireEvent.click(screen.getByRole("button", { name: "Send invitation" }));
  expect(await screen.findByText(expected)).toBeInTheDocument();
  expect(screen.queryByText("Server wording that the page does not repeat.")).not.toBeInTheDocument();
});

it("shows the returned reason and refreshes capacity when a send loses a seat race", async () => {
  invitationList.seats = perSeatCapacity({ purchasable: "5" });
  seatsAfterFailedSend = perSeatCapacity({ invitable: "0", pendingInvitations: "1", purchasable: "5" });
  sendFailure = {
    status: 412,
    message: "Purchase a seat before sending another invitation.",
    reason: "SEAT_PURCHASE_REQUIRED",
  };
  renderMembers();
  await enterRecipient();
  fireEvent.click(screen.getByRole("button", { name: "Send invitation" }));
  expect(await screen.findByText("Purchase a seat before sending another invitation.")).toBeInTheDocument();
  await waitFor(() => expect(purchaseLink()).toBeInTheDocument());
  expect(screen.getByRole("button", { name: "Send invitation" })).toBeDisabled();
  expect(screen.getAllByRole("status", { name: "Invitation capacity" })).toHaveLength(1);
});

it("keeps one capacity notice after resend and cancel", async () => {
  invitationList.seats = perSeatCapacity({ occupied: "9", invitable: "0", pendingInvitations: "1", purchasable: "0" });
  invitationList.invitations = [
    { id: "invitation-1", email: "invitee@example.com", role: "org_operator", state: "pending" },
  ];
  seatsAfterCancel = invitationList.seats;
  renderMembers();
  const actions = () => screen.findByRole("button", { name: "Invitation actions for invitee@example.com" });
  const openActions = async () => fireEvent.keyDown(await actions(), { key: "ArrowDown" });
  await openActions();
  fireEvent.click(await screen.findByRole("menuitem", { name: "Resend invitation" }));
  await waitFor(() => expect(requests.map((request) => request.method)).toEqual(["POST"]));
  expect(screen.getAllByRole("status", { name: "Invitation capacity" })).toHaveLength(1);
  await openActions();
  fireEvent.click(await screen.findByRole("menuitem", { name: "Cancel invitation" }));
  await waitFor(() => expect(screen.queryByText("invitee@example.com")).not.toBeInTheDocument());
  expect(screen.getAllByRole("status", { name: "Invitation capacity" })).toHaveLength(1);
});

it("shows recipient, role, inviter, sent date, expiry, and status for each invitation", async () => {
  const [sentAt, expiresAt] = ["2026-03-03T12:00:00Z", "2026-03-10T12:00:00Z"];
  invitationList.invitations = [
    {
      id: "pending",
      email: "pending@example.com",
      role: "org_operator",
      inviterName: "Grace Hopper",
      state: "pending",
      sentAt,
      expiresAt,
    },
    { id: "expired", email: "expired@example.com", role: "org_admin", state: "expired", sentAt, expiresAt },
  ];
  renderMembers();
  const pending = within((await screen.findByText("pending@example.com")).closest("tr")!);
  expect(pending.getByText("Operator")).toBeInTheDocument();
  expect(pending.getByText("Invited by Grace Hopper")).toBeInTheDocument();
  expect(pending.getByText(`Sent ${formatDate(sentAt)}`)).toBeInTheDocument();
  expect(pending.getByText(`Expires ${formatDate(expiresAt)}`)).toBeInTheDocument();
  expect(pending.getAllByText("Pending").length).toBeGreaterThan(0);
  const expired = within(screen.getByText("expired@example.com").closest("tr")!);
  expect(expired.getByText("Admin")).toBeInTheDocument();
  expect(expired.getByText("Invited by Unknown")).toBeInTheDocument();
  expect(expired.getByText(`Expired ${formatDate(expiresAt)}`)).toBeInTheDocument();
  expect(expired.getAllByText("Expired").length).toBeGreaterThan(0);
});
