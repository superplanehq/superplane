import { describe, expect, it, vi } from "bun:test";
import { render, screen } from "@testing-library/react";

vi.mock("monaco-editor", () => {
  throw new Error("Unit tests must not load Monaco");
});
vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <pre data-testid="editor">{value}</pre>,
}));

const { Editor } = await import("./MonacoEditor");

describe("MonacoEditor in test mode", () => {
  it("renders immediately with the supplied props without loading Monaco", () => {
    render(<Editor value="name: Canvas" language="yaml" />);

    expect(screen.getByTestId("editor")).toHaveTextContent("name: Canvas");
  });
});
