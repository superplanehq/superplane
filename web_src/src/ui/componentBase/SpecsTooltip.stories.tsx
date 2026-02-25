import type { Meta, StoryObj } from "@storybook/react";
import { SpecsTooltip } from "./SpecsTooltip";
import { ComponentBaseSpecValue } from "./index";
import { ListFilter } from "lucide-react";

const mockSpecValuesSingle: ComponentBaseSpecValue[] = [
  {
    badges: [
      { label: "branch=main", bgColor: "bg-blue-500", textColor: "text-blue-700" },
      { label: "event=push", bgColor: "bg-green-500", textColor: "text-green-700" },
    ],
  },
];

const mockSpecValuesMultiple: ComponentBaseSpecValue[] = [
  {
    badges: [
      { label: "tag=latest", bgColor: "bg-purple-500", textColor: "text-purple-600" },
      { label: "push", bgColor: "bg-orange-500", textColor: "text-orange-600" },
    ],
  },
  {
    badges: [{ label: "tag=v1.0", bgColor: "bg-blue-500", textColor: "text-blue-500" }],
  },
  {
    badges: [
      { label: "branch=develop", bgColor: "bg-cyan-500", textColor: "text-cyan-700" },
      { label: "event=pull_request", bgColor: "bg-yellow-500", textColor: "text-yellow-700" },
      { label: "status=open", bgColor: "bg-red-500", textColor: "text-red-500" },
    ],
  },
];

const mockSpecValuesComplex: ComponentBaseSpecValue[] = [
  {
    badges: [
      { label: "env=production", bgColor: "bg-red-600", textColor: "text-red-700" },
      { label: "region=us-east-1", bgColor: "bg-blue-700", textColor: "text-blue-800" },
      { label: "tier=critical", bgColor: "bg-amber-800", textColor: "text-amber-900" },
    ],
  },
  {
    badges: [
      { label: "env=staging", bgColor: "bg-yellow-600", textColor: "text-yellow-700" },
      { label: "region=us-west-2", bgColor: "bg-blue-700", textColor: "text-blue-800" },
      { label: "tier=standard", bgColor: "bg-emerald-600", textColor: "text-emerald-700" },
    ],
  },
];

// Long filter value tests tooltip wrapping
const mockSpecValuesLongFilter: ComponentBaseSpecValue[] = [
  {
    badges: [
      {
        label:
          'let files = concat($.head_commit.added, $.head_commit.modified, $.head_commit.removed) | uniq(); $.ref == "refs/heads/main" and any(files, hasPrefix(#, "helm/staging"))',
        bgColor: "bg-purple-100",
        textColor: "text-purple-700",
      },
    ],
  },
];

// Mock badge component similar to what's in ComponentBase
const MockSpecsBadge: React.FC<{ count: number; title: string }> = ({ count, title }) => (
  <div className="flex items-center gap-3 text-md text-gray-500 cursor-pointer">
    <ListFilter size={18} />
    <span className="text-sm bg-gray-500 px-2 py-1 rounded-md text-white font-mono font-medium hover:bg-gray-600 transition-colors">
      {count} {title + (count > 1 ? "s" : "")}
    </span>
  </div>
);

const meta: Meta<typeof SpecsTooltip> = {
  title: "ui/ComponentBase/SpecsTooltip",
  component: SpecsTooltip,
  parameters: {
    layout: "centered",
  },
  tags: ["autodocs"],
  argTypes: {
    specTitle: {
      control: "text",
      description: 'The title of the specification type (e.g., "filter", "condition")',
    },
    specValues: {
      control: "object",
      description: "Array of specification values with badges",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const SingleFilter: Story = {
  args: {
    specTitle: "filter",
    specValues: mockSpecValuesSingle,
  },
  render: (args) => (
    <div className="p-8">
      <p className="mb-4 text-sm text-gray-500">Hover over the badge below to see the tooltip:</p>
      <SpecsTooltip {...args}>
        <div>
          <MockSpecsBadge count={args.specValues.length} title={args.specTitle} />
        </div>
      </SpecsTooltip>
    </div>
  ),
};

export const MultipleFilters: Story = {
  args: {
    specTitle: "filter",
    tooltipTitle: "filters applied",
    specValues: mockSpecValuesMultiple,
  },
  render: (args) => (
    <div className="p-8">
      <p className="mb-4 text-sm text-gray-500">Hover over the badge below to see the tooltip with multiple filters:</p>
      <SpecsTooltip {...args}>
        <div>
          <MockSpecsBadge count={args.specValues.length} title={args.specTitle} />
        </div>
      </SpecsTooltip>
    </div>
  ),
};

export const Conditions: Story = {
  args: {
    specTitle: "condition",
    specValues: mockSpecValuesComplex,
  },
  render: (args) => (
    <div className="p-8">
      <p className="mb-4 text-sm text-gray-500">Hover over the badge below to see conditions tooltip:</p>
      <SpecsTooltip {...args}>
        <div>
          <MockSpecsBadge count={args.specValues.length} title={args.specTitle} />
        </div>
      </SpecsTooltip>
    </div>
  ),
};

export const WithTextChild: Story = {
  args: {
    specTitle: "rule",
    specValues: mockSpecValuesSingle,
  },
  render: (args) => (
    <div className="p-8">
      <p className="mb-4 text-sm text-gray-500">Tooltip can wrap any child element:</p>
      <SpecsTooltip {...args}>
        <span className="text-blue-600 cursor-pointer underline">
          View {args.specValues.length} {args.specTitle}
        </span>
      </SpecsTooltip>
    </div>
  ),
};

export const LongFilter: Story = {
  args: {
    specTitle: "filter",
    tooltipTitle: "filters applied",
    specValues: mockSpecValuesLongFilter,
  },
  render: (args) => (
    <div className="p-8">
      <p className="mb-4 text-sm text-gray-500">
        Hover over the badge below to see how long filter values wrap in the tooltip:
      </p>
      <SpecsTooltip {...args}>
        <div>
          <MockSpecsBadge count={args.specValues.length} title={args.specTitle} />
        </div>
      </SpecsTooltip>
    </div>
  ),
};
