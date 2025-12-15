import type { Meta, StoryObj } from "@storybook/react";
import { EmptyState } from "./";
import { Clock, Database, FileX, Search, Users } from "lucide-react";

const meta: Meta<typeof EmptyState> = {
  title: "ui/EmptyState",
  component: EmptyState,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    icon: {
      control: false,
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
};

export const WithDescription: Story = {
  args: {
    title: "No data available",
    description: "There's nothing to show here yet. Try creating some content to get started.",
  },
};

export const CustomIcon: Story = {
  args: {
    icon: Database,
    title: "No database connections",
    description: "Connect a database to start managing your data.",
  },
};

export const SearchEmpty: Story = {
  args: {
    icon: Search,
    title: "No results found",
    description: "Try adjusting your search criteria or browse all available items.",
  },
};

export const UsersEmpty: Story = {
  args: {
    icon: Users,
    title: "No team members",
    description: "Invite team members to collaborate on this project.",
  },
};

export const FilesEmpty: Story = {
  args: {
    icon: FileX,
    title: "No files uploaded",
    description: "Drag and drop files here or click to browse and upload.",
  },
};

export const WaitingState: Story = {
  args: {
    icon: Clock,
    title: "Processing...",
    description: "Please wait while we process your request. This may take a few moments.",
  },
};

export const ComponentBaseExample: Story = {
  args: {
    title: "Waiting for the first run",
  },
  render: (args) => (
    <div className="w-[23rem] bg-white border rounded-md">
      <div className="px-4 py-3 bg-gray-50 border-b">
        <h3 className="font-semibold">Component Example</h3>
      </div>
      <EmptyState {...args} />
    </div>
  ),
};
