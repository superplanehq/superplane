import { render, screen } from "@testing-library/react";
import React from "react";
import { describe, expect, it, vi } from "vitest";
import { SpecsTooltip } from "./SpecsTooltip";
import type { ComponentBaseSpecValue } from "./index";

vi.mock("@tippyjs/react/headless", () => ({
  default: ({
    children,
    render: renderTooltip,
  }: {
    children: React.ReactNode;
    render: (attrs: Record<string, unknown>) => React.ReactNode;
  }) => (
    <div>
      {renderTooltip({})}
      {children}
    </div>
  ),
}));

describe("SpecsTooltip", () => {
  it("skips malformed badge labels without crashing", () => {
    const specValues = [
      {
        badges: [
          {
            label: undefined,
            bgColor: "bg-gray-100",
            textColor: "text-gray-800",
          },
          {
            label: "workflow_input",
            bgColor: "bg-purple-100",
            textColor: "text-purple-800",
          },
        ],
      },
    ] as unknown as ComponentBaseSpecValue[];

    expect(() =>
      render(
        <SpecsTooltip specTitle="input" specValues={specValues}>
          <span>trigger</span>
        </SpecsTooltip>,
      ),
    ).not.toThrow();

    expect(screen.getByText("workflow_input")).toBeInTheDocument();
    expect(screen.queryByText("undefined")).not.toBeInTheDocument();
  });

  it("includes dark-mode surface classes for metadata tooltips", () => {
    render(
      <SpecsTooltip
        specTitle="input"
        specValues={[
          {
            badges: [{ label: "workflow_input", bgColor: "bg-gray-100", textColor: "text-gray-800" }],
          },
        ]}
      >
        <span>trigger</span>
      </SpecsTooltip>,
    );

    expect(screen.getByText("1 input").parentElement?.parentElement?.className).toContain("dark:bg-gray-900");
    expect(screen.getByText("1 input").className).toContain("dark:text-gray-400");
  });
});
