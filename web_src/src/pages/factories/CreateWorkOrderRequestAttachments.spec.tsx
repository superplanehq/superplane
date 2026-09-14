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

describe("CreateWorkOrderRequestAttachments", () => {
  afterEach(async () => {
    cleanup();
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
  });

  it("stacks images and fans them horizontally on hover", () => {
    render(
      <div data-testid="create-work-order-request-dialog" className="create-work-order-request-attachments">
        <CreateWorkOrderRequestAttachments images={images} />
      </div>,
    );

    const stack = screen.getByLabelText(CREATE_WORK_ORDER_REQUEST_COPY.attachedImages);
    const cards = stack.querySelectorAll(".t-stack-card");
    expect(stack).toHaveClass("t-stack");
    expect(cards).toHaveLength(3);
    expect(cards[0].getAttribute("style")).toContain("--hx");
    expect(cards[2].getAttribute("style")).toContain("7.5rem");
  });

  it("keeps the fourth and fifth images on their own hover slots", async () => {
    const user = userEvent.setup();
    const five = [
      ...images,
      { id: "file-4", alt: "Invoice", src: "https://cdn.example.com/invoice.png" },
      { id: "file-5", alt: "Label", src: "https://cdn.example.com/label.png" },
    ];
    render(
      <div data-testid="create-work-order-request-dialog" className="create-work-order-request-attachments">
        <CreateWorkOrderRequestAttachments images={five} />
      </div>,
    );

    const cards = screen
      .getByLabelText(CREATE_WORK_ORDER_REQUEST_COPY.attachedImages)
      .querySelectorAll(".t-stack-card");
    expect(cards).toHaveLength(5);
    expect(cards[3].getAttribute("style")).toContain("11.25rem");
    expect(cards[4].getAttribute("style")).toContain("15rem");

    await user.click(screen.getByTestId("create-work-order-request-attachment-file-4"));
    expect(screen.getByTestId("create-work-order-request-image-expand").querySelector("img")).toHaveAttribute(
      "src",
      "https://cdn.example.com/invoice.png",
    );

    await user.click(screen.getByTestId("create-work-order-request-image-close"));
    await user.click(screen.getByTestId("create-work-order-request-attachment-file-5"));
    expect(screen.getByTestId("create-work-order-request-image-expand").querySelector("img")).toHaveAttribute(
      "src",
      "https://cdn.example.com/label.png",
    );
  });

  it("opens an attached image with a resize card and can remove it", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    render(
      <div data-testid="create-work-order-request-dialog" className="create-work-order-request-attachments">
        <CreateWorkOrderRequestAttachments images={images} onRemove={onRemove} />
      </div>,
    );

    await user.click(screen.getByTestId("create-work-order-request-attachment-file-2"));

    const expand = screen.getByTestId("create-work-order-request-image-expand");
    const backdrop = screen.getByTestId("create-work-order-request-image-backdrop");
    expect(expand).toHaveClass("t-resize");
    expect(screen.getByTestId("create-work-order-request-dialog").contains(backdrop)).toBe(false);
    expect(backdrop.parentElement).toHaveClass("t-resize-portal");
    expect(expand.querySelector("img")).toHaveAttribute("src", "https://cdn.example.com/receipt.png");
    expect(screen.getByTestId("create-work-order-request-image-close")).toHaveAccessibleName(
      CREATE_WORK_ORDER_REQUEST_COPY.closeImage,
    );
    expect(screen.getByTestId("create-work-order-request-image-remove")).toHaveAccessibleName(
      CREATE_WORK_ORDER_REQUEST_COPY.deleteImage,
    );

    await user.click(screen.getByTestId("create-work-order-request-image-close"));

    expect(onRemove).not.toHaveBeenCalled();
    expect(screen.queryByTestId("create-work-order-request-image-expand")?.isConnected ?? false).toBe(false);
  });

  it("deletes an expanded image from the bin control", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    render(
      <div data-testid="create-work-order-request-dialog" className="create-work-order-request-attachments">
        <CreateWorkOrderRequestAttachments images={images} onRemove={onRemove} />
      </div>,
    );

    await user.click(screen.getByTestId("create-work-order-request-attachment-file-2"));
    await user.click(screen.getByTestId("create-work-order-request-image-remove"));

    expect(onRemove).toHaveBeenCalledWith("file-2");
    expect(screen.queryByTestId("create-work-order-request-image-expand")).not.toBeInTheDocument();
  });
});
