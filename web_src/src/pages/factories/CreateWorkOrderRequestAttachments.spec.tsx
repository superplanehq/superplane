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

function renderAttachments(nextImages: CreateWorkOrderRequestImage[], onRemove?: (id: string) => void) {
  return render(<CreateWorkOrderRequestAttachments images={nextImages} onRemove={onRemove} />);
}

describe("CreateWorkOrderRequestAttachments", () => {
  afterEach(async () => {
    cleanup();
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
  });

  it("shows images in a flat row and can open and close them", async () => {
    const user = userEvent.setup();
    const five = [
      ...images,
      { id: "file-4", alt: "Invoice", src: "https://cdn.example.com/invoice.png" },
      { id: "file-5", alt: "Label", src: "https://cdn.example.com/label.png" },
    ];
    renderAttachments(five);

    const row = screen.getByLabelText(CREATE_WORK_ORDER_REQUEST_COPY.attachedImages);
    expect(row.className).toContain("flex-wrap");
    expect(row.querySelectorAll("[style*='--rot'], [style*='--hx'], [style*='rotate']")).toHaveLength(0);
    expect(row.querySelectorAll("[data-testid^='create-work-order-request-attachment-file-']")).toHaveLength(5);

    await user.click(screen.getByTestId("create-work-order-request-attachment-file-4"));
    expect(screen.getByRole("dialog", { name: "Invoice" })).toHaveAttribute(
      "data-testid",
      "create-work-order-request-image-expand",
    );
    expect(screen.getByTestId("create-work-order-request-image-close")).toHaveFocus();

    await user.click(screen.getByTestId("create-work-order-request-image-close"));
    expect(screen.getByTestId("create-work-order-request-attachment-file-4")).toHaveFocus();
    await user.click(screen.getByTestId("create-work-order-request-attachment-file-5"));
    expect(screen.getByRole("dialog", { name: "Label" }).querySelector("img")).toHaveAttribute(
      "src",
      "https://cdn.example.com/label.png",
    );
  });

  it("removes an image from its tile without opening the preview", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    renderAttachments(images, onRemove);

    await user.click(screen.getByTestId("create-work-order-request-tile-remove-file-1"));

    expect(onRemove).toHaveBeenCalledWith("file-1");
    expect(screen.queryByTestId("create-work-order-request-image-expand")).not.toBeInTheDocument();
  });

  it("deletes an expanded image from the bin control", async () => {
    const user = userEvent.setup();
    const onRemove = vi.fn();
    renderAttachments(images, onRemove);

    await user.click(screen.getByTestId("create-work-order-request-attachment-file-2"));
    expect(screen.getByTestId("create-work-order-request-image-close")).toHaveFocus();
    await user.tab();
    expect(screen.getByTestId("create-work-order-request-image-remove")).toHaveFocus();
    await user.click(screen.getByTestId("create-work-order-request-image-remove"));

    expect(onRemove).toHaveBeenCalledWith("file-2");
    expect(screen.queryByTestId("create-work-order-request-image-expand")).not.toBeInTheDocument();
  });

  it("keeps a type label on tiles with no frame and still opens them", async () => {
    const user = userEvent.setup();
    renderAttachments([
      { id: "clip", alt: "Demo", src: "https://cdn.example.com/demo.mp4", isVideo: true },
      { id: "voice", alt: "Note", src: "https://cdn.example.com/note.mp3", isAudio: true },
      {
        id: "hosted",
        alt: "Walkthrough",
        src: "https://www.youtube.com/watch?v=abcdefghijk",
        hostedVideo: {
          providerId: "cleanshot",
          providerName: "CleanShot",
          id: "abcd1234",
          pageUrl: "https://cln.sh/abcd1234",
          embedUrl: null,
        },
      },
    ]);

    expect(screen.getByTestId("create-work-order-request-attachment-clip").querySelector("video")).toHaveAttribute(
      "src",
      "https://cdn.example.com/demo.mp4",
    );
    expect(screen.getByText("Audio")).toBeInTheDocument();
    expect(screen.getByText("CleanShot")).toBeInTheDocument();

    await user.click(screen.getByTestId("create-work-order-request-attachment-voice"));
    expect(screen.getByRole("dialog", { name: "Note" }).querySelector("audio")).toHaveAttribute(
      "src",
      "https://cdn.example.com/note.mp3",
    );
    await user.click(screen.getByTestId("create-work-order-request-image-close"));

    await user.click(screen.getByTestId("create-work-order-request-attachment-hosted"));
    expect(screen.getByRole("dialog", { name: "Walkthrough" }).querySelector("a")).toHaveAttribute(
      "href",
      "https://cln.sh/abcd1234",
    );
    await user.click(screen.getByTestId("create-work-order-request-image-close"));

    await user.click(screen.getByTestId("create-work-order-request-attachment-clip"));
    expect(screen.getByRole("dialog", { name: "Demo" }).querySelector("video")).toHaveAttribute(
      "src",
      "https://cdn.example.com/demo.mp4",
    );
  });
});
