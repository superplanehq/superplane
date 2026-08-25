import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeAll, describe, expect, it } from "vitest";

import { WorkOrderSplitRunDescription } from "./WorkOrderSplitRunDescription";

const emptyRect = {
  x: 0,
  y: 0,
  width: 0,
  height: 0,
  top: 0,
  right: 0,
  bottom: 0,
  left: 0,
  toJSON() {
    return this;
  },
};
const emptyRects = {
  item: () => null,
  length: 0,
  [Symbol.iterator]: function* () {},
};

beforeAll(() => {
  document.elementFromPoint = () => null;
  stubClientRects(Range.prototype);
  stubClientRects(Element.prototype);
  stubClientRects(Text.prototype);
});

function stubClientRects(target: object) {
  Object.defineProperty(target, "getBoundingClientRect", {
    configurable: true,
    value: () => emptyRect,
  });
  Object.defineProperty(target, "getClientRects", {
    configurable: true,
    value: () => emptyRects,
  });
}

describe("WorkOrderSplitRunDescription", () => {
  it("keeps a read-only description when the work order is not a draft", () => {
    render(<WorkOrderSplitRunDescription description="Retry webhooks after a timeout." />);

    expect(screen.getByTestId("work-order-description-markdown")).toHaveTextContent("Retry webhooks after a timeout.");
    expect(screen.queryByRole("button", { name: "Edit" })).not.toBeInTheDocument();
  });

  it("switches a draft description into edit mode in place", async () => {
    const user = userEvent.setup();
    render(<WorkOrderSplitRunDescription canEdit description="Retry webhooks after a timeout." />);

    expect(screen.getByRole("button", { name: "Edit" })).toHaveClass("underline");
    await user.click(screen.getByRole("button", { name: "Edit" }));
    expect(await screen.findByTestId("split-run-description-editor")).toBeInTheDocument();
    expect(screen.getByTestId("work-order-description-input")).toHaveClass("text-[13px]");
    expect(screen.queryByTestId("work-order-description-markdown")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByTestId("work-order-description-markdown")).toHaveTextContent("Retry webhooks after a timeout.");
    expect(screen.getByRole("button", { name: "Edit" })).toBeInTheDocument();
  });

  it("saves edits and discards them on cancel", async () => {
    const user = userEvent.setup();
    render(<WorkOrderSplitRunDescription canEdit description="Retry webhooks after a timeout." />);

    await user.click(screen.getByRole("button", { name: "Edit" }));
    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.paste("Saved retry policy.");
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(screen.getByTestId("work-order-description-markdown")).toHaveTextContent("Saved retry policy.");

    await user.click(screen.getByRole("button", { name: "Edit" }));
    await user.click(await screen.findByTestId("work-order-description-input"));
    await user.paste("Discard this.");
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByTestId("work-order-description-markdown")).toHaveTextContent("Saved retry policy.");
    expect(screen.queryByText("Discard this.")).not.toBeInTheDocument();
  });
});
