import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { Toaster } from "sonner";
import { Separator } from "@/components/ui/separator";
import { StartRunForm } from "./StartRunForm";
import { StartRunSummaryTable } from "./StartRunSummaryTable";
import { issueExamplePayload, parseParams, type ParamDefinition } from "./paramSyntax";

function StartRunFormPlayground({
  defs,
  nodeName = "Start",
  templateName = "Deploy machine",
}: {
  defs: ParamDefinition[];
  nodeName?: string;
  templateName?: string;
}) {
  const [submittedParams, setSubmittedParams] = useState<Record<string, unknown> | null>(null);

  return (
    <div className="w-[28rem] space-y-4 rounded-lg border border-slate-200 bg-background p-4 shadow-sm">
      <StartRunSummaryTable nodeName={nodeName} templateName={templateName} />
      <Separator />
      <StartRunForm
        defs={defs}
        onClose={() => {
          console.log("StartRunForm: close");
        }}
        onRun={async (params) => {
          console.log("StartRunForm: run", params);
          setSubmittedParams(params);
        }}
      />
      {submittedParams ? (
        <div className="rounded-md border border-slate-200 bg-slate-50 p-3">
          <p className="mb-2 text-[10px] font-semibold uppercase tracking-wide text-slate-500">Submitted params</p>
          <pre className="overflow-x-auto font-mono text-[11px] leading-snug text-slate-700">
            {JSON.stringify(submittedParams, null, 2)}
          </pre>
        </div>
      ) : null}
    </div>
  );
}

const meta = {
  title: "workflowv2/start/StartRunForm",
  component: StartRunForm,
  parameters: {
    layout: "padded",
  },
  decorators: [
    (Story) => (
      <>
        <Story />
        <Toaster position="bottom-center" closeButton />
      </>
    ),
  ],
} satisfies Meta<typeof StartRunForm>;

export default meta;

type Story = StoryObj<typeof meta>;

export const IssueExample: Story = {
  render: () => <StartRunFormPlayground defs={parseParams(issueExamplePayload())} />,
};

export const RequiredSelectOnly: Story = {
  render: () => (
    <StartRunFormPlayground
      defs={parseParams({
        size: "param(type:select, values:'2 vCPU|4 vCPU|8 vCPU', title:'Select size', required:true)",
      })}
      templateName="Scale up"
    />
  ),
};
