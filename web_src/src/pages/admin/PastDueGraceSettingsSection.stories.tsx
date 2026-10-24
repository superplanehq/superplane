import type { Meta, StoryObj } from "@storybook/react-vite";
import { userEvent } from "storybook/test";
import { useRef, useState, type ComponentProps } from "react";
import { Toaster } from "sonner";
import { PastDueGraceSettingsSection } from "./PastDueGraceSettingsSection";

function InteractiveGraceSettings({
  failFirstSave = false,
  ...args
}: ComponentProps<typeof PastDueGraceSettingsSection> & { failFirstSave?: boolean }) {
  const [settings, setSettings] = useState(args.settings);
  const failed = useRef(false);
  return (
    <div className="max-w-3xl p-4">
      <PastDueGraceSettingsSection
        {...args}
        settings={settings}
        patchSettings={async (body, successMessage, fallbackError) => {
          console.log("Save payment settings", body);
          if (failFirstSave && !failed.current) {
            failed.current = true;
            throw new Error(fallbackError);
          }
          await args.patchSettings(body, successMessage, fallbackError);
          setSettings({ past_due_grace_days: Number(body.past_due_grace_days) });
        }}
      />
      <Toaster />
    </div>
  );
}

const meta = {
  title: "Admin/Past due grace settings",
  component: PastDueGraceSettingsSection,
  args: {
    settings: { past_due_grace_days: 0 },
    patchSettings: async () => {},
  },
  render: (args) => <InteractiveGraceSettings {...args} />,
} satisfies Meta<typeof PastDueGraceSettingsSection>;

export default meta;
type Story = StoryObj<typeof meta>;

export const NoGrace: Story = {};
export const SevenDayGrace: Story = { args: { settings: { past_due_grace_days: 7 } } };
export const MissingSettings: Story = { args: { settings: null } };

const saveChangedGrace: NonNullable<Story["play"]> = async ({ canvas }) => {
  const input = canvas.getByRole("spinbutton", { name: "Payment grace period (days)" });
  await userEvent.clear(input);
  await userEvent.type(input, "7");
  await userEvent.click(canvas.getByRole("button", { name: "Save payment settings" }));
};

export const Saving: Story = {
  args: { patchSettings: () => new Promise<void>(() => {}) },
  play: saveChangedGrace,
};

export const SaveError: Story = {
  render: (args) => <InteractiveGraceSettings {...args} failFirstSave />,
  play: saveChangedGrace,
};
