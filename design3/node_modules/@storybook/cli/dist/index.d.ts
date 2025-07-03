import { CommonOptions } from 'execa';
import { PackageJson } from '@storybook/types';
export { PackageJson } from '@storybook/types';
import { WriteStream } from 'fs-extra';

declare const _default: {
    '@storybook/addon-a11y': string;
    '@storybook/addon-actions': string;
    '@storybook/addon-backgrounds': string;
    '@storybook/addon-controls': string;
    '@storybook/addon-docs': string;
    '@storybook/addon-essentials': string;
    '@storybook/addon-highlight': string;
    '@storybook/addon-interactions': string;
    '@storybook/addon-jest': string;
    '@storybook/addon-links': string;
    '@storybook/addon-mdx-gfm': string;
    '@storybook/addon-measure': string;
    '@storybook/addon-outline': string;
    '@storybook/addon-themes': string;
    '@storybook/addon-storyshots': string;
    '@storybook/addon-storyshots-puppeteer': string;
    '@storybook/addon-storysource': string;
    '@storybook/addon-toolbars': string;
    '@storybook/addon-viewport': string;
    '@storybook/addons': string;
    '@storybook/angular': string;
    '@storybook/api': string;
    '@storybook/blocks': string;
    '@storybook/builder-manager': string;
    '@storybook/builder-vite': string;
    '@storybook/builder-webpack5': string;
    '@storybook/channel-postmessage': string;
    '@storybook/channel-websocket': string;
    '@storybook/channels': string;
    '@storybook/cli': string;
    '@storybook/client-api': string;
    '@storybook/client-logger': string;
    '@storybook/codemod': string;
    '@storybook/components': string;
    '@storybook/core-client': string;
    '@storybook/core-common': string;
    '@storybook/core-events': string;
    '@storybook/core-server': string;
    '@storybook/core-webpack': string;
    '@storybook/csf-plugin': string;
    '@storybook/csf-tools': string;
    '@storybook/docs-tools': string;
    '@storybook/ember': string;
    '@storybook/html': string;
    '@storybook/html-vite': string;
    '@storybook/html-webpack5': string;
    '@storybook/instrumenter': string;
    '@storybook/manager': string;
    '@storybook/manager-api': string;
    '@storybook/nextjs': string;
    '@storybook/node-logger': string;
    '@storybook/postinstall': string;
    '@storybook/preact': string;
    '@storybook/preact-vite': string;
    '@storybook/preact-webpack5': string;
    '@storybook/preset-create-react-app': string;
    '@storybook/preset-html-webpack': string;
    '@storybook/preset-preact-webpack': string;
    '@storybook/preset-react-webpack': string;
    '@storybook/preset-server-webpack': string;
    '@storybook/preset-svelte-webpack': string;
    '@storybook/preset-vue-webpack': string;
    '@storybook/preset-vue3-webpack': string;
    '@storybook/preset-web-components-webpack': string;
    '@storybook/preview': string;
    '@storybook/preview-api': string;
    '@storybook/preview-web': string;
    '@storybook/react': string;
    '@storybook/react-dom-shim': string;
    '@storybook/react-vite': string;
    '@storybook/react-webpack5': string;
    '@storybook/router': string;
    '@storybook/server': string;
    '@storybook/server-webpack5': string;
    '@storybook/source-loader': string;
    '@storybook/store': string;
    '@storybook/svelte': string;
    '@storybook/svelte-vite': string;
    '@storybook/svelte-webpack5': string;
    '@storybook/sveltekit': string;
    '@storybook/telemetry': string;
    '@storybook/theming': string;
    '@storybook/types': string;
    '@storybook/test': string;
    '@storybook/vue': string;
    '@storybook/vue-vite': string;
    '@storybook/vue-webpack5': string;
    '@storybook/vue3': string;
    '@storybook/vue3-vite': string;
    '@storybook/vue3-webpack5': string;
    '@storybook/web-components': string;
    '@storybook/web-components-vite': string;
    '@storybook/web-components-webpack5': string;
    sb: string;
    storybook: string;
};

declare const useNpmWarning: () => void;

type PackageJsonWithDepsAndDevDeps = PackageJson & Required<Pick<PackageJson, 'dependencies' | 'devDependencies'>>;
type PackageJsonWithMaybeDeps = Partial<Pick<PackageJson, 'dependencies' | 'devDependencies' | 'peerDependencies' | 'files'>>;

