import { PresetProperty, StorybookConfig, Options } from '@storybook/types';

declare const framework: PresetProperty<'framework', StorybookConfig>;
declare const stories: PresetProperty<'stories', StorybookConfig>;
declare const typescript: PresetProperty<'typescript', StorybookConfig>;
declare const docs: PresetProperty<'docs', StorybookConfig>;
declare const build: (value: StorybookConfig['build'], options: Options) => Promise<{
    test: {
        disableBlocks: boolean;
        disabledAddons: string[];
        disableMDXEntries: boolean;
        disableAutoDocs: boolean;
        disableDocgen: boolean;
        disableSourcemaps: boolean;
        disableTreeShaking: boolean;
        esbuildMinify: boolean;
    };
}>;

export { build, docs, framework, stories, typescript };
