import { PresetPropertyFn, Options, CoreConfig, StorybookConfig, Indexer, CLIOptions } from '@storybook/types';
import { Channel } from '@storybook/channels';

declare const staticDirs: PresetPropertyFn<'staticDirs'>;
declare const favicon: (value: string | undefined, options: Pick<Options, 'presets' | 'configDir' | 'staticDir'>) => Promise<string>;
declare const babel: (_: unknown, options: Options) => Promise<{}>;
declare const title: (previous: string, options: Options) => string | false;
declare const logLevel: (previous: any, options: Options) => any;
declare const previewHead: (base: any, { configDir, presets }: Options) => Promise<string>;
declare const env: () => Promise<Record<string, string>>;
declare const previewBody: (base: any, { configDir, presets }: Options) => Promise<string>;
declare const typescript: () => {
    check: boolean;
    reactDocgen: string;
    reactDocgenTypescriptOptions: {
        shouldExtractLiteralValuesFromEnum: boolean;
        shouldRemoveUndefinedFromOptional: boolean;
        propFilter: (prop: any) => boolean;
        savePropValueAsString: boolean;
    };
};
/**
 * If for some reason this config is not applied, the reason is that
 * likely there is an addon that does `export core = () => ({ someConfig })`,
 * instead of `export core = (existing) => ({ ...existing, someConfig })`,
 * just overwriting everything and not merging with the existing values.
 */
declare const core: (existing: CoreConfig, options: Options) => Promise<CoreConfig>;
declare const previewAnnotations: (base: any, options: Options) => Promise<any[]>;
declare const features: (existing: StorybookConfig['features']) => Promise<StorybookConfig['features']>;
declare const csfIndexer: Indexer;
declare const experimental_indexers: StorybookConfig['experimental_indexers'];
declare const frameworkOptions: (_: never, options: Options) => Promise<Record<string, any> | null>;
declare const docs: (docsOptions: StorybookConfig['docs'], { docs: docsMode }: CLIOptions) => StorybookConfig['docs'];
declare const managerHead: (_: any, options: Options) => Promise<string>;
type OptionsWithRequiredCache = Exclude<Options, 'cache'> & Required<Pick<Options, 'cache'>>;
declare const experimental_serverChannel: (channel: Channel, options: OptionsWithRequiredCache) => Promise<Channel>;

export { babel, core, csfIndexer, docs, env, experimental_indexers, experimental_serverChannel, favicon, features, frameworkOptions, logLevel, managerHead, previewAnnotations, previewBody, previewHead, staticDirs, title, typescript };
