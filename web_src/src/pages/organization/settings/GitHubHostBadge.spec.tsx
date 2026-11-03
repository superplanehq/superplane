import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { GitHubHostBadge } from "./GitHubHostBadge";

describe("GitHubHostBadge", () => {
  it("hides github.com", () => {
    const { container } = render(
      <GitHubHostBadge
        integration={{ metadata: { integrationName: "github" }, spec: { configuration: { host: "" } } }}
      />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("shows a GitHub Enterprise host", () => {
    render(
      <GitHubHostBadge
        integration={{
          metadata: { integrationName: "github" },
          status: { properties: [{ name: "host", value: "acme.ghe.com" }] },
        }}
      />,
    );
    expect(screen.getByTestId("github-host-badge")).toHaveTextContent("acme.ghe.com");
  });
});
