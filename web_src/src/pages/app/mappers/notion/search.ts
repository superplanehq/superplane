import type { MetadataItem } from "@/ui/metadataList";
import { buildSubtitle } from "../eventDisplay";
import type { ComponentBaseMapper, SubtitleContext } from "../types";
import { notionComponentProps, notionExecutionDetails, notionOutput } from "./base";
import type { SearchConfiguration, SearchResult } from "./types";

export const searchMapper: ComponentBaseMapper = {
  props(context) {
    const configuration = context.node.configuration as SearchConfiguration | undefined;
    const metadata: MetadataItem[] = [{ icon: "search", label: configuration?.query || "All titles" }];
    if (configuration?.objectType === "page") {
      metadata.push({ icon: "file-text", label: "Pages" });
    }
    if (configuration?.objectType === "data_source") {
      metadata.push({ icon: "database", label: "Data sources" });
    }
    return notionComponentProps(context, metadata, searchSubtitle);
  },

  getExecutionDetails({ execution }) {
    const details = notionExecutionDetails(execution);
    const result = notionOutput<SearchResult>(execution);
    if (Array.isArray(result?.results)) details["Results"] = String(result.results.length);
    if (typeof result?.has_more === "boolean") details["More results"] = result.has_more ? "Yes" : "No";
    return details;
  },

  subtitle: searchSubtitle,
};

function searchSubtitle({ execution }: SubtitleContext) {
  const result = notionOutput<SearchResult>(execution);
  const count = Array.isArray(result?.results) ? result.results.length : undefined;
  const label = count === undefined ? undefined : `${count} ${count === 1 ? "result" : "results"}`;
  return buildSubtitle(label, execution.createdAt);
}
