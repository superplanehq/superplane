import { ComponentBaseProps } from "@/ui/componentBase";
import { MetadataItem } from "@/ui/metadataList";
import { ComponentBaseContext, ComponentBaseMapper, NodeInfo } from "../types";
import { cloudBuildBaseMapper } from "./base";

export const runTriggerMapper: ComponentBaseMapper = {
  ...cloudBuildBaseMapper,
  props(context: ComponentBaseContext): ComponentBaseProps {
    const baseProps = cloudBuildBaseMapper.props(context);
    return {
      ...baseProps,
      metadata: runTriggerMetadataList(context.node),
    };
  },
};

function runTriggerMetadataList(node: NodeInfo): MetadataItem[] {
  const metadata: MetadataItem[] = [];
  const configuration = node.configuration as any;
  const nodeMetadata = node.metadata as any;

  if (nodeMetadata?.triggerName) {
    metadata.push({ icon: "zap", label: nodeMetadata.triggerName });
  }

  if (configuration?.ref) {
    metadata.push({ icon: "git-branch", label: shortRef(configuration.ref) });
  }

  return metadata;
}

function shortRef(ref: string): string {
  if (ref.startsWith("refs/heads/")) return ref.slice("refs/heads/".length);
  if (ref.startsWith("refs/tags/")) return ref.slice("refs/tags/".length);
  return ref;
}
