import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { ComponentStoryShell } from "@/pages/factories/__fixtures__/ComponentStoryShell";

import { CreateIntakeDialog } from "./CreateIntakeDialog";

function mockCreateIntakeFetch(args: { ok: boolean }) {
  return async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.toString() : input.url;
    const method = init?.method ?? "GET";
    if (url.endsWith("/admin/api/intake-catalog") && method === "POST") {
      if (!args.ok) {
        return new Response("an intake with this key already exists", { status: 409 });
      }
      return Response.json({
        key: "linear-issues",
        name: "Linear issues",
        category: "issue_tracking",
        status: "planned",
        status_note: "",
        implemented: true,
        deletable: false,
        created_at: new Date(Date.now() - 60_000).toISOString(),
        updated_at: new Date().toISOString(),
        updated_by_name: "Ada Lovelace",
      });
    }
    return new Response("not mocked", { status: 500 });
  };
}

function CreateDialogStory({ ok }: { ok: boolean }) {
  const [open, setOpen] = useState(false);
  const queryClient = useMemo(() => new QueryClient(), []);

  useEffect(() => {
    const original = window.fetch.bind(window);
    window.fetch = mockCreateIntakeFetch({ ok });
    return () => {
      window.fetch = original;
    };
  }, [ok]);

  return (
    <QueryClientProvider client={queryClient}>
      <div className="flex flex-col gap-4">
        <Button variant="outline" onClick={() => setOpen(true)}>
          Open dialog
        </Button>
        <CreateIntakeDialog
          open={open}
          onOpenChange={setOpen}
          onCreated={(entry) => {
            // eslint-disable-next-line no-console
            console.log("onCreated", entry);
          }}
        />
      </div>
    </QueryClientProvider>
  );
}

const meta = {
  title: "Admin/Intakes/CreateIntakeDialog",
  component: CreateIntakeDialog,
  decorators: [
    (Story) => (
      <ComponentStoryShell className="min-h-screen bg-slate-100 p-6 dark:bg-gray-950">
        <div className="max-w-xl">
          <Story />
        </div>
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof CreateIntakeDialog>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Success: Story = {
  render: () => <CreateDialogStory ok />,
};

export const Conflict: Story = {
  render: () => <CreateDialogStory ok={false} />,
};
