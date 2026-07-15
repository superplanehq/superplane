import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MetadataList } from "./index";

describe("MetadataList", () => {
  it("renders string labels", () => {
    render(<MetadataList items={[{ icon: "sparkles", label: "claude-opus-4-6" }]} />);
    expect(screen.getByText("claude-opus-4-6")).toBeInTheDocument();
  });

  it("coerces IntegrationResourceRef objects to their name instead of crashing", () => {
    render(
      <MetadataList
        items={[
          {
            icon: "sparkles",
            label: { id: "model-id", name: "claude-opus-4-6", type: "model" } as unknown as string,
          },
        ]}
      />,
    );

    expect(screen.getByText("claude-opus-4-6")).toBeInTheDocument();
  });
});
