import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import OwnerSetup from "./OwnerSetup";

vi.mock("@/posthog", () => ({
  isPostHogEnabled: false,
  posthog: { getActiveMatchingSurveys: vi.fn() },
}));

const mockFetch = vi.fn();
vi.stubGlobal("fetch", mockFetch);

const mockLocationAssign = vi.fn();
Object.defineProperty(window, "location", {
  value: { set href(url: string) { mockLocationAssign(url); } },
  writable: true,
});

type User = ReturnType<typeof userEvent.setup>;

async function fillOwnerForm(
  user: User,
  overrides: { organizationName?: string; email?: string } = {},
) {
  const orgInput = screen.getByPlaceholderText("Acme Inc.");
  await user.clear(orgInput);
  await user.type(orgInput, overrides.organizationName ?? "Acme Corp");
  await user.type(screen.getByPlaceholderText("First name"), "Jane");
  await user.type(screen.getByPlaceholderText("Last name"), "Doe");
  await user.type(screen.getByPlaceholderText("you@example.com"), overrides.email ?? "jane@example.com");
  await user.type(screen.getByPlaceholderText("Password"), "Password1");
  await user.type(screen.getByPlaceholderText("Confirm password"), "Password1");
}

async function advanceToSmtpPrompt(user: User) {
  await fillOwnerForm(user);
  await user.click(screen.getByRole("button", { name: "Next" }));
  await screen.findByText("Private network access");
  await user.click(screen.getByRole("button", { name: "Next" }));
  await screen.findByText("Set up email delivery?");
}

async function advanceToSmtpConfig(user: User) {
  await advanceToSmtpPrompt(user);
  await user.click(screen.getByRole("button", { name: "Set up SMTP" }));
  await screen.findByText("SMTP configuration");
}

