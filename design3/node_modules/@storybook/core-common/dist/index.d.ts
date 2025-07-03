import { PresetConfig, CoreCommon_ResolvedAddonPreset, CoreCommon_ResolvedAddonVirtual, LoadedPreset, Presets, CLIOptions, LoadOptions, BuilderOptions, StorybookConfig, CoreCommon_AddonInfo, Options as Options$2, PackageJson, CoreCommon_StorybookInfo, Ref, StoriesEntry, NormalizedStoriesSpecifier } from '@storybook/types';
import * as Cache from 'file-system-cache';
import Cache__default from 'file-system-cache';

type InterPresetOptions = Omit<CLIOptions & LoadOptions & BuilderOptions & {
    isCritical?: boolean;
    build?: StorybookConfig['build'];
}, 'frameworkPresets'>;
declare function filterPresetsConfig(presetsConfig: PresetConfig[]): PresetConfig[];
/**
 * Parse an addon into either a managerEntries or a preset. Throw on invalid input.
 *
 * Valid inputs:
 * - '@storybook/addon-actions/manager'
 *   =>  { type: 'virtual', item }
 *
 * - '@storybook/addon-docs/preset'
 *   =>  { type: 'presets', item }
 *
 * - '@storybook/addon-docs'
 *   =>  { type: 'presets', item: '@storybook/addon-docs/preset' }
 *
 * - { name: '@storybook/addon-docs(/preset)?', options: { ... } }
 *   =>  { type: 'presets', item: { name: '@storybook/addon-docs/preset', options } }
 */
declare const resolveAddonName: (configDir: string, name: string, options: any) => CoreCommon_ResolvedAddonPreset | CoreCommon_ResolvedAddonVirtual | undefined;
declare function loadPreset(input: PresetConfig, level: number, storybookOptions: InterPresetOptions): Promise<LoadedPreset[]>;
declare function getPresets(presets: PresetConfig[], storybookOptions: InterPresetOptions): Promise<Presets>;
declare function loadAllPresets(options: CLIOptions & LoadOptions & BuilderOptions & {
    corePresets: PresetConfig[];
    overridePresets: PresetConfig[];
    /** Whether preset failures should be critical or not */
    isCritical?: boolean;
    build?: StorybookConfig['build'];
}): Promise<Presets>;

declare const cache: Cache.FileSystemCache;

interface Options$1 {
    before: CoreCommon_AddonInfo;
    after: CoreCommon_AddonInfo;
    configFile: string;
    getConfig: (path: string) => any;
}
declare const checkAddonOrder: ({ before, after, configFile, getConfig }: Options$1) => Promise<void>;

declare function loadEnvs(options?: {
    production?: boolean;
}): {
    stringified: Record<string, string>;
    raw: Record<string, string>;
};
declare const stringifyEnvs: (raw: Record<string, string>) => Record<string, string>;
declare const stringifyProcessEnvs: (raw: Record<string, string>) => Record<string, string>;

declare const findDistEsm: (cwd: string, relativePath: string) => string;

declare const commonGlobOptions: (glob: string) => {
    ignore?: undefined;
} | {
    ignore: string[];
};

/**
 * Builder options can be specified in `core.builder.options` or `framework.options.builder`.
 * Preference is given here to `framework.options.builder` if both are specified.
 */
declare function getBuilderOptions<T extends Record<string, any>>(options: Options$2): Promise<T | Record<string, never>>;

/**
 * Framework can be a string or an object.  This utility always returns the string name.
 */
declare function getFrameworkName(options: Options$2): Promise<string>;

/**
 * Render is set as a string on core. It must be set by the framework
 */
declare function getRendererName(options: Options$2): Promise<string>;

declare function getStorybookConfiguration(storybookScript: string, shortName: string, longName: string): string | null;

declare const rendererPackages: Record<string, string>;
declare const frameworkPackages: Record<string, string>;
declare const builderPackages: string[];
declare const findConfigFile: (prefix: string, configDir: string) => string | null;
declare const getStorybookInfo: (packageJson: PackageJson, configDir?: string) => CoreCommon_StorybookInfo;

declare const getAutoRefs: (options: Options$2) => Promise<Record<string, Ref>>;
declare function getRefs(options: Options$2): Promise<Record<string, Ref>>;

declare function globToRegexp(glob: string): RegExp;

declare function handlebars(source: string, data: any): string;

