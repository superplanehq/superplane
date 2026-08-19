import type { Meta, StoryObj } from "@storybook/react-vite";

import { subHourVelocityDurationFormat } from "./velocityDurationFormat";

/**
 * Support story for the sub-hour-aware duration formatter used by the
 * Velocity work-order time scorecards and chart (see
 * `VelocityWorkOrderFlowCard.stories.tsx` for it in context). Not the review
 * home — see Factories/Pages/Velocity → Default for the real screen.
 */
const FORMATTER_EXAMPLES: { case: string; hours: number }[] = [
  { case: "Sub-hour", hours: 0.2 },
  { case: "In range", hours: 5 },
  { case: "Multi-day", hours: 60 },
  { case: "Genuine zero", hours: 0 },
];

function VelocityDurationFormatMatrix() {
  return (
    <table className="w-full max-w-sm border-collapse text-[13px]" data-testid="velocity-duration-formatter-matrix">
      <thead>
        <tr className="text-left text-muted-foreground">
          <th className="border-b border-border pb-2 pr-6 font-medium">Case</th>
          <th className="border-b border-border pb-2 pr-6 font-medium">Hours in</th>
          <th className="border-b border-border pb-2 font-medium">Label out</th>
        </tr>
      </thead>
      <tbody>
        {FORMATTER_EXAMPLES.map((example) => (
          <tr key={example.case}>
            <td className="border-b border-border py-2 pr-6 text-foreground">{example.case}</td>
            <td className="border-b border-border py-2 pr-6 tabular-nums text-muted-foreground">{example.hours}</td>
            <td className="border-b border-border py-2 font-mono tabular-nums text-foreground">
              {subHourVelocityDurationFormat.formatDuration(example.hours)}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

const meta = {
  title: "Factories/Components/Velocity Duration Format",
  component: VelocityDurationFormatMatrix,
} satisfies Meta<typeof VelocityDurationFormatMatrix>;

export default meta;

type Story = StoryObj<typeof meta>;

export const FormatterMatrix: Story = {};
