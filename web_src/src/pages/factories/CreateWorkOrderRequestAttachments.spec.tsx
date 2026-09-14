import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { CreateWorkOrderRequestAttachments } from "./CreateWorkOrderRequestAttachments";
import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";
import type { CreateWorkOrderRequestImage } from "./lib/createWorkOrderRequestImages";

const images: CreateWorkOrderRequestImage[] = [
  { id: "file-1", alt: "Checkout", src: "https://cdn.example.com/checkout.png" },
  { id: "file-2", alt: "Receipt", src: "https://cdn.example.com/receipt.png" },
  { id: "file-3", alt: "Error", src: "https://cdn.example.com/error.png" },
];

function renderStack(stackImages: CreateWorkOrderRequestImage[], onRemove?: (id: string) => void) {
  return render(
    <div className="create-work-order-request-attachments">
      <CreateWorkOrderRequestAttachments images={stackImages} onRemove={onRemove} />
    </div>,
  );
}

describe("CreateWorkOrderRequestAttachments", () => {
  afterEach(async () => {
    cleanup();
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
  });

  it("keeps later images on their own hover slots and can open them", async () => {
    const user = userEvent.setup();
    const five = [
      ...images,
      { id: "file-4", alt: "Invoice", src: "https://cdn.example.com/invoice.png" },
      { id: "file-5", alt: "Label", src: "https://cdn.example.com/label.png" },
    ];
    renderStack(five);

    const cards = screen
      .getByLabelText(CREATE_WORK_ORDER_REQUEST_COPY.attachedImages)
      .querySelectorAll(".t-stack-card");
    expect(cards).toHaveLength(5);
    expect(cards[3].getAttribute("style")).toContain("11.25rem");
    expect(cards[4].getAttribute("style")).toContain("15rem");

    await user.click(screen.getByTestId("create-work-order-request-attachment-file-4"));
    expect(screen.getByRole("dialog", { name: "Invoice" })).toHaveAttribute(
      "data-testid",
      "create-work-order-request-image-expand",
    );
    expect(screen.getByTestId("create-work-order-request-image-close")).toHaveFocus();

    await user.click(screen.getByTestId("create-work-order-request-image-close"));
    await user.click(screen.getByTestId("create-work-order-request-attachment-file-5"));
    expect(screen.getByRole("dialog", { name: "Label" }).querySelector("img")).toHaveAttribute(
      "src",
      "https://cdn.example.com/label.png",
    );
  });

  it("deletes an expanded image from the bin control", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    renderStack(images, onRemove);

    await user.click(screen.getByTestId("create-work-order-request-attachment-file-2"));
    await user.click(screen.getByTestId("create-work-order-request-image-remove"));

    expect(onRemove).toHaveBeenCalledWith("file-2");
    expect(screen.queryByTestId("create-work-order-request-image-expand")).not.toBeInTheDocument();
  });
});
