import { beforeEach, describe, expect, it } from "bun:test";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";

import { OrganizationSsoLogin } from "./OrganizationSsoLogin";

describe("OrganizationSsoLogin", () => {
  beforeEach(() => {
    Object.defineProperty(window, "location", {
      configurable: true,
      value: {
        replace: (url: string) => {
          window.location.href = url;
        },
        href: "",
      },
    });
  });

  it("sends the browser to the organization start URL and keeps redirect", async () => {
    render(
      <MemoryRouter initialEntries={["/login/sso/acme?redirect=/acme/settings"]}>
        <Routes>
          <Route path="/login/sso/:slug" element={<OrganizationSsoLogin />} />
        </Routes>
      </MemoryRouter>,
    );
    expect(await screen.findByText("Continue to your identity provider.")).toBeInTheDocument();
    expect(window.location.href).toBe("/auth/sso/organizations/acme/start?redirect=%2Facme%2Fsettings");
  });
});