describe("OwnerSetup", () => {
  beforeEach(() => {
    mockFetch.mockReset();
    mockLocationAssign.mockReset();
  });

  describe("owner step", () => {
    it("renders with Organization Name defaulting to Demo", () => {
      render(<OwnerSetup />);
      expect(screen.getByPlaceholderText("Acme Inc.")).toHaveValue("Demo");
    });

    it("shows validation errors when required fields are empty", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await user.click(screen.getByRole("button", { name: "Next" }));
      expect(await screen.findByText("Email is required.")).toBeInTheDocument();
      expect(screen.getByText("First name is required.")).toBeInTheDocument();
      expect(screen.getByText("Last name is required.")).toBeInTheDocument();
      expect(screen.getByText("Password is required.")).toBeInTheDocument();
    });

    it("shows invalid email error", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await user.type(screen.getByPlaceholderText("you@example.com"), "not-an-email");
      await user.click(screen.getByRole("button", { name: "Next" }));
      expect(await screen.findByText("Please enter a valid email address.")).toBeInTheDocument();
    });

    it("shows password strength error", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await user.type(screen.getByPlaceholderText("Password"), "weak");
      await user.click(screen.getByRole("button", { name: "Next" }));
      expect(
        await screen.findByText("Password must be 8+ characters with at least 1 number and 1 capital letter."),
      ).toBeInTheDocument();
    });

    it("shows error when passwords do not match", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await user.type(screen.getByPlaceholderText("Password"), "Password1");
      await user.type(screen.getByPlaceholderText("Confirm password"), "Different1");
      await user.click(screen.getByRole("button", { name: "Next" }));
      expect(await screen.findByText("Passwords do not match.")).toBeInTheDocument();
    });

    it("advances to private network step when form is valid", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await fillOwnerForm(user);
      await user.click(screen.getByRole("button", { name: "Next" }));
      expect(await screen.findByText("Private network access")).toBeInTheDocument();
    });
  });

  describe("private network step", () => {
    it("goes back to owner step", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await fillOwnerForm(user);
      await user.click(screen.getByRole("button", { name: "Next" }));
      await screen.findByText("Private network access");
      await user.click(screen.getByRole("button", { name: "Back" }));
      expect(await screen.findByText("Set up owner account")).toBeInTheDocument();
    });

    it("advances to SMTP prompt step", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await fillOwnerForm(user);
      await user.click(screen.getByRole("button", { name: "Next" }));
      await screen.findByText("Private network access");
      await user.click(screen.getByRole("button", { name: "Next" }));
      expect(await screen.findByText("Set up email delivery?")).toBeInTheDocument();
    });
  });

  describe("SMTP prompt step", () => {
    it("submits without SMTP when skipped and redirects", async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => ({ organization_id: "org-123" }) });
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpPrompt(user);
      await user.click(screen.getByRole("button", { name: "Do this later" }));
      await waitFor(() => expect(mockLocationAssign).toHaveBeenCalledWith("/org-123"));
    });

    it("advances to SMTP config when Set up SMTP is clicked", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpPrompt(user);
      await user.click(screen.getByRole("button", { name: "Set up SMTP" }));
      expect(await screen.findByText("SMTP configuration")).toBeInTheDocument();
    });

    it("goes back to private network step", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpPrompt(user);
      await user.click(screen.getByRole("button", { name: "Back" }));
      expect(await screen.findByText("Private network access")).toBeInTheDocument();
    });
  });

  describe("SMTP config step", () => {
    it("shows validation errors for missing required SMTP fields", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpConfig(user);
      await user.click(screen.getByRole("button", { name: "Finish setup" }));
      expect(await screen.findByText("SMTP host is required.")).toBeInTheDocument();
      expect(screen.getByText("SMTP port is required.")).toBeInTheDocument();
      expect(screen.getByText("SMTP from email is required.")).toBeInTheDocument();
    });

    it("shows error when SMTP port is not a number", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpConfig(user);
      await user.type(screen.getByPlaceholderText("smtp.example.com"), "smtp.example.com");
      await user.type(screen.getByPlaceholderText("587"), "abc");
      await user.type(screen.getByPlaceholderText("noreply@example.com"), "noreply@example.com");
      await user.click(screen.getByRole("button", { name: "Finish setup" }));
      expect(await screen.findByText("SMTP port must be a number.")).toBeInTheDocument();
    });

    it("shows error when username provided without password", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpConfig(user);
      await user.type(screen.getByPlaceholderText("smtp.example.com"), "smtp.example.com");
      await user.type(screen.getByPlaceholderText("587"), "587");
      await user.type(screen.getByPlaceholderText("smtp-user"), "user");
      await user.type(screen.getByPlaceholderText("noreply@example.com"), "noreply@example.com");
      await user.click(screen.getByRole("button", { name: "Finish setup" }));
      expect(
        await screen.findByText("SMTP password is required when username is provided."),
      ).toBeInTheDocument();
    });

    it("submits with SMTP config and redirects", async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => ({ organization_id: "org-456" }) });
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpConfig(user);
      await user.type(screen.getByPlaceholderText("smtp.example.com"), "smtp.example.com");
      await user.type(screen.getByPlaceholderText("587"), "587");
      await user.type(screen.getByPlaceholderText("noreply@example.com"), "noreply@example.com");
      await user.click(screen.getByRole("button", { name: "Finish setup" }));
      await waitFor(() => expect(mockLocationAssign).toHaveBeenCalledWith("/org-456"));
    });

    it("can skip SMTP from the config step", async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => ({ organization_id: "org-789" }) });
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpConfig(user);
      await user.click(screen.getByRole("button", { name: "Do this later" }));
      await waitFor(() => expect(mockLocationAssign).toHaveBeenCalledWith("/org-789"));
    });

    it("goes back to SMTP prompt", async () => {
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpConfig(user);
      await user.click(screen.getByRole("button", { name: "Back" }));
      expect(await screen.findByText("Set up email delivery?")).toBeInTheDocument();
    });
  });

  describe("API submission", () => {
    it("sends organization_name in the request body", async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => ({ organization_id: "org-123" }) });
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await fillOwnerForm(user, { organizationName: "My Company" });
      await user.click(screen.getByRole("button", { name: "Next" }));
      await screen.findByText("Private network access");
      await user.click(screen.getByRole("button", { name: "Next" }));
      await screen.findByText("Set up email delivery?");
      await user.click(screen.getByRole("button", { name: "Do this later" }));
      await waitFor(() => expect(mockFetch).toHaveBeenCalled());
      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body.organization_name).toBe("My Company");
    });

    it("sends Demo as organization_name when left at default", async () => {
      mockFetch.mockResolvedValue({ ok: true, json: async () => ({ organization_id: "org-123" }) });
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await user.type(screen.getByPlaceholderText("you@example.com"), "jane@example.com");
      await user.type(screen.getByPlaceholderText("First name"), "Jane");
      await user.type(screen.getByPlaceholderText("Last name"), "Doe");
      await user.type(screen.getByPlaceholderText("Password"), "Password1");
      await user.type(screen.getByPlaceholderText("Confirm password"), "Password1");
      await user.click(screen.getByRole("button", { name: "Next" }));
      await screen.findByText("Private network access");
      await user.click(screen.getByRole("button", { name: "Next" }));
      await screen.findByText("Set up email delivery?");
      await user.click(screen.getByRole("button", { name: "Do this later" }));
      await waitFor(() => expect(mockFetch).toHaveBeenCalled());
      const body = JSON.parse(mockFetch.mock.calls[0][1].body);
      expect(body.organization_name).toBe("Demo");
    });

    it("shows error banner when API returns an error message", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => ({ message: "Server exploded" }),
      });
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpPrompt(user);
      await user.click(screen.getByRole("button", { name: "Do this later" }));
      expect(await screen.findByText("Server exploded")).toBeInTheDocument();
    });

    it("shows conflict message when instance is already initialized", async () => {
      mockFetch.mockResolvedValue({
        ok: false,
        status: 409,
        json: async () => { throw new Error("no json"); },
      });
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpPrompt(user);
      await user.click(screen.getByRole("button", { name: "Do this later" }));
      expect(await screen.findByText("This instance is already initialized.")).toBeInTheDocument();
    });

    it("shows network error message on fetch failure", async () => {
      mockFetch.mockRejectedValue(new Error("Network down"));
      const user = userEvent.setup();
      render(<OwnerSetup />);
      await advanceToSmtpPrompt(user);
      await user.click(screen.getByRole("button", { name: "Do this later" }));
      expect(await screen.findByText("Network error occurred")).toBeInTheDocument();
    });
  });
});
