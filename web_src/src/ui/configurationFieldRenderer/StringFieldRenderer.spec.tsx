import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { ConfigurationField } from "@/api-client";

import { ExpressionFieldRenderer } from "./ExpressionFieldRenderer";
import { StringFieldRenderer } from "./StringFieldRenderer";

const field = { name: "value", type: "string", label: "Value" } as ConfigurationField;

describe("configuration field string normalization", () => {
  it("normalizes non-string values for string fields with expressions", () => {
    render(<StringFieldRenderer field={field} value={42} onChange={vi.fn()} allowExpressions />);

    expect(screen.getByRole("textbox")).toHaveValue("42");
  });

  it("normalizes non-string values for expression fields", () => {
    render(<ExpressionFieldRenderer field={field} value={42} onChange={vi.fn()} allowExpressions />);

    expect(screen.getByRole("textbox")).toHaveValue("42");
  });
});
