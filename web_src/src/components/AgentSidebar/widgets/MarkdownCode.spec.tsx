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
  it("renders an empty fence as an empty code block, never the word undefined", () => {
    render(<MarkdownCode className="language-ts">{undefined}</MarkdownCode>);

    expect(screen.getByTestId("monaco-stub").textContent).toBe("");
    expect(screen.queryByText("undefined")).not.toBeInTheDocument();
  });
});
