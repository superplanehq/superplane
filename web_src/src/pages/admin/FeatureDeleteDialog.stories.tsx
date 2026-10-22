import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { useState, type ComponentProps } from "react";
import { expect, userEvent, within } from "storybook/test";
import { Button } from "@/components/ui/button";
import { FeatureDeleteDialog } from "./FeatureDeleteDialog";

function InteractiveDialog(args: ComponentProps<typeof FeatureDeleteDialog>) {
  const [open, setOpen] = useState(true);
  const [deleted, setDeleted] = useState(false);
  const [trigger, setTrigger] = useState<HTMLButtonElement | null>(null);
  const [survivor, setSurvivor] = useState<HTMLButtonElement | null>(null);
  return (
    <>
      <Button ref={setSurvivor} variant="outline" onClick={() => console.log("Refresh catalog")}>
        Refresh catalog
      </Button>
      {!deleted && (
        <Button ref={setTrigger} variant="destructive" onClick={() => setOpen(true)}>
          Delete feature
        </Button>
      )}
      {deleted && <p role="status">Feature deleted.</p>}
      {open && (
        <FeatureDeleteDialog
          {...args}
          returnFocus={trigger}
          survivorFocus={survivor}
          onDeleted={async () => {
            await args.onDeleted();
            setDeleted(true);
          }}
          onClose={() => {
            args.onClose();
            setOpen(false);
          }}
        />
      )}
    </>
  );
}

const endpoint = "/admin/api/feature-definitions/custom_reports";
const meta = {
  title: "Admin/FeatureDeleteDialog",
  component: FeatureDeleteDialog,
  parameters: {
    layout: "centered",
    msw: { handlers: [http.delete(endpoint, () => new HttpResponse(null, { status: 204 }))] },
  },
  args: {
    feature: { key: "custom_reports", label: "Custom reports", gated: true },
    returnFocus: null,
    survivorFocus: null,
    onDeleted: async () => console.log("Feature deleted; refresh catalog"),
    onClose: () => console.log("Close feature delete dialog"),
  },
  render: (args) => <InteractiveDialog {...args} />,
} satisfies Meta<typeof FeatureDeleteDialog>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Confirmation: Story = {};

export const PendingDelete: Story = {
  parameters: {
    msw: {
      handlers: [
        http.delete(endpoint, async () => {
          await delay("infinite");
          return new HttpResponse(null, { status: 204 });
        }),
      ],
    },
  },
  play: async () => {
    const dialog = within(await within(document.body).findByRole("dialog"));
    await userEvent.click(dialog.getByRole("button", { name: "Delete feature" }));
    await expect(dialog.getByRole("button", { name: "Deleting..." })).toBeDisabled();
    await expect(dialog.getByRole("button", { name: "Keep feature" })).toBeDisabled();
  },
};

export const FailedDeleteWithRetry: Story = {
  parameters: {
    msw: {
      handlers: [
        http.delete(
          endpoint,
          () => new HttpResponse("SuperPlane could not delete the feature. Try again.", { status: 500 }),
          {
            once: true,
          },
        ),
        http.delete(endpoint, () => new HttpResponse(null, { status: 204 })),
      ],
    },
  },
  play: async () => {
    const dialog = within(await within(document.body).findByRole("dialog"));
    await userEvent.click(dialog.getByRole("button", { name: "Delete feature" }));
    await expect(await dialog.findByRole("alert")).toHaveTextContent("Try again.");
    await expect(dialog.getByRole("button", { name: "Delete feature" })).toBeEnabled();
  },
};