type PackageMetadata = {
    version: string;
    location?: string;
    reasons?: string[];
};
type InstallationMetadata = {
    dependencies: Record<string, PackageMetadata[]>;
    duplicatedDependencies: Record<string, string[]>;
    infoCommand: string;
    dedupeCommand: string;
};

type PackageManagerName = 'npm' | 'yarn1' | 'yarn2' | 'pnpm';
/**
 * Extract package name and version from input
 *
 * @param pkg A string like `@storybook/cli`, `react` or `react@^16`
 * @return A tuple of 2 elements: [packageName, packageVersion]
 */
declare function getPackageDetails(pkg: string): [string, string?];
interface JsPackageManagerOptions {
    cwd?: string;
}
declare abstract class JsPackageManager {
    abstract readonly type: PackageManagerName;
    abstract initPackageJson(): Promise<void>;
    abstract getRunStorybookCommand(): string;
    abstract getRunCommand(command: string): string;
    readonly cwd?: string;
    abstract getPackageJSON(packageName: string, basePath?: string): Promise<PackageJson | null>;
    abstract getPackageVersion(packageName: string, basePath?: string): Promise<string | null>;
    setRegistryURL(url: string): Promise<void>;
    getRegistryURL(): Promise<string>;
    constructor(options?: JsPackageManagerOptions);
    /** Detect whether Storybook gets initialized in a monorepository/workspace environment
     * The cwd doesn't have to be the root of the monorepo, it can be a subdirectory
     * @returns true, if Storybook is initialized inside a monorepository/workspace
     */
    isStorybookInMonorepo(): boolean;
    /**
     * Install dependencies listed in `package.json`
     */
    installDependencies(): Promise<void>;
    packageJsonPath(): string;
    readPackageJson(): Promise<PackageJson>;
    writePackageJson(packageJson: PackageJson): Promise<void>;
    /**
     * Read the `package.json` file available in the directory the command was call from
     * If there is no `package.json` it will create one.
     */
    retrievePackageJson(): Promise<PackageJsonWithDepsAndDevDeps>;
    getAllDependencies(): Promise<Record<string, string>>;
    /**
     * Add dependencies to a project using `yarn add` or `npm install`.
     *
     * @param {Object} options contains `skipInstall`, `packageJson` and `installAsDevDependencies` which we use to determine how we install packages.
     * @param {Array} dependencies contains a list of packages to add.
     * @example
     * addDependencies(options, [
     *   `@storybook/react@${storybookVersion}`,
     *   `@storybook/addon-actions@${actionsVersion}`,
     *   `@storybook/addon-links@${linksVersion}`,
     *   `@storybook/preview-api@${addonsVersion}`,
     * ]);
     */
    addDependencies(options: {
        skipInstall?: boolean;
        installAsDevDependencies?: boolean;
        packageJson?: PackageJson;
    }, dependencies: string[]): Promise<void>;
    /**
     * Remove dependencies from a project using `yarn remove` or `npm uninstall`.
     *
     * @param {Object} options contains `skipInstall`, `packageJson` and `installAsDevDependencies` which we use to determine how we install packages.
     * @param {Array} dependencies contains a list of packages to remove.
     * @example
     * removeDependencies(options, [
     *   `@storybook/react`,
     *   `@storybook/addon-actions`,
     * ]);
     */
    removeDependencies(options: {
        skipInstall?: boolean;
        packageJson?: PackageJson;
    }, dependencies: string[]): Promise<void>;
    /**
     * Return an array of strings matching following format: `<package_name>@<package_latest_version>`
     *
     * For packages in the storybook monorepo, when the latest version is equal to the version of the current CLI
     * the version is not added to the string.
     *
     * When a package is in the monorepo, and the version is not equal to the CLI version, the version is taken from the versions.ts file and added to the string.
     *
     * @param packages
     */
    getVersionedPackages(packages: string[]): Promise<string[]>;
    /**
     * Return an array of string standing for the latest version of the input packages.
     * To be able to identify which version goes with which package the order of the input array is keep.
     *
     * @param packageNames
     */
    getVersions(...packageNames: string[]): Promise<string[]>;
    /**
     * Return the latest version of the input package available on npmjs registry.
     * If constraint are provided it return the latest version matching the constraints.
     *
     * For `@storybook/*` packages the latest version is retrieved from `cli/src/versions.json` file directly
     *
     * @param packageName The name of the package
     * @param constraint A valid semver constraint, example: '1.x || >=2.5.0 || 5.0.0 - 7.2.3'
     */
    getVersion(packageName: string, constraint?: string): Promise<string>;
    /**
     * Get the latest version of the package available on npmjs.com.
     * If constraint is set then it returns a version satisfying it, otherwise the latest version available is returned.
     *
     * @param packageName Name of the package
     * @param constraint Version range to use to constraint the returned version
     */
    latestVersion(packageName: string, constraint?: string): Promise<string>;
    addStorybookCommandInScripts(options?: {
        port: number;
        preCommand?: string;
    }): Promise<void>;
    addScripts(scripts: Record<string, string>): Promise<void>;
    addPackageResolutions(versions: Record<string, string>): Promise<void>;
    protected abstract runInstall(): Promise<void>;
    protected abstract runAddDeps(dependencies: string[], installAsDevDependencies: boolean): Promise<void>;
    protected abstract runRemoveDeps(dependencies: string[]): Promise<void>;
    protected abstract getResolutions(packageJson: PackageJson, versions: Record<string, string>): Record<string, any>;
    /**
     * Get the latest or all versions of the input package available on npmjs.com
     *
     * @param packageName Name of the package
     * @param fetchAllVersions Should return
     */
    protected abstract runGetVersions<T extends boolean>(packageName: string, fetchAllVersions: T): Promise<T extends true ? string[] : string>;
    abstract runPackageCommand(command: string, args: string[], cwd?: string, stdio?: string): Promise<string>;
    abstract runPackageCommandSync(command: string, args: string[], cwd?: string, stdio?: 'inherit' | 'pipe'): string;
    abstract findInstallations(pattern?: string[]): Promise<InstallationMetadata | undefined>;
    abstract parseErrorFromLogs(logs?: string): string;
    executeCommandSync({ command, args, stdio, cwd, ignoreError, env, ...execaOptions }: CommonOptions<string> & {
        command: string;
        args: string[];
        cwd?: string;
        ignoreError?: boolean;
    }): string;
    executeCommand({ command, args, stdio, cwd, ignoreError, env, ...execaOptions }: CommonOptions<string> & {
        command: string;
        args: string[];
        cwd?: string;
        ignoreError?: boolean;
    }): Promise<string>;
}