/**
 * Return a string corresponding to template filled with bindings using following pattern:
 * For each (key, value) of `bindings` replace, in template, `{{key}}` by escaped version of `value`
 *
 * @param template {String} Template with `{{binding}}`
 * @param bindings {Object} key-value object use to fill the template, `{{key}}` will be replaced by `escaped(value)`
 * @returns {String} Filled template
 */
declare const interpolate: (template: string, bindings: Record<string, string>) => string;

declare const boost: Set<string>;
declare function getInterpretedFile(pathToFile: string): string | undefined;
declare function getInterpretedFileWithExt(pathToFile: string): {
    path: string;
    ext: string;
} | undefined;

declare function interopRequireDefault(filePath: string): any;
declare function serverRequire(filePath: string | string[]): any;
declare function serverResolve(filePath: string | string[]): string | null;

declare function loadCustomPresets({ configDir }: {
    configDir: string;
}): PresetConfig[];

declare function loadMainConfig({ configDir, noCache, }: {
    configDir: string;
    noCache?: boolean;
}): Promise<StorybookConfig>;

declare function loadManagerOrAddonsFile({ configDir }: {
    configDir: string;
}): string | undefined;

declare function loadPreviewOrConfigFile({ configDir }: {
    configDir: string;
}): string | undefined;

declare function logConfig(caption: unknown, config: unknown): void;

declare const getDirectoryFromWorkingDir: ({ configDir, workingDir, directory, }: NormalizeOptions & {
    directory: string;
}) => string;
declare const normalizeStoriesEntry: (entry: StoriesEntry, { configDir, workingDir, defaultFilesPattern }: NormalizeOptions) => NormalizedStoriesSpecifier;
interface NormalizeOptions {
    configDir: string;
    workingDir: string;
    defaultFilesPattern?: string;
}
declare const normalizeStories: (entries: StoriesEntry[], options: NormalizeOptions) => NormalizedStoriesSpecifier[];

declare const getProjectRoot: () => string;
declare const nodePathsToArray: (nodePath: string) => string[];
/**
 * Ensures that a path starts with `./` or `../`, or is entirely `.` or `..`
 */
declare function normalizeStoryPath(filename: string): string;

declare function readTemplate(filename: string): Promise<string>;

/**
 * Get the path of the file or directory with input name inside the Storybook cache directory:
 *  - `node_modules/.cache/storybook/{directoryName}` in a Node.js project or npm package
 *  - `.cache/storybook/{directoryName}` otherwise
 *
 * @param fileOrDirectoryName {string} Name of the file or directory
 * @return {string} Absolute path to the file or directory
 */
declare function resolvePathInStorybookCache(fileOrDirectoryName: string): string;

declare function isPreservingSymlinks(): boolean | undefined;

declare function getPreviewBodyTemplate(configDirPath: string, interpolations?: Record<string, string>): string;
declare function getPreviewHeadTemplate(configDirPath: string, interpolations?: Record<string, string>): string;

declare function validateFrameworkName(frameworkName: string | undefined): asserts frameworkName is string;

declare function validateConfigurationFiles(configDir: string): Promise<void>;

/**
 * Mimicking the satisfies operator until we can upgrade to TS4.9
 */
declare function satisfies<A>(): <T extends A>(x: T) => T;

declare function stripAbsNodeModulesPath(absPath: string): string;

type Options = Parameters<typeof Cache__default>['0'];
type FileSystemCache = ReturnType<typeof Cache__default>;
declare function createFileSystemCache(options: Options): FileSystemCache;

export { boost, builderPackages, cache, checkAddonOrder, commonGlobOptions, createFileSystemCache, filterPresetsConfig, findConfigFile, findDistEsm, frameworkPackages, getAutoRefs, getBuilderOptions, getDirectoryFromWorkingDir, getFrameworkName, getInterpretedFile, getInterpretedFileWithExt, getPresets, getPreviewBodyTemplate, getPreviewHeadTemplate, getProjectRoot, getRefs, getRendererName, getStorybookConfiguration, getStorybookInfo, globToRegexp, handlebars, interopRequireDefault, interpolate, isPreservingSymlinks, loadAllPresets, loadCustomPresets, loadEnvs, loadMainConfig, loadManagerOrAddonsFile, loadPreset, loadPreviewOrConfigFile, logConfig, nodePathsToArray, normalizeStories, normalizeStoriesEntry, normalizeStoryPath, readTemplate, rendererPackages, resolveAddonName, resolvePathInStorybookCache, satisfies, serverRequire, serverResolve, stringifyEnvs, stringifyProcessEnvs, stripAbsNodeModulesPath, validateConfigurationFiles, validateFrameworkName };
