import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import { setWorkOrderFilePreviewUrl, workOrderFileRef } from "@/lib/workOrderFiles";

import { CreateWorkOrderRequestDialog } from "./CreateWorkOrderRequestDialog";
import { ComponentStoryShell } from "./__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "./__fixtures__/factoriesStoryTheme";

const STORY_IMAGE_ID = "story-checkout";
const STORY_IMAGE_URL = "https://placehold.co/320x160/png?text=Checkout";
const STORY_STACK_IMAGES = [
  { id: "story-checkout", url: "https://placehold.co/320x240/f97316/ffffff/png?text=Checkout", alt: "Checkout error" },
  { id: "story-receipt", url: "https://placehold.co/320x240/2563eb/ffffff/png?text=Receipt", alt: "Receipt" },
  { id: "story-retry", url: "https://placehold.co/320x240/059669/ffffff/png?text=Retry", alt: "Retry screen" },
  { id: "story-invoice", url: "https://placehold.co/320x240/7c3aed/ffffff/png?text=Invoice", alt: "Invoice" },
  { id: "story-label", url: "https://placehold.co/320x240/db2777/ffffff/png?text=Label", alt: "Label" },
] as const;
const STORY_IMAGE_MARKDOWN = `Refunds fail when the customer retries checkout.

![Checkout error](${workOrderFileRef(STORY_IMAGE_ID)})`;
const STORY_STACK_MARKDOWN = `Refunds fail when the customer retries checkout.

${STORY_STACK_IMAGES.map((image) => `![${image.alt}](${workOrderFileRef(image.id)})`).join("\n\n")}`;

const STORY_LONG_MARKDOWN = `Refunds fail when the customer retries checkout.

![Checkout error](${workOrderFileRef(STORY_IMAGE_ID)})

${Array.from({ length: 16 }, (_, index) => `Note ${index + 1}: the customer sees a double charge after retry.`).join("\n\n")}`;

async function mockUploadFiles(files: FileList | File[]): Promise<UploadedWorkOrderFile[]> {
  return Array.from(files).map((file, index) => {
    const id = `story-upload-${index}-${file.name}`;
    const previewUrl = URL.createObjectURL(file);
    setWorkOrderFilePreviewUrl(id, previewUrl);
    return {
      id,
      filename: file.name,
      contentType: file.type,
      ref: workOrderFileRef(id),
      previewUrl,
      isImage: file.type.startsWith("image/"),
    };
  });
}

function RequestDialogPlayground({
  initialDescription = "",
  isCreating = false,
  fileUrls,
}: {
  initialDescription?: string;
  isCreating?: boolean;
  fileUrls?: Record<string, string>;
}) {
  const [description, setDescription] = useState(initialDescription);

  return (
    <CreateWorkOrderRequestDialog
      open
      description={description}
      maxLength={5000}
      isCreating={isCreating}
      fileUrls={fileUrls}
      onClose={() => {
        console.log("close");
      }}
      onDescriptionChange={setDescription}
      onCreate={(draft) => {
        console.log("create", draft);
      }}
      onUploadFiles={mockUploadFiles}
    />
  );
}

/**
 * Presentational request composer. The live create dialog uses this when
 * Task Refinement is on.
 */
const meta = {
  title: "Factories/Pages/Create Task/Request card",
  parameters: {
    layout: "fullscreen",
    options: { showPanel: false },
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-svh bg-background p-0">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta;

export default meta;

type Story = StoryObj;

export const Empty: Story = {
  name: "Empty",
  render: () => <RequestDialogPlayground />,
};

export const Typed: Story = {
  name: "Typed message",
  render: () => <RequestDialogPlayground initialDescription={"Refunds fail when the customer retries checkout."} />,
};

export const WithImage: Story = {
  name: "Attached image",
  render: () => (
    <RequestDialogPlayground
      initialDescription={STORY_IMAGE_MARKDOWN}
      fileUrls={{ [STORY_IMAGE_ID]: STORY_IMAGE_URL }}
    />
  ),
};

export const WithImageStack: Story = {
  name: "Attached images",
  render: () => (
    <RequestDialogPlayground
      initialDescription={STORY_STACK_MARKDOWN}
      fileUrls={Object.fromEntries(STORY_STACK_IMAGES.map((image) => [image.id, image.url]))}
    />
  ),
};

export const Creating: Story = {
  name: "Creating",
  render: () => (
    <RequestDialogPlayground initialDescription={"Refunds fail when the customer retries checkout."} isCreating />
  ),
};

export const LongContent: Story = {
  name: "Long content",
  render: () => (
    <RequestDialogPlayground
      initialDescription={STORY_LONG_MARKDOWN}
      fileUrls={{ [STORY_IMAGE_ID]: STORY_IMAGE_URL }}
    />
  ),
};
