import { act, cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { WORK_ORDER_VISUAL_FILE_ACCEPT } from "@/lib/workOrderFiles";

import { CreateWorkOrderRequestDialog } from "./CreateWorkOrderRequestDialog";
import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";

const { showErrorToast } = vi.hoisted(() => ({
  showErrorToast: vi.fn(),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast,
}));

vi.mock("./WorkOrderDescriptionEditor", () => ({
  WorkOrderDescriptionEditor: ({
    autoFocus,
    className,
    value,
    onChange,
    onFocus,
  }: {
    autoFocus?: boolean;
    className?: string;
    value?: string;
    onChange?: (next: string) => void;
    onFocus?: () => void;
  }) => (
    <textarea
      id="work-order-description-input"
      data-testid="work-order-description-input"
      className={className}
      autoFocus={autoFocus}
      value={value}
      onChange={(event) => onChange?.(event.target.value)}
      onFocus={() => onFocus?.()}
    />
  ),
}));

type ResultEvent = {
  resultIndex: number;
  results: Array<{ isFinal: boolean; 0: { transcript: string } }>;
};

class FakeSpeechRecognition {
  static instances: FakeSpeechRecognition[] = [];

  continuous = false;
  interimResults = false;
  lang = "";
  onresult: ((event: ResultEvent) => void) | null = null;
  onerror: ((event: { error: string }) => void) | null = null;
  onend: (() => void) | null = null;
  start = vi.fn();
  stop = vi.fn();
  abort = vi.fn();

  constructor() {
    FakeSpeechRecognition.instances.push(this);
  }
}

function latestRecognition(): FakeSpeechRecognition {
  const recognition = FakeSpeechRecognition.instances.at(-1);
  if (!recognition) {
    throw new Error("Speech recognition was not created");
  }
  return recognition;
}

function emitTranscript(transcript: string, isFinal: boolean) {
  latestRecognition().onresult?.({
    resultIndex: 0,
    results: [Object.assign([{ transcript }], { isFinal, 0: { transcript } })],
  });
}

function emitFinalPhrases(transcripts: string[]) {
  latestRecognition().onresult?.({
    resultIndex: 0,
    results: transcripts.map((transcript) => Object.assign([{ transcript }], { isFinal: true, 0: { transcript } })),
  });
}

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
    FakeSpeechRecognition.instances = [];
    vi.unstubAllGlobals();
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
    expect(screen.getByTestId("create-work-order-request-image-input").getAttribute("accept")).toBe(
      WORK_ORDER_VISUAL_FILE_ACCEPT,
    );
    expect(screen.getByTestId("create-work-order-request-create")).toBeDisabled();
  });

  it("shows Command plus Enter on Mac", () => {
    vi.stubGlobal("navigator", { platform: "MacIntel" });
    renderRequestDialog();

    expect(screen.getByTestId("create-work-order-request-create-kbd")).toHaveTextContent("⌘Enter");
  });

  it("shows Control plus Enter on Windows", () => {
    vi.stubGlobal("navigator", { platform: "Win32" });
    renderRequestDialog();

    expect(screen.getByTestId("create-work-order-request-create-kbd")).toHaveTextContent("Ctrl+Enter");
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

  it("strips paired markdown emphasis from a derived title", () => {
    renderRequestDialog({ description: "**Refunds fail on retry.**" });

    expect(screen.getByTestId("create-work-order-request-title")).toHaveValue("Refunds fail on retry.");
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
    expect(showErrorToast).toHaveBeenCalledWith("Attachments are limited to 8 images or videos.");
  });

  it("hides the dictate button when speech recognition is missing", () => {
    renderRequestDialog();

    expect(screen.queryByTestId("dictate-button")).not.toBeInTheDocument();
  });

  it("shows the dictate button and does not listen until click", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderRequestDialog();

    expect(screen.getByTestId("dictate-button")).toHaveAccessibleName(CREATE_WORK_ORDER_REQUEST_COPY.dictate);
    expect(FakeSpeechRecognition.instances).toHaveLength(0);

    await user.click(screen.getByTestId("dictate-button"));

    expect(latestRecognition().start).toHaveBeenCalledTimes(1);
  });

  it("shows the interim phrase without appending it", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    const onDescriptionChange = vi.fn();
    renderRequestDialog({ onDescriptionChange });

    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitTranscript("Fix refunds", false);
    });

    expect(screen.getByTestId("dictate-interim")).toHaveTextContent("Fix refunds");
    expect(onDescriptionChange).not.toHaveBeenCalled();
    expect(screen.getByTestId("create-work-order-request-title")).toHaveValue("");
  });

  it("appends a final phrase to the description by default", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    const onDescriptionChange = vi.fn();
    renderRequestDialog({ onDescriptionChange });

    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitTranscript("Refunds fail on retry", true);
    });

    expect(onDescriptionChange).toHaveBeenCalledWith("Refunds fail on retry");
  });

  it("keeps every final phrase from one recognition event", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    const onDescriptionChange = vi.fn();
    renderRequestDialog({ description: "Refunds fail.", onDescriptionChange });

    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitFinalPhrases(["Retry the job", "Check logs"]);
    });

    expect(onDescriptionChange).toHaveBeenNthCalledWith(1, "Refunds fail. Retry the job");
    expect(onDescriptionChange).toHaveBeenNthCalledWith(2, "Refunds fail. Retry the job Check logs");
  });

  it("appends a final phrase to the title when the title was last focused", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    const onDescriptionChange = vi.fn();
    renderRequestDialog({ onDescriptionChange });

    await user.click(screen.getByTestId("create-work-order-request-title"));
    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitTranscript("Checkout retry", true);
    });

    expect(screen.getByTestId("create-work-order-request-title")).toHaveValue("Checkout retry");
    expect(onDescriptionChange).not.toHaveBeenCalled();
  });

  it("inserts a space and truncates a description to 5000 characters", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    const onDescriptionChange = vi.fn();
    const current = "B".repeat(4996);
    renderRequestDialog({ description: current, onDescriptionChange });

    await user.click(screen.getByTestId("work-order-description-input"));
    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitTranscript("overflow", true);
    });

    expect(onDescriptionChange).toHaveBeenCalledWith(`${current} ove`);
  });

  it("stops dictation when the dialog closes", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderRequestDialog();

    await user.click(screen.getByTestId("dictate-button"));
    const recognition = latestRecognition();
    await user.click(screen.getByTestId("create-work-order-request-close"));

    expect(recognition.abort).toHaveBeenCalled();
  });

  it("stops dictation before create", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    const onCreate = vi.fn();
    renderRequestDialog({ description: "Refunds fail on retry.", onCreate });

    await user.click(screen.getByTestId("dictate-button"));
    const recognition = latestRecognition();
    await user.click(screen.getByTestId("create-work-order-request-create"));

    expect(recognition.abort).toHaveBeenCalled();
    expect(onCreate).toHaveBeenCalled();
  });

  it("shows a permission error toast and returns to idle", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderRequestDialog();

    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      latestRecognition().onerror?.({ error: "not-allowed" });
    });

    expect(showErrorToast).toHaveBeenCalledWith(CREATE_WORK_ORDER_REQUEST_COPY.microphoneDenied);
    expect(screen.getByTestId("dictate-button")).toHaveAttribute("aria-pressed", "false");
  });
});
