import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { MarkdownCode } from "./MarkdownCode";

vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

vi.mock("@/contexts/useTheme", () => ({
  useTheme: () => ({ preference: "light", resolvedTheme: "light", setPreference: () => undefined }),
}));

describe("MarkdownCode", () => {
  it("does not render undefined for an empty fenced block", () => {
    render(<MarkdownCode className="language-go">{undefined}</MarkdownCode>);

    expect(screen.getByTestId("code-block-editor")).toBeInTheDocument();
    expect(screen.queryByText("undefined")).not.toBeInTheDocument();
    expect(screen.getByTestId("monaco-stub")).toHaveTextContent("");
  });
});
