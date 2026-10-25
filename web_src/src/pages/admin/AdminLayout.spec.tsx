import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes } from "react-router";

import AdminLayout from "./AdminLayout";

const { accountRef, loadingRef } = vi.hoisted(() => ({
  accountRef: { current: null as { name: string; installation_admin: boolean } | null },
  loadingRef: { current: false },
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: accountRef.current,
    loading: loadingRef.current,
    setupRequired: false,
    refreshAccount: async () => undefined,
  }),
}));

const pageLinks = [
  { label: "Organizations", href: "/admin", icon: "lucide-building" },
  { label: "Accounts", href: "/admin/accounts", icon: "lucide-users" },
  { label: "Settings", href: "/admin/settings", icon: "lucide-network" },
  { label: "Encryption", href: "/admin/encryption", icon: "lucide-key-round" },
  { label: "License", href: "/admin/license", icon: "lucide-key-round" },
  { label: "Billing plans", href: "/admin/billing-plans", icon: "lucide-book-open" },
  { label: "Credits", href: "/admin/credits", icon: "lucide-credit-card" },
  { label: "Features", href: "/admin/features", icon: "lucide-book-open" },
  { label: "Price Books", href: "/admin/price-books", icon: "lucide-book-open" },
  { label: "Fleets", href: "/admin/runner-tasks", icon: "lucide-terminal" },
  { label: "Fleet setup", href: "/admin/fleet-manager-setup", icon: "lucide-terminal" },
  { label: "Polar Webhooks", href: "/admin/polar-webhooks", icon: "lucide-webhook" },
  { label: "Webhooks", href: "/admin/webhooks", icon: "lucide-webhook" },
];

const renderLayout = (path: string) =>
  render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/" element={<p>App home</p>} />
        <Route path="/admin" element={<AdminLayout />}>
          <Route index element={<h1>Organizations page</h1>} />
          <Route path="organizations/:orgId" element={<h1>Organization detail</h1>} />
          <Route path="features" element={<h1>Features page</h1>} />
          <Route path="webhooks" element={<h1>Webhooks page</h1>} />
          <Route path="*" element={null} />
        </Route>
      </Routes>
    </MemoryRouter>,
  );

beforeEach(() => {
  accountRef.current = { name: "Ada Lovelace", installation_admin: true };
  loadingRef.current = false;
});

describe("AdminLayout", () => {
  it("navigates to Features from the sidebar", async () => {
    renderLayout("/admin");
    await userEvent.setup().click(screen.getByRole("link", { name: "Features" }));
    expect(screen.getByRole("heading", { name: "Features page" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Features" })).toHaveAttribute("aria-current", "page");
  });
  it("places page links in the left sidebar and keeps header chrome", () => {
    renderLayout("/admin");

    const header = screen.getByRole("banner");
    const nav = screen.getByRole("navigation");
    const main = screen.getByRole("main");
    const links = within(nav).getAllByRole("link");

    expect(header.contains(nav)).toBe(false);
    expect(nav.parentElement?.firstElementChild).toBe(nav);
    expect(nav.parentElement?.lastElementChild).toBe(main);
    expect(header.parentElement).toBe(nav.parentElement?.parentElement);
    expect(header.parentElement).toHaveClass("h-full", "min-h-0");
    expect(header.parentElement).not.toHaveClass("h-screen");
    expect(nav).toHaveClass("w-60", "bg-white", "border-r");
    expect(main).toHaveClass("overflow-auto");

    expect(within(header).getByRole("link", { name: "SuperPlane" })).toHaveAttribute("href", "/");
    expect(within(header).getByText("Installation Admin")).toBeInTheDocument();
    expect(within(header).getByText("Ada Lovelace")).toBeInTheDocument();
    expect(within(header).getByRole("link", { name: "Back to app" })).toHaveAttribute("href", "/");
    expect(within(header).queryByRole("link", { name: "Organizations" })).not.toBeInTheDocument();

    expect(links.map((link) => link.textContent)).toEqual(pageLinks.map((link) => link.label));
    expect(links.map((link) => link.getAttribute("href"))).toEqual(pageLinks.map((link) => link.href));
    pageLinks.forEach((pageLink, index) => {
      expect(links[index].querySelector("svg")).toHaveClass(pageLink.icon);
    });
    expect(within(nav).queryByRole("link", { name: "Sentry" })).not.toBeInTheDocument();
    expect(within(nav).queryByRole("tab", { name: "Sentry" })).not.toBeInTheDocument();

    expect(within(main).getByRole("heading", { name: "Organizations page" })).toBeInTheDocument();
    expect(links[0]).toHaveAttribute("aria-current", "page");
    expect(links[0]).toHaveClass("bg-sky-100");
  });

  it("does not mark Organizations active on an organization detail page", () => {
    renderLayout("/admin/organizations/org-1");

    expect(screen.getByRole("link", { name: "Organizations" })).not.toHaveAttribute("aria-current");
    expect(screen.getByRole("heading", { name: "Organization detail" })).toBeInTheDocument();
  });

  it("highlights Webhooks without highlighting Polar Webhooks", () => {
    renderLayout("/admin/webhooks");

    expect(screen.getByRole("link", { name: "Webhooks" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Polar Webhooks" })).not.toHaveAttribute("aria-current");
    expect(screen.getByRole("heading", { name: "Webhooks page" })).toBeInTheDocument();
  });

  it("redirects a non-admin away from Installation Admin", () => {
    accountRef.current = { name: "Ada Lovelace", installation_admin: false };

    renderLayout("/admin/license");

    expect(screen.getByText("App home")).toBeInTheDocument();
    expect(screen.queryByText("Installation Admin")).not.toBeInTheDocument();
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
  });

  it("shows the loading state before the account is known", () => {
    loadingRef.current = true;
    accountRef.current = null;

    renderLayout("/admin");

    expect(screen.getByText("Loading...")).toBeInTheDocument();
    expect(screen.queryByText("App home")).not.toBeInTheDocument();
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
  });
});
