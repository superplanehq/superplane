import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { CreateWorkOrderRequestDialog } from "./CreateWorkOrderRequestDialog";
import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";

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

  it("names the dialog New task and keeps title optional", () => {
    renderRequestDialog();

    expect(screen.getByRole("dialog", { name: CREATE_WORK_ORDER_REQUEST_COPY.title })).toBeInTheDocument();
    expect(screen.getByTestId("create-work-order-request-title")).toHaveValue("");
    expect(screen.getByTestId("create-work-order-request-fullscreen")).toHaveAccessibleName(
      CREATE_WORK_ORDER_REQUEST_COPY.expand,
    );
  });

  it("uses one surface with attach, and disables create when the message is empty", () => {
    renderRequestDialog();

    const dialog = screen.getByTestId("create-work-order-request-dialog");
    expect(dialog.querySelector(".sp-user-note")).toBeNull();
    expect(screen.getByTestId("create-work-order-request-body").className).toMatch(
      /overflow-y-auto.*\[scrollbar-width:thin\]/,
    );
    const attach = screen.getByTestId("create-work-order-request-attach");
    expect(attach).toHaveAccessibleName(CREATE_WORK_ORDER_REQUEST_COPY.attach);
    expect(attach.compareDocumentPosition(screen.getByTestId("create-work-order-request-create"))).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    );
    const create = screen.getByTestId("create-work-order-request-create");
    expect(create).toBeDisabled();
    expect(create).toHaveAccessibleName(CREATE_WORK_ORDER_REQUEST_COPY.create);
    expect(create).toHaveClass("rounded-full");
    expect(create).not.toHaveTextContent("Create");
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

  it("uses a paperclip to attach images", () => {
    renderRequestDialog();

    const attach = screen.getByTestId("create-work-order-request-attach");
    expect(attach).toHaveAccessibleName(CREATE_WORK_ORDER_REQUEST_COPY.attach);
    expect(attach.querySelector(".t-goo-swap")).toBeNull();
    expect(screen.getByTestId("create-work-order-request-image-input").getAttribute("accept")).toContain("image/png");
    expect(screen.queryByTestId("create-work-order-request-file-input")).not.toBeInTheDocument();
  });

  it("fills the title from the first body line when the title is empty", () => {
    renderRequestDialog({ description: "Refunds fail on retry.\n\nMore context." });

    expect(screen.getByTestId("create-work-order-request-title")).toHaveValue("Refunds fail on retry.");
    expect(screen.getByTestId("create-work-order-request-create")).not.toBeDisabled();
  });

  it("keeps description images inline and also shows them in the footer stack", () => {
    renderRequestDialog({
      description: "Refunds fail.\n\n![Checkout](sp-file://file-1)",
      fileUrls: { "file-1": "https://cdn.example.com/checkout.png" },
    });

    expect(screen.getByTestId("create-work-order-request-attachments")).toBeInTheDocument();
    expect(screen.getByLabelText(CREATE_WORK_ORDER_REQUEST_COPY.attachedImages)).toBeInTheDocument();
    expect(screen.getByTestId("create-work-order-request-attachment-file-1")).toHaveAccessibleName(
      `${CREATE_WORK_ORDER_REQUEST_COPY.openImage}: Checkout`,
    );
    expect(screen.getByTestId("work-order-description-input").className).toMatch(/work-order-file-image]:max-h-40/);
    expect(screen.getByTestId("work-order-description-input")).not.toHaveClass("[&_.work-order-file-image]:hidden");
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

  it("attaches files without adding them to the description", async () => {
    const user = userEvent.setup();
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
      onDescriptionChange,
      onUploadFiles,
    });

    await user.upload(
      screen.getByTestId("create-work-order-request-image-input"),
      new File(["receipt"], "receipt.png", { type: "image/png" }),
    );

    expect(onDescriptionChange).not.toHaveBeenCalled();
    expect(screen.getByTestId("create-work-order-request-attachment-file-2")).toHaveAccessibleName(
      `${CREATE_WORK_ORDER_REQUEST_COPY.openImage}: receipt.png`,
    );
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

  it("wraps a long title instead of clipping it", () => {
    renderRequestDialog({
      description: "Refunds fail when the customer retries checkout.asda: extra words keep the title long.",
    });

    const title = screen.getByTestId("create-work-order-request-title");
    expect(title.tagName).toBe("TEXTAREA");
    expect(title).toHaveClass("wrap-anywhere");
    expect(title).toHaveClass("field-sizing-content");
  });

  it("expands width and height together", async () => {
    const user = userEvent.setup();
    renderRequestDialog();

    const dialog = screen.getByTestId("create-work-order-request-dialog");
    expect(dialog.className).toContain("sm:max-w-[32rem]");

    await user.click(screen.getByTestId("create-work-order-request-fullscreen"));

    expect(dialog.className).toContain("w-[90vw]");
    expect(dialog.className).toContain("h-[90vh]");
    expect(dialog.className).toContain("sm:max-w-none");
    expect(screen.getByTestId("create-work-order-request-fullscreen")).toHaveAccessibleName(
      CREATE_WORK_ORDER_REQUEST_COPY.collapse,
    );
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
});
