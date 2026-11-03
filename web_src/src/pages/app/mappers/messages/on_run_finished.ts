import { defaultTriggerRenderer } from "../default";
import type { TriggerRenderer } from "../types";

interface RunFinishedData {
  runId?: string;
  result?: string;
  entryNodeId?: string;
}

export const onRunFinishedTriggerRenderer: TriggerRenderer = {
  ...defaultTriggerRenderer,
  getTitleAndSubtitle: ({ event }) => {
    const data = event?.data as RunFinishedData | undefined;
    return { title: data?.result ? `Run ${data.result}` : "Run finished", subtitle: data?.entryNodeId ?? "" };
  },
  getRootEventValues: ({ event }) => {
    const data = event?.data as RunFinishedData | undefined;
    return { "Source run ID": data?.runId ?? "", Result: data?.result ?? "", "Entry node ID": data?.entryNodeId ?? "" };
  },
  getTriggerProps: (context) => {
    const props = defaultTriggerRenderer.getTriggerProps(context);
    const configuration = context.node.configuration as { sourceEntryNodeId?: string } | undefined;
    props.metadata = configuration?.sourceEntryNodeId ? [{ icon: "play", label: configuration.sourceEntryNodeId }] : [];
    if (props.lastEventData) {
      props.lastEventData.title = onRunFinishedTriggerRenderer.getTitleAndSubtitle({ event: context.lastEvent }).title;
    }
    return props;
  },
};
