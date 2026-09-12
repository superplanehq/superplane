import type { Meta, StoryObj } from "@storybook/react-vite";
import { Bug, MessageSquare } from "lucide-react";
import { useEffect, useState } from "react";
import { MemoryRouter } from "react-router";
import { Toaster } from "sonner";
import { userEvent, within } from "storybook/test";

import { FeedbackDialog } from "@/components/FeedbackDialog";
import { Button } from "@/components/ui/button";
import type { FeedbackCategory } from "@/lib/submitFeedback";

/**
 * Start-menu feedback form. Report issue opens Report a bug. Send feedback
 * opens Something else. Submit posts to POST /api/v1/me/feedback.
 */
const meta = {
  title: "Components/FeedbackDialog",
  component: FeedbackDialog,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "The organization start menu opens this form. Report issue selects Report a bug. Send feedback selects Something else. The user can attach one allowed file up to 5 MB.",
      },
    },
  },
  tags: ["autodocs"],
  decorators: [
    (Story) => (
      <MemoryRouter initialEntries={["/acme/apps/deploy"]}>
        <div className="flex min-h-[560px] w-[760px] items-start justify-center bg-gray-50 p-6 dark:bg-gray-950">
          <Story />
        </div>
        <Toaster position="bottom-center" closeButton />
      </MemoryRouter>
    ),
  ],
} satisfies Meta<typeof FeedbackDialog>;

export default meta;

type Story = StoryObj<typeof meta>;

type SubmitMode = "success" | "error" | "pending";

function useFeedbackSubmit(mode: SubmitMode) {
  useEffect(() => {
    const originalFetch = window.fetch.bind(window);
    window.fetch = (input, init) => {
      const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
      if (!url.includes("/api/v1/me/feedback")) {
        return originalFetch(input, init);
      }
      if (mode === "pending") {
        return new Promise(() => undefined);
      }
      if (mode === "error") {
        return Promise.resolve(
          new Response(JSON.stringify({ message: "SuperPlane could not send your feedback. Try again." }), {
            status: 500,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      return Promise.resolve(new Response(null, { status: 204 }));
    };
    return () => {
      window.fetch = originalFetch;
    };
  }, [mode]);
}

function FeedbackDialogPlayground({
  initialCategory,
  initialOpen = true,
  submitMode = "success",
}: {
  initialCategory?: FeedbackCategory;
  initialOpen?: boolean;
  submitMode?: SubmitMode;
}) {
  const [open, setOpen] = useState(initialOpen);
  const [category, setCategory] = useState(initialCategory);
  useFeedbackSubmit(submitMode);

  const openWith = (nextCategory: FeedbackCategory) => {
    setCategory(nextCategory);
    setOpen(true);
  };

  return (
    <>
      <div className="flex gap-2">
        <Button type="button" variant="outline" onClick={() => openWith("bug")}>
          <Bug className="h-4 w-4" aria-hidden />
          Report issue
        </Button>
        <Button type="button" variant="outline" onClick={() => openWith("other")}>
          <MessageSquare className="h-4 w-4" aria-hidden />
          Send feedback
        </Button>
      </div>
      <FeedbackDialog open={open} onOpenChange={setOpen} organizationId="org-storybook" initialCategory={category} />
    </>
  );
}

const dialogCanvas = () => within(document.body);

const darkGlobals = { theme: "dark" as const };

/** Start-menu actions. Select Report issue or Send feedback to open the form. */
export const StartMenu: Story = {
  name: "0 Start menu",
  args: {
    open: false,
    onOpenChange: () => undefined,
  },
  render: () => <FeedbackDialogPlayground initialOpen={false} />,
};

/** Report issue opens the form with Report a bug selected. */
export const ReportIssue: Story = {
  name: "1 Report issue",
  args: {
    open: true,
    onOpenChange: () => undefined,
    initialCategory: "bug",
  },
  render: () => <FeedbackDialogPlayground initialCategory="bug" />,
};

export const ReportIssueDark: Story = {
  name: "1b Report issue (dark)",
  globals: darkGlobals,
  args: {
    open: true,
    onOpenChange: () => undefined,
    initialCategory: "bug",
  },
  render: () => <FeedbackDialogPlayground initialCategory="bug" />,
};

/** Send feedback opens the form with Something else selected. */
export const SendFeedback: Story = {
  name: "2 Send feedback",
  args: {
    open: true,
    onOpenChange: () => undefined,
    initialCategory: "other",
  },
  render: () => <FeedbackDialogPlayground initialCategory="other" />,
};

export const SendFeedbackDark: Story = {
  name: "2b Send feedback (dark)",
  globals: darkGlobals,
  args: {
    open: true,
    onOpenChange: () => undefined,
    initialCategory: "other",
  },
  render: () => <FeedbackDialogPlayground initialCategory="other" />,
};

/** Request a feature is the third category on the same form. */
export const RequestFeature: Story = {
  name: "3 Request a feature",
  args: {
    open: true,
    onOpenChange: () => undefined,
    initialCategory: "feature",
  },
  render: () => <FeedbackDialogPlayground initialCategory="feature" />,
};

/** Send stays in the loading state while the request is in flight. */
export const Sending: Story = {
  name: "4 Sending",
  args: {
    open: true,
    onOpenChange: () => undefined,
    initialCategory: "bug",
  },
  render: () => <FeedbackDialogPlayground initialCategory="bug" submitMode="pending" />,
  play: async () => {
    const body = dialogCanvas();
    await userEvent.type(body.getByTestId("feedback-details"), "The canvas did not load.");
    await userEvent.click(body.getByTestId("feedback-send"));
  },
};

/** Submit failure shows an error toast. The form stays open so the user can retry. */
export const SubmitError: Story = {
  name: "5 Submit error",
  args: {
    open: true,
    onOpenChange: () => undefined,
    initialCategory: "bug",
  },
  render: () => <FeedbackDialogPlayground initialCategory="bug" submitMode="error" />,
  play: async () => {
    const body = dialogCanvas();
    await userEvent.type(body.getByTestId("feedback-details"), "The canvas did not load.");
    await userEvent.click(body.getByTestId("feedback-send"));
  },
};

/** A disallowed file stays off the form and shows an inline error. */
export const InvalidAttachment: Story = {
  name: "6 Invalid attachment",
  args: {
    open: true,
    onOpenChange: () => undefined,
    initialCategory: "bug",
  },
  render: () => <FeedbackDialogPlayground initialCategory="bug" />,
  play: async () => {
    const body = dialogCanvas();
    const file = new File(["binary"], "notes.exe", { type: "application/octet-stream" });
    await userEvent.upload(body.getByTestId("feedback-file-input"), file);
  },
};

/** An allowed text file appears on the form and can be removed. */
export const AttachedFile: Story = {
  name: "7 Attached file",
  args: {
    open: true,
    onOpenChange: () => undefined,
    initialCategory: "other",
  },
  render: () => <FeedbackDialogPlayground initialCategory="other" />,
  play: async () => {
    const body = dialogCanvas();
    const file = new File(["The canvas did not load."], "notes.txt", { type: "text/plain" });
    await userEvent.upload(body.getByTestId("feedback-file-input"), file);
  },
};
