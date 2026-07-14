import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { MetadataItem } from "./";
import { MetadataList } from "./";

describe("MetadataList", () => {
  it("renders string labels", () => {
    render(<MetadataList items={[{ icon: "book", label: "monarch-app" }]} />);
    expect(screen.getByText("monarch-app")).toBeInTheDocument();
  });

  it("renders numeric labels", () => {
    render(<MetadataList items={[{ icon: "hash", label: 42 as unknown as string }]} />);
    expect(screen.getByText("42")).toBeInTheDocument();
  });

  it("renders React element labels", () => {
    render(<MetadataList items={[{ icon: "clock", label: <span>2 hours ago</span> }]} />);
    expect(screen.getByText("2 hours ago")).toBeInTheDocument();
  });

  it("renders the name of a resource-reference object instead of crashing", () => {
    // Mappers occasionally build a label from a raw IntegrationResourceRef
    // ({ id, name, type }) rather than a string. Rendering the object directly
    // throws "Objects are not valid as a React child" and can crash the canvas.
    const resourceRef = { id: "res_1", name: "claude-opus-4", type: "model" };

    expect(() =>
      render(<MetadataList items={[{ icon: "sparkles", label: resourceRef as unknown as string }]} />),
    ).not.toThrow();
    expect(screen.getByText("claude-opus-4")).toBeInTheDocument();
  });

  it("falls back to the id when a resource object has no name", () => {
    const resourceRef = { id: "res_9", type: "model" };

    render(<MetadataList items={[{ icon: "sparkles", label: resourceRef as unknown as string }]} />);
    expect(screen.getByText("res_9")).toBeInTheDocument();
  });

  it("renders nothing for an object without a displayable field", () => {
    const items: MetadataItem[] = [{ icon: "sparkles", label: { foo: "bar" } as unknown as string }];

    expect(() => render(<MetadataList items={items} />)).not.toThrow();
  });
});
