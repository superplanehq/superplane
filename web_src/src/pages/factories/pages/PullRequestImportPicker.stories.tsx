import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, useState } from "react";
import { PullRequestImportResults } from "./PullRequestImportPicker";

const meta = {
  title: "Factories/Pull request import",
  component: PullRequestImportResults,
  decorators: [
    function Interactive(Story, { args }) {
      const [query, setQuery] = useState(args.query);
      useEffect(() => setQuery(args.query), [args.query]);
      return (
        <div className="w-96 rounded-md border p-2">
          <Story
            args={{
              ...args,
              query,
              onQueryChange: (query: string) => {
                setQuery(query);
                args.onQueryChange(query);
              },
            }}
          />
        </div>
      );
    },
  ],
  args: {
    query: "",
    items: [
      { url: "https://github.com/acme/app/pull/42", title: "Fix retry handling", number: "42", draft: true },
      {
        url: "https://github.com/acme/app/pull/43",
        title: "Fix duplicate imports",
        number: "43",
        existingTask: { id: "task-43" },
      },
    ],
    onQueryChange: (query) => console.log("Search", query),
    onSelect: (item) => console.log("Select", item),
    onLoadMore: () => console.log("Load more"),
    onRetry: () => console.log("Retry"),
  },
} satisfies Meta<typeof PullRequestImportResults>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Recent: Story = {};
export const Loading: Story = { args: { items: [], isLoading: true } };
export const Empty: Story = { args: { items: [] } };
export const ConnectionError: Story = {
  args: { items: [], error: "Check the workspace repository connection and try again." },
};
export const ImportPending: Story = { args: { isPending: true } };
export const LoadingMore: Story = { args: { isFetching: true } };
export const Mobile: Story = {
  decorators: [
    (Story) => (
      <div className="w-80 [&_button]:min-h-11 [&_input]:h-11 [&_input]:text-base">
        <Story />
      </div>
    ),
  ],
};
