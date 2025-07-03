import { loadAllPresets } from '@storybook/core-common';
export { getPreviewBodyTemplate, getPreviewHeadTemplate } from '@storybook/core-common';
import { CLIOptions, LoadOptions, BuilderOptions } from '@storybook/types';
import { EventType } from '@storybook/telemetry';

type BuildStaticStandaloneOptions = CLIOptions & LoadOptions & BuilderOptions & {
    outputDir: string;
};
declare function buildStaticStandalone(options: BuildStaticStandaloneOptions): Promise<void>;

declare function buildDevStandalone(options: CLIOptions & LoadOptions & BuilderOptions): Promise<{
    port: number;
    address: string;
    networkAddress: string;
}>;

type TelemetryOptions = {
    cliOptions: CLIOptions;
    presetOptions?: Parameters<typeof loadAllPresets>[0];
    printError?: (err: any) => void;
    skipPrompt?: boolean;
};
type ErrorLevel = 'none' | 'error' | 'full';
declare function getErrorLevel({ cliOptions, presetOptions, skipPrompt, }: TelemetryOptions): Promise<ErrorLevel>;
declare function sendTelemetryError(_error: unknown, eventType: EventType, options: TelemetryOptions): Promise<void>;
declare function withTelemetry<T>(eventType: EventType, options: TelemetryOptions, run: () => Promise<T>): Promise<T | undefined>;

declare function build(options?: any, frameworkOptions?: any): Promise<void | {
    port: number;
    address: string;
    networkAddress: string;
}>;

export { BuildStaticStandaloneOptions, build, buildDevStandalone, buildStaticStandalone, getErrorLevel, sendTelemetryError, withTelemetry };
