import { ComponentBaseProps } from "@/ui/componentBase";
import {
  ComponentBaseContext,
  ComponentBaseMapper,
  ExecutionDetailsContext,
  OutputPayload,
  SubtitleContext,
} from "../types";
import { baseProps } from "./base";
import { buildGitlabExecutionSubtitle } from "./utils";

interface PipelineOutput {
  id?: number;
  iid?: number;
  status?: string;
  ref?: string;
  sha?: string;
  web_url?: string;
  url?: string;
}

interface TestReportSummaryOutput {
  total?: {
    count?: number;
    success?: number;
    failed?: number;
    skipped?: number;
    error?: number;
    time?: number;
  };
  test_suites?: Array<{
    name?: string;
    total_count?: number;
    success_count?: number;
    failed_count?: number;
    skipped_count?: number;
    error_count?: number;
  }>;
}

function getOutputData(context: { execution: { outputs?: unknown } }): unknown {
  const outputs = context.execution.outputs as { default?: OutputPayload[] } | undefined;
  return outputs?.default?.[0]?.data;
}

export const pipelineLookupMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    const pipeline = getOutputData(context) as PipelineOutput | undefined;
    if (pipeline?.status) {
      return buildGitlabExecutionSubtitle(context.execution, `Pipeline ${pipeline.status}`);
    }
    return buildGitlabExecutionSubtitle(context.execution, "Pipeline Retrieved");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const pipeline = getOutputData(context) as PipelineOutput | undefined;
    const details: Record<string, string> = {};

    if (!pipeline) {
      return details;
    }

    if (pipeline.id) details["Pipeline ID"] = pipeline.id.toString();
    if (pipeline.iid) details["Pipeline IID"] = pipeline.iid.toString();
    if (pipeline.status) details["Status"] = pipeline.status;
    if (pipeline.ref) details["Ref"] = pipeline.ref;
    if (pipeline.sha) details["SHA"] = pipeline.sha;
    if (pipeline.web_url || pipeline.url) details["Pipeline URL"] = pipeline.web_url || pipeline.url || "";

    return details;
  },
};

export const testReportSummaryMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    return baseProps(context.nodes, context.node, context.componentDefinition, context.lastExecutions);
  },

  subtitle(context: SubtitleContext): string {
    const summary = getOutputData(context) as TestReportSummaryOutput | undefined;
    const failed = summary?.total?.failed;
    if (failed !== undefined) {
      return buildGitlabExecutionSubtitle(context.execution, `${failed} failed tests`);
    }
    return buildGitlabExecutionSubtitle(context.execution, "Test Report Retrieved");
  },

  getExecutionDetails(context: ExecutionDetailsContext): Record<string, string> {
    const summary = getOutputData(context) as TestReportSummaryOutput | undefined;
    const details: Record<string, string> = {};
    const total = summary?.total;

    if (!total) {
      return details;
    }

    if (total.count !== undefined) details["Total Tests"] = total.count.toString();
    if (total.success !== undefined) details["Passed Tests"] = total.success.toString();
    if (total.failed !== undefined) details["Failed Tests"] = total.failed.toString();
    if (total.skipped !== undefined) details["Skipped Tests"] = total.skipped.toString();
    if (total.error !== undefined) details["Errored Tests"] = total.error.toString();
    if (total.time !== undefined) details["Total Time (s)"] = total.time.toString();
    if (summary?.test_suites) details["Test Suites"] = summary.test_suites.length.toString();

    return details;
  },
};
