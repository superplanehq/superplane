import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { CreateWorkOrderRequestDialog } from "./CreateWorkOrderRequestDialog";
import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";

const { showErrorToast } = vi.hoisted(() => ({
  showErrorToast: vi.fn(),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast,
}));

vi.mock("./WorkOrderDescriptionEditor", () => ({
  WorkOrderDescriptionEditor: ({ autoFocus, className }: { autoFocus?: boolean; className?: string }) => (
    <textarea
      id="work-order-description-input"
      data-testid="work-order-description-input"
      className={className}
      autoFocus={autoFocus}
    />
  ),
}));

function renderRequestDialog(overrides: Partial<Parameters<typeof CreateWorkOrderRequestDialog>[0]> = {}) {
  return render(
    <CreateWorkOrderRequestDialog
      open
      description=""
      maxLength={5000}
      onClose={vi.fn()}
      onDescriptionChange={vi.fn()}
      onCreate={vi.fn()}
      onUploadFiles={vi.fn()}
      {...overrides}
    />,
  );
}

describe("CreateWorkOrderRequestDialog", () => {
  afterEach(async () => {
    cleanup();
    showErrorToast.mockReset();
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
  });

  it("focuses the description box when the dialog opens", async () => {
    renderRequestDialog();

    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });

    expect(screen.getByTestId("work-order-description-input")).toHaveFocus();
    expect(screen.getByTestId("create-work-order-request-title")).not.toHaveFocus();
  });

  it("keeps the title optional and disables create when the message is empty", () => {
    renderRequestDialog();

    expect(screen.getByRole("dialog", { name: CREATE_WORK_ORDER_REQUEST_COPY.title })).toBeInTheDocument();
    expect(screen.getByLabelText(CREATE_WORK_ORDER_REQUEST_COPY.titleField)).toHaveValue("");
    expect(screen.getByLabelText(CREATE_WORK_ORDER_REQUEST_COPY.placeholder)).toBeInTheDocument();
    expect(screen.getByTestId("create-work-order-request-title")).toHaveValue("");
    expect(screen.getByTestId("create-work-order-request-attach")).toHaveAccessibleName(
      CREATE_WORK_ORDER_REQUEST_COPY.attach,
    );
    expect(screen.getByTestId("create-work-order-request-image-input").getAttribute("accept")).toContain("image/png");
    expect(screen.getByTestId("create-work-order-request-create")).toBeDisabled();
  });

  it("creates the task with Command+Enter", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn();
    renderRequestDialog({ description: "Refunds fail on retry.", onCreate });

    await user.keyboard("{Meta>}{Enter}{/Meta}");

    expect(onCreate).toHaveBeenCalledWith({
      title: "Refunds fail on retry.",
      description: "Refunds fail on retry.",
    });
  });

  it("does not create with Command+Enter when the form is empty", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn();
    renderRequestDialog({ onCreate });

    await user.keyboard("{Meta>}{Enter}{/Meta}");

    expect(onCreate).not.toHaveBeenCalled();
  });

  it("does not close while an image upload is in progress", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderRequestDialog({ isUploading: true, onClose });

    await user.click(screen.getByTestId("create-work-order-request-close"));

    expect(onClose).not.toHaveBeenCalled();
  });

  it("fills the title from the first body line when the title is empty", () => {
    renderRequestDialog({ description: "Refunds fail on retry.\n\nMore context." });

    expect(screen.getByTestId("create-work-order-request-title")).toHaveValue("Refunds fail on retry.");
    expect(screen.getByTestId("create-work-order-request-create")).not.toBeDisabled();
  });

  it("adds attach-only images to the create payload", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn();
    const onDescriptionChange = vi.fn();
    const onUploadFiles = vi.fn().mockResolvedValue([
      {
        id: "file-2",
        filename: "receipt.png",
        contentType: "image/png",
        ref: "sp-file://file-2",
        previewUrl: "https://cdn.example.com/receipt.png",
        isImage: true,
      },
    ]);

    renderRequestDialog({
      description: "Refunds fail.",
      onCreate,
      onDescriptionChange,
      onUploadFiles,
    });

    await user.upload(
      screen.getByTestId("create-work-order-request-image-input"),
      new File(["receipt"], "receipt.png", { type: "image/png" }),
    );
    await user.click(screen.getByTestId("create-work-order-request-create"));

    expect(onDescriptionChange).not.toHaveBeenCalled();
    expect(onCreate).toHaveBeenCalledWith({
      title: "Refunds fail.",
      description: "Refunds fail.\n\n![receipt.png](sp-file://file-2)",
    });
  });

  it("shows description images and attach-only images in the same stack", async () => {
    const user = userEvent.setup();
    const onUploadFiles = vi.fn().mockResolvedValue([
      {
        id: "file-2",
        filename: "receipt.png",
        contentType: "image/png",
        ref: "sp-file://file-2",
        previewUrl: "https://cdn.example.com/receipt.png",
        isImage: true,
      },
    ]);

    renderRequestDialog({
      description: "Refunds fail.\n\n![Checkout](sp-file://file-1)",
      fileUrls: { "file-1": "https://cdn.example.com/checkout.png" },
      onUploadFiles,
    });

    await user.upload(
      screen.getByTestId("create-work-order-request-image-input"),
      new File(["receipt"], "receipt.png", { type: "image/png" }),
    );

    expect(screen.getByTestId("create-work-order-request-attachment-file-1")).toBeInTheDocument();
    expect(screen.getByTestId("create-work-order-request-attachment-file-2")).toBeInTheDocument();
  });

  it("keeps a title the user types", async () => {
    const user = userEvent.setup();
    renderRequestDialog({ description: "Refunds fail on retry." });

    const title = screen.getByTestId("create-work-order-request-title");
    await user.clear(title);
    await user.type(title, "Checkout retry");

    expect(title).toHaveValue("Checkout retry");
  });

  it("does not attach a non-image file", async () => {
    const user = userEvent.setup();
    const onUploadFiles = vi.fn().mockResolvedValue([
      {
        id: "file-3",
        filename: "notes.md",
        contentType: "text/markdown",
        ref: "sp-file://file-3",
        previewUrl: "https://cdn.example.com/notes.md",
        isImage: false,
      },
    ]);

    renderRequestDialog({
      description: "Refunds fail.",
      onUploadFiles,
    });

    await user.upload(
      screen.getByTestId("create-work-order-request-image-input"),
      new File(["notes"], "notes.md", { type: "text/markdown" }),
    );

    expect(screen.queryByTestId("create-work-order-request-attachment-file-3")).not.toBeInTheDocument();
  });

  it("removes a description image from the expand card", async () => {
    const user = userEvent.setup();
    const onDescriptionChange = vi.fn();
    renderRequestDialog({
      description: "Refunds fail.\n\n![Checkout](sp-file://file-1)",
      fileUrls: { "file-1": "https://cdn.example.com/checkout.png" },
      onDescriptionChange,
    });

    await user.click(screen.getByTestId("create-work-order-request-attachment-file-1"));
    await user.click(screen.getByTestId("create-work-order-request-image-remove"));

    expect(onDescriptionChange).toHaveBeenCalledWith("Refunds fail.");
  });

  it("uploads only the remaining image slots", async () => {
    const user = userEvent.setup();
    const onUploadFiles = vi.fn().mockImplementation(async (files: FileList | File[]) =>
      Array.from(files).map((file, index) => ({
        id: `file-${index + 9}`,
        filename: file.name,
        contentType: "image/png",
        ref: `sp-file://file-${index + 9}`,
        previewUrl: `https://cdn.example.com/${file.name}`,
        isImage: true,
      })),
    );
    const description = Array.from({ length: 7 }, (_, index) => `![shot-${index}](sp-file://file-${index})`).join(
      "\n\n",
    );

    renderRequestDialog({ description, onUploadFiles });

    await user.upload(
      screen.getByTestId("create-work-order-request-image-input"),
      Array.from({ length: 3 }, (_, index) => new File(["img"], `extra-${index}.png`, { type: "image/png" })),
    );

    expect(onUploadFiles).toHaveBeenCalledTimes(1);
    expect(Array.from(onUploadFiles.mock.calls[0][0] as File[])).toHaveLength(1);
    expect(showErrorToast).toHaveBeenCalledWith("Attachments are limited to 8 images.");
  });
});
