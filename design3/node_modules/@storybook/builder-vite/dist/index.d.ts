import { Builder, Options, StorybookConfig } from '@storybook/types';
import { UserConfig, InlineConfig, PluginOption } from 'vite';

type ViteStats = {
    toJson: () => any;
};
type ViteBuilder = Builder<UserConfig, ViteStats>;
type ViteFinal = (config: InlineConfig, options: Options) => InlineConfig | Promise<InlineConfig>;
type StorybookConfigVite = {
    viteFinal?: ViteFinal;
};
type BuilderOptions = {
    /**
     * Path to vite.config file, relative to CWD.
     */
    viteConfigPath?: string;
};

/**
 * Recursively removes all plugins with the names given
 * Resolves async plugins
 */
declare const withoutVitePlugins: (plugins: PluginOption[] | undefined, namesToRemove: string[]) => Promise<PluginOption[]>;

/**
 * Returns true if ANY of the plugins in the array have a name that matches one of the names in the names array.
 * Will resolve any promises in the array.
 */
declare function hasVitePlugins(plugins: PluginOption[], names: string[]): Promise<boolean>;

/**
 * @deprecated
 *
 * Import `StorybookConfig` from your framework, such as:
 *
 * `import type { StorybookConfig } from '@storybook/react-vite';`
 */
type StorybookViteConfig = StorybookConfig & StorybookConfigVite;
declare function bail(): Promise<void>;
declare const start: ViteBuilder['start'];
declare const build: ViteBuilder['build'];

export { BuilderOptions, StorybookConfigVite, StorybookViteConfig, ViteBuilder, ViteFinal, bail, build, hasVitePlugins, start, withoutVitePlugins };
