import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { MetadataList, type MetadataItem } from "./";

describe("MetadataList", () => {
  it("renders string labels", () => {
    render(<MetadataList items={[{ icon: "sparkles", label: "claude-opus-4" }]} />);
    expect(screen.getByText("claude-opus-4")).toBeInTheDocument();
  });

  it("renders React node labels", () => {
    render(<MetadataList items={[{ icon: "braces", label: <span>Structured output</span> }]} />);
    expect(screen.getByText("Structured output")).toBeInTheDocument();
  });

  it("renders the resource name instead of crashing when a label is a raw resource object", () => {
    // Regression for #6072: an integration resource reference ({ id, name, type })
    // leaking through as a label used to throw "Objects are not valid as a React child"
    // and crash the whole canvas.
    const item = { icon: "sparkles", label: { id: "m_1", name: "claude-opus-4", type: "model" } } as MetadataItem;
    render(<MetadataList items={[item]} />);
    expect(screen.getByText("claude-opus-4")).toBeInTheDocument();
  });

  it("falls back to the resource id when the object has no name", () => {
    const item = { icon: "sparkles", label: { id: "m_1", type: "model" } } as MetadataItem;
    render(<MetadataList items={[item]} />);
    expect(screen.getByText("m_1")).toBeInTheDocument();
  });
});
