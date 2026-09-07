import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { IntegrationInstructions } from "./IntegrationInstructions";

describe("IntegrationInstructions", () => {
  it("uses settings color tokens when requested", () => {
    const { container } = render(<IntegrationInstructions description="Configure the integration." tone="settings" />);

    expect(container.firstElementChild).toHaveClass("border-border", "bg-card", "text-card-foreground");
  });
});
