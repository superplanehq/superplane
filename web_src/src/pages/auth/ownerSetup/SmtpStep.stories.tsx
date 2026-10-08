import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect } from "react";
import { expect, userEvent, within } from "storybook/test";

import { OWNER_SETUP_COPY } from "./ownerSetupCopy";
import { SmtpStep } from "./SmtpStep";

function OwnerSetupStoryFrame({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen justify-center bg-background p-8">
      <div className="w-full max-w-xl">{children}</div>
    </div>
  );
}

function useSmtpSaveResponse(status: number) {
  useEffect(() => {
    const original = window.fetch.bind(window);
    window.fetch = (async (input: RequestInfo | URL) => {
      const url = String(input);
      if (!url.includes("/admin/api/installation/network-settings")) {
        return original(input);
      }
      return new Response("{}", { status });
    }) as typeof fetch;
    return () => {
      window.fetch = original;
    };
  }, [status]);
}

const meta = {
  title: "Auth/Owner setup/SMTP",
  component: SmtpStep,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Owner setup can save SMTP or skip this step. Skip leaves password login working.",
      },
    },
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <OwnerSetupStoryFrame>
        <Story />
      </OwnerSetupStoryFrame>
    ),
  ],
  args: {
    onContinue: () => {
      // eslint-disable-next-line no-console
      console.log("Continue");
    },
    onSkip: () => {
      // eslint-disable-next-line no-console
      console.log("Skip");
    },
  },
} satisfies Meta<typeof SmtpStep>;

export default meta;

type Story = StoryObj<typeof meta>;

/** Empty SMTP form. Save requires host, port, and from email. */
export const Form: Story = {};

/** Save with empty required fields. The form shows the required-field error. */
export const ValidationError: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByRole("button", { name: OWNER_SETUP_COPY.smtp.save }));
    await expect(canvas.getByText(OWNER_SETUP_COPY.smtp.required)).toBeInTheDocument();
  },
};

function SmtpSaveErrorStep(props: { onContinue: () => void; onSkip: () => void }) {
  useSmtpSaveResponse(500);
  return <SmtpStep {...props} />;
}

/** Save fails. The form keeps the values and shows a recovery error. */
export const SaveError: Story = {
  render: (args) => <SmtpSaveErrorStep {...args} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.type(canvas.getByTestId("owner-setup-smtp-host"), "smtp.example.com");
    await userEvent.type(canvas.getByTestId("owner-setup-smtp-from-email"), "noreply@example.com");
    await userEvent.click(canvas.getByRole("button", { name: OWNER_SETUP_COPY.smtp.save }));
    await expect(canvas.getByText(OWNER_SETUP_COPY.smtp.error)).toBeInTheDocument();
  },
};
