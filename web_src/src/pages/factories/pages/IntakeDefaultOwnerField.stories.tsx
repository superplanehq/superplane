import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { delay, http, HttpResponse } from "msw";
import { useState, type ComponentProps, type ReactNode } from "react";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { IntakeDefaultOwnerField } from "./IntakeDefaultOwnerField";

const ALICE_ID = "alice";

function member(id: string, name: string) {
  return {
    metadata: { id, email: `${id}@example.com` },
    spec: { displayName: name },
  };
}

const members = [member(ALICE_ID, "Alice"), member("bob", "Bob")];

function usersHandler(mode: "ready" | "loading" | "error") {
  return http.get("/api/v1/users", async () => {
    if (mode === "error") {
      return new HttpResponse(null, { status: 500 });
    }
    if (mode === "loading") {
      await delay("infinite");
    }
    return HttpResponse.json({ users: members });
  });
}

function FreshQueryClient({ children }: { children: ReactNode }) {
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, staleTime: Infinity } } }),
  );
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

function InteractiveOwnerField({
  value: initialValue,
  onChange,
  ...props
}: ComponentProps<typeof IntakeDefaultOwnerField>) {
  const [value, setValue] = useState(initialValue);
  return (
    <IntakeDefaultOwnerField
      {...props}
      value={value}
      onChange={(ownerId) => {
        console.log("Default task owner", ownerId);
        onChange(ownerId);
        setValue(ownerId);
      }}
    />
  );
}

const meta = {
  title: "Factories/Components/IntakeDefaultOwnerField",
  component: IntakeDefaultOwnerField,
  parameters: { layout: "padded", msw: { handlers: [usersHandler("ready")] } },
  args: {
    organizationId: "intake-owner-org",
    value: "",
    disabled: false,
    onChange: (ownerId: string) => console.log("Default task owner", ownerId),
  },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <FreshQueryClient>
        <ComponentStoryShell className="max-w-md bg-background p-6">
          <Story />
        </ComponentStoryShell>
      </FreshQueryClient>
    ),
  ],
  render: (args) => <InteractiveOwnerField {...args} />,
} satisfies Meta<typeof IntakeDefaultOwnerField>;

export default meta;
type Story = StoryObj<typeof meta>;

export const NoOwner: Story = {
  name: "No owner",
  args: { organizationId: "intake-owner-none", value: "" },
};

export const SelectedMember: Story = {
  name: "Selected member",
  args: { organizationId: "intake-owner-selected", value: ALICE_ID },
};

export const Loading: Story = {
  name: "Loading",
  args: { organizationId: "intake-owner-loading", value: ALICE_ID },
  parameters: { msw: { handlers: [usersHandler("loading")] } },
};

export const MissingMember: Story = {
  name: "Missing member",
  args: { organizationId: "intake-owner-missing", value: "removed-member" },
};

export const FailedMemberRequest: Story = {
  name: "Failed member request",
  args: { organizationId: "intake-owner-error", value: ALICE_ID },
  parameters: { msw: { handlers: [usersHandler("error")] } },
};

export const Disabled: Story = {
  name: "Disabled",
  args: { organizationId: "intake-owner-disabled", value: ALICE_ID, disabled: true },
};
