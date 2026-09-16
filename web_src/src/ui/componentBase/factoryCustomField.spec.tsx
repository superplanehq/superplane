import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { ComponentBase } from "./index";

describe("ComponentBase factory custom field", () => {
  it("shows the custom field on factory node cards", () => {
    render(
      <ComponentBase
        title="Run Shell Command"
        componentLabel="Run Shell Command"
        nodeName="Build Storybook"
        isFactoryApp
        canvasMode="live"
        customField={<button type="button">See logs</button>}
      />,
    );

    expect(screen.getByRole("button", { name: "See logs" })).toBeInTheDocument();
  });

  it("hides live-only custom fields in edit mode", () => {
    render(
      <ComponentBase
        title="Wait"
        isFactoryApp
        canvasMode="edit"
        customFieldVisibility="live-only"
        customField={<button type="button">Push through</button>}
      />,
    );

    expect(screen.queryByRole("button", { name: "Push through" })).not.toBeInTheDocument();
  });
});