declare class JsPackageManagerFactory {
    static getPackageManager({ force }?: {
        force?: PackageManagerName;
    }, cwd?: string): JsPackageManager;
}

declare function parseList(str: string): string[];
declare function getStorybookVersion(packageManager: JsPackageManager): Promise<string>;
declare function getEnvConfig(program: Record<string, any>, configEnv: Record<string, any>): void;
/**
 * Given a file name, creates an object with utilities to manage a log file.
 * It creates a temporary log file which you can manage with the returned functions.
 * You can then decide whether to move the log file to the users project, or remove it.
 *
 * @example
 * ```
 *  const { logStream, moveLogFile, removeLogFile, clearLogFile, readLogFile } = await createLogStream('my-log-file.log');
 *
 *  // SCENARIO 1:
 *  // you can write custom messages to generate a log file
 *  logStream.write('my log message');
 *  await moveLogFile();
 *
 *  // SCENARIO 2:
 *  // or you can pass it to stdio and capture the output of that command
 *  try {
 *    await this.executeCommand({
 *      command: 'pnpm',
 *      args: ['info', packageName, ...args],
 *      // do not output to the user, and send stdio and stderr to log file
 *      stdio: ['ignore', logStream, logStream]
 *    });
 *  } catch (err) {
 *    // do something with the log file content
 *    const output = await readLogFile();
 *    // move the log file to the users project
 *    await moveLogFile();
 *  }
 *  // success, no need to keep the log file
 *  await removeLogFile();
 *
 * ```
 */
declare const createLogStream: (logFileName?: string) => Promise<{
    moveLogFile: () => Promise<void>;
    removeLogFile: () => Promise<void>;
    clearLogFile: () => Promise<void>;
    readLogFile: () => Promise<string>;
    logStream: WriteStream;
}>;
declare const isCorePackage: (pkg: string) => boolean;

export { JsPackageManager, JsPackageManagerFactory, PackageJsonWithDepsAndDevDeps, PackageJsonWithMaybeDeps, PackageManagerName, createLogStream, getEnvConfig, getPackageDetails, getStorybookVersion, isCorePackage, parseList, useNpmWarning, _default as versions };
