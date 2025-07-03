import { FileSystemCache } from 'file-system-cache';
import { TransformOptions } from '@babel/core';
import { Server } from 'http';
import { PM } from 'detect-package-manager';
import { BinaryLike } from 'crypto';

/**
Matches any [primitive value](https://developer.mozilla.org/en-US/docs/Glossary/Primitive).

@category Type
*/
type Primitive =
	| null
	| undefined
	| string
	| number
	| boolean
	| symbol
	| bigint;

declare global {
	interface SymbolConstructor {
		readonly observable: symbol;
	}
}

/**
Allows creating a union type by combining primitive types and literal types without sacrificing auto-completion in IDEs for the literal type part of the union.

Currently, when a union type of a primitive type is combined with literal types, TypeScript loses all information about the combined literals. Thus, when such type is used in an IDE with autocompletion, no suggestions are made for the declared literals.

This type is a workaround for [Microsoft/TypeScript#29729](https://github.com/Microsoft/TypeScript/issues/29729). It will be removed as soon as it's not needed anymore.

@example
```
import type {LiteralUnion} from 'type-fest';

// Before

type Pet = 'dog' | 'cat' | string;

const pet: Pet = '';
// Start typing in your TypeScript-enabled IDE.
// You **will not** get auto-completion for `dog` and `cat` literals.

// After

type Pet2 = LiteralUnion<'dog' | 'cat', string>;

const pet: Pet2 = '';
// You **will** get auto-completion for `dog` and `cat` literals.
```

@category Type
*/
type LiteralUnion<
	LiteralType,
	BaseType extends Primitive,
> = LiteralType | (BaseType & Record<never, never>);

declare namespace PackageJson$1 {
	/**
	A person who has been involved in creating or maintaining the package.
	*/
	export type Person =
		| string
		| {
			name: string;
			url?: string;
			email?: string;
		};

	export type BugsLocation =
		| string
		| {
			/**
			The URL to the package's issue tracker.
			*/
			url?: string;

			/**
			The email address to which issues should be reported.
			*/
			email?: string;
		};

	export interface DirectoryLocations {
		[directoryType: string]: unknown;

		/**
		Location for executable scripts. Sugar to generate entries in the `bin` property by walking the folder.
		*/
		bin?: string;

		/**
		Location for Markdown files.
		*/
		doc?: string;

		/**
		Location for example scripts.
		*/
		example?: string;

		/**
		Location for the bulk of the library.
		*/
		lib?: string;

		/**
		Location for man pages. Sugar to generate a `man` array by walking the folder.
		*/
		man?: string;

		/**
		Location for test files.
		*/
		test?: string;
	}

	export type Scripts = {
		/**
		Run **before** the package is published (Also run on local `npm install` without any arguments).
		*/
		prepublish?: string;

		/**
		Run both **before** the package is packed and published, and on local `npm install` without any arguments. This is run **after** `prepublish`, but **before** `prepublishOnly`.
		*/
		prepare?: string;

		/**
		Run **before** the package is prepared and packed, **only** on `npm publish`.
		*/
		prepublishOnly?: string;

		/**
		Run **before** a tarball is packed (on `npm pack`, `npm publish`, and when installing git dependencies).
		*/
		prepack?: string;

		/**
		Run **after** the tarball has been generated and moved to its final destination.
		*/
		postpack?: string;

		/**
		Run **after** the package is published.
		*/
		publish?: string;

		/**
		Run **after** the package is published.
		*/
		postpublish?: string;

		/**
		Run **before** the package is installed.
		*/
		preinstall?: string;

		/**
		Run **after** the package is installed.
		*/
		install?: string;

		/**
		Run **after** the package is installed and after `install`.
		*/
		postinstall?: string;

		/**
		Run **before** the package is uninstalled and before `uninstall`.
		*/
		preuninstall?: string;

		/**
		Run **before** the package is uninstalled.
		*/
		uninstall?: string;

		/**
		Run **after** the package is uninstalled.
		*/
		postuninstall?: string;

		/**
		Run **before** bump the package version and before `version`.
		*/
		preversion?: string;

		/**
		Run **before** bump the package version.
		*/
		version?: string;

		/**
		Run **after** bump the package version.
		*/
		postversion?: string;

		/**
		Run with the `npm test` command, before `test`.
		*/
		pretest?: string;

		/**
		Run with the `npm test` command.
		*/
		test?: string;

		/**
		Run with the `npm test` command, after `test`.
		*/
		posttest?: string;

		/**
		Run with the `npm stop` command, before `stop`.
		*/
		prestop?: string;

		/**
		Run with the `npm stop` command.
		*/
		stop?: string;

		/**
		Run with the `npm stop` command, after `stop`.
		*/
		poststop?: string;

		/**
		Run with the `npm start` command, before `start`.
		*/
		prestart?: string;

		/**
		Run with the `npm start` command.
		*/
		start?: string;

		/**
		Run with the `npm start` command, after `start`.
		*/
		poststart?: string;

		/**
		Run with the `npm restart` command, before `restart`. Note: `npm restart` will run the `stop` and `start` scripts if no `restart` script is provided.
		*/
		prerestart?: string;

		/**
		Run with the `npm restart` command. Note: `npm restart` will run the `stop` and `start` scripts if no `restart` script is provided.
		*/
		restart?: string;

		/**
		Run with the `npm restart` command, after `restart`. Note: `npm restart` will run the `stop` and `start` scripts if no `restart` script is provided.
		*/
		postrestart?: string;
	} & Partial<Record<string, string>>;

	/**
	Dependencies of the package. The version range is a string which has one or more space-separated descriptors. Dependencies can also be identified with a tarball or Git URL.
	*/
	export type Dependency = Partial<Record<string, string>>;

	/**
	Conditions which provide a way to resolve a package entry point based on the environment.
	*/
	export type ExportCondition = LiteralUnion<
		| 'import'
		| 'require'
		| 'node'
		| 'node-addons'
		| 'deno'
		| 'browser'
		| 'electron'
		| 'react-native'
		| 'default',
		string
	>;

	type ExportConditions = {[condition in ExportCondition]: Exports};

	/**
	Entry points of a module, optionally with conditions and subpath exports.
	*/
	export type Exports =
	| null
	| string
	| Array<string | ExportConditions>
	| ExportConditions
	| {[path: string]: Exports}; // eslint-disable-line @typescript-eslint/consistent-indexed-object-style

	/**
	Import map entries of a module, optionally with conditions.
	*/
	export type Imports = { // eslint-disable-line @typescript-eslint/consistent-indexed-object-style
		[key: string]: string | {[key in ExportCondition]: Exports};
	};

	export interface NonStandardEntryPoints {
		/**
		An ECMAScript module ID that is the primary entry point to the program.
		*/
		module?: string;

		/**
		A module ID with untranspiled code that is the primary entry point to the program.
		*/
		esnext?:
		| string
		| {
			[moduleName: string]: string | undefined;
			main?: string;
			browser?: string;
		};

		/**
		A hint to JavaScript bundlers or component tools when packaging modules for client side use.
		*/
		browser?:
		| string
		| Partial<Record<string, string | false>>;

		/**
		Denote which files in your project are "pure" and therefore safe for Webpack to prune if unused.

		[Read more.](https://webpack.js.org/guides/tree-shaking/)
		*/
		sideEffects?: boolean | string[];
	}

	export interface TypeScriptConfiguration {
		/**
		Location of the bundled TypeScript declaration file.
		*/
		types?: string;

		/**
		Version selection map of TypeScript.
		*/
		typesVersions?: Partial<Record<string, Partial<Record<string, string[]>>>>;

		/**
		Location of the bundled TypeScript declaration file. Alias of `types`.
		*/
		typings?: string;
	}

	/**
	An alternative configuration for Yarn workspaces.
	*/
	export interface WorkspaceConfig {
		/**
		An array of workspace pattern strings which contain the workspace packages.
		*/
		packages?: WorkspacePattern[];

		/**
		Designed to solve the problem of packages which break when their `node_modules` are moved to the root workspace directory - a process known as hoisting. For these packages, both within your workspace, and also some that have been installed via `node_modules`, it is important to have a mechanism for preventing the default Yarn workspace behavior. By adding workspace pattern strings here, Yarn will resume non-workspace behavior for any package which matches the defined patterns.

		[Read more](https://classic.yarnpkg.com/blog/2018/02/15/nohoist/)
		*/
		nohoist?: WorkspacePattern[];
	}

	/**
	A workspace pattern points to a directory or group of directories which contain packages that should be included in the workspace installation process.

	The patterns are handled with [minimatch](https://github.com/isaacs/minimatch).

	@example
	`docs` → Include the docs directory and install its dependencies.
	`packages/*` → Include all nested directories within the packages directory, like `packages/cli` and `packages/core`.
	*/
	type WorkspacePattern = string;

	export interface YarnConfiguration {
		/**
		Used to configure [Yarn workspaces](https://classic.yarnpkg.com/docs/workspaces/).

		Workspaces allow you to manage multiple packages within the same repository in such a way that you only need to run `yarn install` once to install all of them in a single pass.

		Please note that the top-level `private` property of `package.json` **must** be set to `true` in order to use workspaces.
		*/
		workspaces?: WorkspacePattern[] | WorkspaceConfig;

		/**
		If your package only allows one version of a given dependency, and you’d like to enforce the same behavior as `yarn install --flat` on the command-line, set this to `true`.

		Note that if your `package.json` contains `"flat": true` and other packages depend on yours (e.g. you are building a library rather than an app), those other packages will also need `"flat": true` in their `package.json` or be installed with `yarn install --flat` on the command-line.
		*/
		flat?: boolean;

		/**
		Selective version resolutions. Allows the definition of custom package versions inside dependencies without manual edits in the `yarn.lock` file.
		*/
		resolutions?: Dependency;
	}

	export interface JSPMConfiguration {
		/**
		JSPM configuration.
		*/
		jspm?: PackageJson$1;
	}

	/**
	Type for [npm's `package.json` file](https://docs.npmjs.com/creating-a-package-json-file). Containing standard npm properties.
	*/
	export interface PackageJsonStandard {
		/**
		The name of the package.
		*/
		name?: string;

		/**
		Package version, parseable by [`node-semver`](https://github.com/npm/node-semver).
		*/
		version?: string;

		/**
		Package description, listed in `npm search`.
		*/
		description?: string;

		/**
		Keywords associated with package, listed in `npm search`.
		*/
		keywords?: string[];

		/**
		The URL to the package's homepage.
		*/
		homepage?: LiteralUnion<'.', string>;

		/**
		The URL to the package's issue tracker and/or the email address to which issues should be reported.
		*/
		bugs?: BugsLocation;

		/**
		The license for the package.
		*/
		license?: string;

		/**
		The licenses for the package.
		*/
		licenses?: Array<{
			type?: string;
			url?: string;
		}>;

		author?: Person;

		/**
		A list of people who contributed to the package.
		*/
		contributors?: Person[];

		/**
		A list of people who maintain the package.
		*/
		maintainers?: Person[];

		/**
		The files included in the package.
		*/
		files?: string[];

		/**
		Resolution algorithm for importing ".js" files from the package's scope.

		[Read more.](https://nodejs.org/api/esm.html#esm_package_json_type_field)
		*/
		type?: 'module' | 'commonjs';

		/**
		The module ID that is the primary entry point to the program.
		*/
		main?: string;

		/**
		Subpath exports to define entry points of the package.

		[Read more.](https://nodejs.org/api/packages.html#subpath-exports)
		*/
		exports?: Exports;

		/**
		Subpath imports to define internal package import maps that only apply to import specifiers from within the package itself.

		[Read more.](https://nodejs.org/api/packages.html#subpath-imports)
		*/
		imports?: Imports;

		/**
		The executable files that should be installed into the `PATH`.
		*/
		bin?:
		| string
		| Partial<Record<string, string>>;

		/**
		Filenames to put in place for the `man` program to find.
		*/
		man?: string | string[];

		/**
		Indicates the structure of the package.
		*/
		directories?: DirectoryLocations;

		/**
		Location for the code repository.
		*/
		repository?:
		| string
		| {
			type: string;
			url: string;

			/**
			Relative path to package.json if it is placed in non-root directory (for example if it is part of a monorepo).

			[Read more.](https://github.com/npm/rfcs/blob/latest/implemented/0010-monorepo-subdirectory-declaration.md)
			*/
			directory?: string;
		};

		/**
		Script commands that are run at various times in the lifecycle of the package. The key is the lifecycle event, and the value is the command to run at that point.
		*/
		scripts?: Scripts;

		/**
		Is used to set configuration parameters used in package scripts that persist across upgrades.
		*/
		config?: Record<string, unknown>;

		/**
		The dependencies of the package.
		*/
		dependencies?: Dependency;

		/**
		Additional tooling dependencies that are not required for the package to work. Usually test, build, or documentation tooling.
		*/
		devDependencies?: Dependency;

		/**
		Dependencies that are skipped if they fail to install.
		*/
		optionalDependencies?: Dependency;

		/**
		Dependencies that will usually be required by the package user directly or via another dependency.
		*/
		peerDependencies?: Dependency;

		/**
		Indicate peer dependencies that are optional.
		*/
		peerDependenciesMeta?: Partial<Record<string, {optional: true}>>;

		/**
		Package names that are bundled when the package is published.
		*/
		bundledDependencies?: string[];

		/**
		Alias of `bundledDependencies`.
		*/
		bundleDependencies?: string[];

		/**
		Engines that this package runs on.
		*/
		engines?: {
			[EngineName in 'npm' | 'node' | string]?: string;
		};

		/**
		@deprecated
		*/
		engineStrict?: boolean;

		/**
		Operating systems the module runs on.
		*/
		os?: Array<LiteralUnion<
		| 'aix'
		| 'darwin'
		| 'freebsd'
		| 'linux'
		| 'openbsd'
		| 'sunos'
		| 'win32'
		| '!aix'
		| '!darwin'
		| '!freebsd'
		| '!linux'
		| '!openbsd'
		| '!sunos'
		| '!win32',
		string
		>>;

		/**
		CPU architectures the module runs on.
		*/
		cpu?: Array<LiteralUnion<
		| 'arm'
		| 'arm64'
		| 'ia32'
		| 'mips'
		| 'mipsel'
		| 'ppc'
		| 'ppc64'
		| 's390'
		| 's390x'
		| 'x32'
		| 'x64'
		| '!arm'
		| '!arm64'
		| '!ia32'
		| '!mips'
		| '!mipsel'
		| '!ppc'
		| '!ppc64'
		| '!s390'
		| '!s390x'
		| '!x32'
		| '!x64',
		string
		>>;

		/**
		If set to `true`, a warning will be shown if package is installed locally. Useful if the package is primarily a command-line application that should be installed globally.

		@deprecated
		*/
		preferGlobal?: boolean;

		/**
		If set to `true`, then npm will refuse to publish it.
		*/
		private?: boolean;

		/**
		A set of config values that will be used at publish-time. It's especially handy to set the tag, registry or access, to ensure that a given package is not tagged with 'latest', published to the global public registry or that a scoped module is private by default.
		*/
		publishConfig?: PublishConfig;

		/**
		Describes and notifies consumers of a package's monetary support information.

		[Read more.](https://github.com/npm/rfcs/blob/latest/accepted/0017-add-funding-support.md)
		*/
		funding?: string | {
			/**
			The type of funding.
			*/
			type?: LiteralUnion<
			| 'github'
			| 'opencollective'
			| 'patreon'
			| 'individual'
			| 'foundation'
			| 'corporation',
			string
			>;

			/**
			The URL to the funding page.
			*/
			url: string;
		};
	}

	export interface PublishConfig {
		/**
		Additional, less common properties from the [npm docs on `publishConfig`](https://docs.npmjs.com/cli/v7/configuring-npm/package-json#publishconfig).
		*/
		[additionalProperties: string]: unknown;

		/**
		When publishing scoped packages, the access level defaults to restricted. If you want your scoped package to be publicly viewable (and installable) set `--access=public`. The only valid values for access are public and restricted. Unscoped packages always have an access level of public.
		*/
		access?: 'public' | 'restricted';

		/**
		The base URL of the npm registry.

		Default: `'https://registry.npmjs.org/'`
		*/
		registry?: string;

		/**
		The tag to publish the package under.

		Default: `'latest'`
		*/
		tag?: string;
	}
}

/**
Type for [npm's `package.json` file](https://docs.npmjs.com/creating-a-package-json-file). Also includes types for fields used by other popular projects, like TypeScript and Yarn.

@category File
*/
type PackageJson$1 =
PackageJson$1.PackageJsonStandard &
PackageJson$1.NonStandardEntryPoints &
PackageJson$1.TypeScriptConfiguration &
PackageJson$1.YarnConfiguration &
PackageJson$1.JSPMConfiguration;

type StoryId = string;
type ComponentTitle = string;
type StoryName = string;
type Tag = string;
interface Parameters {
    [name: string]: any;
}

interface Plugin {
    (module: Program): Program;
}
type TerserEcmaVersion = 5 | 2015 | 2016 | string | number;
interface JsMinifyOptions {
    compress?: TerserCompressOptions | boolean;
    format?: JsFormatOptions & ToSnakeCaseProperties<JsFormatOptions>;
    mangle?: TerserMangleOptions | boolean;
    ecma?: TerserEcmaVersion;
    keep_classnames?: boolean;
    keep_fnames?: boolean;
    module?: boolean;
    safari10?: boolean;
    toplevel?: boolean;
    sourceMap?: boolean;
    outputPath?: string;
    inlineSourcesContent?: boolean;
}
/**
 * @example ToSnakeCase<'indentLevel'> == 'indent_level'
 */
type ToSnakeCase<T extends string> = T extends `${infer A}${infer B}` ? `${A extends Lowercase<A> ? A : `_${Lowercase<A>}`}${ToSnakeCase<B>}` : T;
/**
 * @example ToSnakeCaseProperties<{indentLevel: 3}> == {indent_level: 3}
 */
type ToSnakeCaseProperties<T> = {
    [K in keyof T as K extends string ? ToSnakeCase<K> : K]: T[K];
};
/**
 * These properties are mostly not implemented yet,
 * but it exists to support passing terser config to swc minify
 * without modification.
 */
interface JsFormatOptions {
    /**
     * Currently noop.
     * @default false
     * @alias ascii_only
     */
    asciiOnly?: boolean;
    /**
     * Currently noop.
     * @default false
     */
    beautify?: boolean;
    /**
     * Currently noop.
     * @default false
     */
    braces?: boolean;
    /**
     * - `false`: removes all comments
     * - `'some'`: preserves some comments
     * - `'all'`: preserves all comments
     * @default false
     */
    comments?: false | "some" | "all";
    /**
     * Currently noop.
     * @default 5
     */
    ecma?: TerserEcmaVersion;
    /**
     * Currently noop.
     * @alias indent_level
     */
    indentLevel?: number;
    /**
     * Currently noop.
     * @alias indent_start
     */
    indentStart?: number;
    /**
     * Currently noop.
     * @alias inline_script
     */
    inlineScript?: number;
    /**
     * Currently noop.
     * @alias keep_numbers
     */
    keepNumbers?: number;
    /**
     * Currently noop.
     * @alias keep_quoted_props
     */
    keepQuotedProps?: boolean;
    /**
     * Currently noop.
     * @alias max_line_len
     */
    maxLineLen?: number | false;
    /**
     * Currently noop.
     */
    preamble?: string;
    /**
     * Currently noop.
     * @alias quote_keys
     */
    quoteKeys?: boolean;
    /**
     * Currently noop.
     * @alias quote_style
     */
    quoteStyle?: boolean;
    /**
     * Currently noop.
     * @alias preserve_annotations
     */
    preserveAnnotations?: boolean;
    /**
     * Currently noop.
     */
    safari10?: boolean;
    /**
     * Currently noop.
     */
    semicolons?: boolean;
    /**
     * Currently noop.
     */
    shebang?: boolean;
    /**
     * Currently noop.
     */
    webkit?: boolean;
    /**
     * Currently noop.
     * @alias wrap_iife
     */
    wrapIife?: boolean;
    /**
     * Currently noop.
     * @alias wrap_func_args
     */
    wrapFuncArgs?: boolean;
}
interface TerserCompressOptions {
    arguments?: boolean;
    arrows?: boolean;
    booleans?: boolean;
    booleans_as_integers?: boolean;
    collapse_vars?: boolean;
    comparisons?: boolean;
    computed_props?: boolean;
    conditionals?: boolean;
    dead_code?: boolean;
    defaults?: boolean;
    directives?: boolean;
    drop_console?: boolean;
    drop_debugger?: boolean;
    ecma?: TerserEcmaVersion;
    evaluate?: boolean;
    expression?: boolean;
    global_defs?: any;
    hoist_funs?: boolean;
    hoist_props?: boolean;
    hoist_vars?: boolean;
    ie8?: boolean;
    if_return?: boolean;
    inline?: 0 | 1 | 2 | 3;
    join_vars?: boolean;
    keep_classnames?: boolean;
    keep_fargs?: boolean;
    keep_fnames?: boolean;
    keep_infinity?: boolean;
    loops?: boolean;
    negate_iife?: boolean;
    passes?: number;
    properties?: boolean;
    pure_getters?: any;
    pure_funcs?: string[];
    reduce_funcs?: boolean;
    reduce_vars?: boolean;
    sequences?: any;
    side_effects?: boolean;
    switches?: boolean;
    top_retain?: any;
    toplevel?: any;
    typeofs?: boolean;
    unsafe?: boolean;
    unsafe_passes?: boolean;
    unsafe_arrows?: boolean;
    unsafe_comps?: boolean;
    unsafe_function?: boolean;
    unsafe_math?: boolean;
    unsafe_symbols?: boolean;
    unsafe_methods?: boolean;
    unsafe_proto?: boolean;
    unsafe_regexp?: boolean;
    unsafe_undefined?: boolean;
    unused?: boolean;
    const_to_let?: boolean;
    module?: boolean;
}
interface TerserMangleOptions {
    props?: TerserManglePropertiesOptions;
    toplevel?: boolean;
    keep_classnames?: boolean;
    keep_fnames?: boolean;
    keep_private_props?: boolean;
    ie8?: boolean;
    safari10?: boolean;
    reserved?: string[];
}
interface TerserManglePropertiesOptions {
}
/**
 * Programmatic options.
 */
interface Options$2 extends Config {
    /**
     * If true, a file is parsed as a script instead of module.
     */
    script?: boolean;
    /**
     * The working directory that all paths in the programmatic
     * options will be resolved relative to.
     *
     * Defaults to `process.cwd()`.
     */
    cwd?: string;
    caller?: CallerOptions;
    /** The filename associated with the code currently being compiled,
     * if there is one. The filename is optional, but not all of Swc's
     * functionality is available when the filename is unknown, because a
     * subset of options rely on the filename for their functionality.
     *
     * The three primary cases users could run into are:
     *
     * - The filename is exposed to plugins. Some plugins may require the
     * presence of the filename.
     * - Options like "test", "exclude", and "ignore" require the filename
     * for string/RegExp matching.
     * - .swcrc files are loaded relative to the file being compiled.
     * If this option is omitted, Swc will behave as if swcrc: false has been set.
     */
    filename?: string;
    /**
     * The initial path that will be processed based on the "rootMode" to
     * determine the conceptual root folder for the current Swc project.
     * This is used in two primary cases:
     *
     * - The base directory when checking for the default "configFile" value
     * - The default value for "swcrcRoots".
     *
     * Defaults to `opts.cwd`
     */
    root?: string;
    /**
     * This option, combined with the "root" value, defines how Swc chooses
     * its project root. The different modes define different ways that Swc
     * can process the "root" value to get the final project root.
     *
     * "root" - Passes the "root" value through as unchanged.
     * "upward" - Walks upward from the "root" directory, looking for a directory
     * containing a swc.config.js file, and throws an error if a swc.config.js
     * is not found.
     * "upward-optional" - Walk upward from the "root" directory, looking for
     * a directory containing a swc.config.js file, and falls back to "root"
     *  if a swc.config.js is not found.
     *
     *
     * "root" is the default mode because it avoids the risk that Swc
     * will accidentally load a swc.config.js that is entirely outside
     * of the current project folder. If you use "upward-optional",
     * be aware that it will walk up the directory structure all the
     * way to the filesystem root, and it is always possible that someone
     * will have a forgotten swc.config.js in their home directory,
     * which could cause unexpected errors in your builds.
     *
     *
     * Users with monorepo project structures that run builds/tests on a
     * per-package basis may well want to use "upward" since monorepos
     * often have a swc.config.js in the project root. Running Swc
     * in a monorepo subdirectory without "upward", will cause Swc
     * to skip loading any swc.config.js files in the project root,
     * which can lead to unexpected errors and compilation failure.
     */
    rootMode?: "root" | "upward" | "upward-optional";
    /**
     * The current active environment used during configuration loading.
     * This value is used as the key when resolving "env" configs,
     * and is also available inside configuration functions, plugins,
     * and presets, via the api.env() function.
     *
     * Defaults to `process.env.SWC_ENV || process.env.NODE_ENV || "development"`
     */
    envName?: string;
    /**
     * Defaults to searching for a default `.swcrc` file, but can
     * be passed the path of any JS or JSON5 config file.
     *
     *
     * NOTE: This option does not affect loading of .swcrc files,
     * so while it may be tempting to do configFile: "./foo/.swcrc",
     * it is not recommended. If the given .swcrc is loaded via the
     * standard file-relative logic, you'll end up loading the same
     * config file twice, merging it with itself. If you are linking
     * a specific config file, it is recommended to stick with a
     * naming scheme that is independent of the "swcrc" name.
     *
     * Defaults to `path.resolve(opts.root, ".swcrc")`
     */
    configFile?: string | boolean;
    /**
     * true will enable searching for configuration files relative to the "filename" provided to Swc.
     *
     * A swcrc value passed in the programmatic options will override one set within a configuration file.
     *
     * Note: .swcrc files are only loaded if the current "filename" is inside of
     *  a package that matches one of the "swcrcRoots" packages.
     *
     *
     * Defaults to true as long as the filename option has been specified
     */
    swcrc?: boolean;
    /**
     * By default, Babel will only search for .babelrc files within the "root" package
     *  because otherwise Babel cannot know if a given .babelrc is meant to be loaded,
     *  or if it's "plugins" and "presets" have even been installed, since the file
     *  being compiled could be inside node_modules, or have been symlinked into the project.
     *
     *
     * This option allows users to provide a list of other packages that should be
     * considered "root" packages when considering whether to load .babelrc files.
     *
     *
     * For example, a monorepo setup that wishes to allow individual packages
     * to have their own configs might want to do
     *
     *
     *
     * Defaults to `opts.root`
     */
    swcrcRoots?: boolean | MatchPattern | MatchPattern[];
    /**
     * `true` will attempt to load an input sourcemap from the file itself, if it
     * contains a //# sourceMappingURL=... comment. If no map is found, or the
     * map fails to load and parse, it will be silently discarded.
     *
     *  If an object is provided, it will be treated as the source map object itself.
     *
     * Defaults to `true`.
     */
    inputSourceMap?: boolean | string;
    /**
     * The name to use for the file inside the source map object.
     *
     * Defaults to `path.basename(opts.filenameRelative)` when available, or `"unknown"`.
     */
    sourceFileName?: string;
    /**
     * The sourceRoot fields to set in the generated source map, if one is desired.
     */
    sourceRoot?: string;
    plugin?: Plugin;
    isModule?: boolean | "unknown";
    /**
     * Destination path. Note that this value is used only to fix source path
     * of source map files and swc does not write output to this path.
     */
    outputPath?: string;
}
interface CallerOptions {
    name: string;
    [key: string]: any;
}
/**
 * .swcrc
 */
interface Config {
    /**
     * Note: The type is string because it follows rust's regex syntax.
     */
    test?: string | string[];
    /**
     * Note: The type is string because it follows rust's regex syntax.
     */
    exclude?: string | string[];
    env?: EnvConfig;
    jsc?: JscConfig;
    module?: ModuleConfig;
    minify?: boolean;
    /**
     * - true to generate a sourcemap for the code and include it in the result object.
     * - "inline" to generate a sourcemap and append it as a data URL to the end of the code, but not include it in the result object.
     *
     * `swc-cli` overloads some of these to also affect how maps are written to disk:
     *
     * - true will write the map to a .map file on disk
     * - "inline" will write the file directly, so it will have a data: containing the map
     * - Note: These options are bit weird, so it may make the most sense to just use true
     *  and handle the rest in your own code, depending on your use case.
     */
    sourceMaps?: boolean | "inline";
    inlineSourcesContent?: boolean;
}
/**
 * Configuration ported from babel-preset-env
 */
interface EnvConfig {
    mode?: "usage" | "entry";
    debug?: boolean;
    dynamicImport?: boolean;
    loose?: boolean;
    skip?: string[];
    include?: string[];
    exclude?: string[];
    /**
     * The version of the used core js.
     *
     */
    coreJs?: string;
    targets?: any;
    path?: string;
    shippedProposals?: boolean;
    /**
     * Enable all transforms
     */
    forceAllTransforms?: boolean;
}
interface JscConfig {
    loose?: boolean;
    /**
     * Defaults to EsParserConfig
     */
    parser?: ParserConfig;
    transform?: TransformConfig;
    /**
     * Use `@swc/helpers` instead of inline helpers.
     */
    externalHelpers?: boolean;
    /**
     * Defaults to `es3` (which enabled **all** pass).
     */
    target?: JscTarget;
    /**
     * Keep class names.
     */
    keepClassNames?: boolean;
    /**
     * This is experimental, and can be removed without a major version bump.
     */
    experimental?: {
        optimizeHygiene?: boolean;
        /**
         * Preserve `with` in imports and exports.
         */
        keepImportAttributes?: boolean;
        /**
         * Use `assert` instead of `with` for imports and exports.
         * This option only works when `keepImportAttributes` is `true`.
         */
        emitAssertForImportAttributes?: boolean;
        /**
         * Specify the location where SWC stores its intermediate cache files.
         * Currently only transform plugin uses this. If not specified, SWC will
         * create `.swc` directories.
         */
        cacheRoot?: string;
        /**
         * List of custom transform plugins written in WebAssembly.
         * First parameter of tuple indicates the name of the plugin - it can be either
         * a name of the npm package can be resolved, or absolute path to .wasm binary.
         *
         * Second parameter of tuple is JSON based configuration for the plugin.
         */
        plugins?: Array<[string, Record<string, any>]>;
        /**
         * Disable builtin transforms. If enabled, only Wasm plugins are used.
         */
        disableBuiltinTransformsForInternalTesting?: boolean;
    };
    baseUrl?: string;
    paths?: {
        [from: string]: string[];
    };
    minify?: JsMinifyOptions;
    preserveAllComments?: boolean;
}
type JscTarget = "es3" | "es5" | "es2015" | "es2016" | "es2017" | "es2018" | "es2019" | "es2020" | "es2021" | "es2022" | "esnext";
type ParserConfig = TsParserConfig | EsParserConfig;
interface TsParserConfig {
    syntax: "typescript";
    /**
     * Defaults to `false`.
     */
    tsx?: boolean;
    /**
     * Defaults to `false`.
     */
    decorators?: boolean;
    /**
     * Defaults to `false`
     */
    dynamicImport?: boolean;
}
interface EsParserConfig {
    syntax: "ecmascript";
    /**
     * Defaults to false.
     */
    jsx?: boolean;
    /**
     * @deprecated Always true because it's in ecmascript spec.
     */
    numericSeparator?: boolean;
    /**
     * @deprecated Always true because it's in ecmascript spec.
     */
    classPrivateProperty?: boolean;
    /**
     * @deprecated Always true because it's in ecmascript spec.
     */
    privateMethod?: boolean;
    /**
     * @deprecated Always true because it's in ecmascript spec.
     */
    classProperty?: boolean;
    /**
     * Defaults to `false`
     */
    functionBind?: boolean;
    /**
     * Defaults to `false`
     */
    decorators?: boolean;
    /**
     * Defaults to `false`
     */
    decoratorsBeforeExport?: boolean;
    /**
     * Defaults to `false`
     */
    exportDefaultFrom?: boolean;
    /**
     * @deprecated Always true because it's in ecmascript spec.
     */
    exportNamespaceFrom?: boolean;
    /**
     * @deprecated Always true because it's in ecmascript spec.
     */
    dynamicImport?: boolean;
    /**
     * @deprecated Always true because it's in ecmascript spec.
     */
    nullishCoalescing?: boolean;
    /**
     * @deprecated Always true because it's in ecmascript spec.
     */
    optionalChaining?: boolean;
    /**
     * @deprecated Always true because it's in ecmascript spec.
     */
    importMeta?: boolean;
    /**
     * @deprecated Always true because it's in ecmascript spec.
     */
    topLevelAwait?: boolean;
    /**
     * Defaults to `false`
     */
    importAssertions?: boolean;
}
/**
 * Options for transform.
 */
interface TransformConfig {
    /**
     * Effective only if `syntax` supports ƒ.
     */
    react?: ReactConfig;
    constModules?: ConstModulesConfig;
    /**
     * Defaults to null, which skips optimizer pass.
     */
    optimizer?: OptimizerConfig;
    /**
     * https://swc.rs/docs/configuring-swc.html#jsctransformlegacydecorator
     */
    legacyDecorator?: boolean;
    /**
     * https://swc.rs/docs/configuring-swc.html#jsctransformdecoratormetadata
     */
    decoratorMetadata?: boolean;
    treatConstEnumAsEnum?: boolean;
    useDefineForClassFields?: boolean;
}
interface ReactConfig {
    /**
     * Replace the function used when compiling JSX expressions.
     *
     * Defaults to `React.createElement`.
     */
    pragma?: string;
    /**
     * Replace the component used when compiling JSX fragments.
     *
     * Defaults to `React.Fragment`
     */
    pragmaFrag?: string;
    /**
     * Toggles whether or not to throw an error if a XML namespaced tag name is used. For example:
     * `<f:image />`
     *
     * Though the JSX spec allows this, it is disabled by default since React's
     * JSX does not currently have support for it.
     *
     */
    throwIfNamespace?: boolean;
    /**
     * Toggles plugins that aid in development, such as @swc/plugin-transform-react-jsx-self
     * and @swc/plugin-transform-react-jsx-source.
     *
     * Defaults to `false`,
     *
     */
    development?: boolean;
    /**
     * Use `Object.assign()` instead of `_extends`. Defaults to false.
     * @deprecated
     */
    useBuiltins?: boolean;
    /**
     * Enable fast refresh feature for React app
     */
    refresh?: boolean;
    /**
     * jsx runtime
     */
    runtime?: "automatic" | "classic";
    /**
     * Declares the module specifier to be used for importing the `jsx` and `jsxs` factory functions when using `runtime` 'automatic'
     */
    importSource?: string;
}
/**
 *  - `import { DEBUG } from '@ember/env-flags';`
 *  - `import { FEATURE_A, FEATURE_B } from '@ember/features';`
 *
 * See: https://github.com/swc-project/swc/issues/18#issuecomment-466272558
 */
interface ConstModulesConfig {
    globals?: {
        [module: string]: {
            [name: string]: string;
        };
    };
}
interface OptimizerConfig {
    simplify?: boolean;
    globals?: GlobalPassOption;
    jsonify?: {
        minCost: number;
    };
}
/**
 * Options for inline-global pass.
 */
interface GlobalPassOption {
    /**
     * Global variables that should be inlined with passed value.
     *
     * e.g. `{ __DEBUG__: true }`
     */
    vars?: Record<string, string>;
    /**
     * Names of environment variables that should be inlined with the value of corresponding env during build.
     *
     * Defaults to `["NODE_ENV", "SWC_ENV"]`
     */
    envs?: string[];
    /**
     * Replaces typeof calls for passed variables with corresponding value
     *
     * e.g. `{ window: 'object' }`
     */
    typeofs?: Record<string, string>;
}
type ModuleConfig = Es6Config | CommonJsConfig | UmdConfig | AmdConfig | NodeNextConfig | SystemjsConfig;
interface BaseModuleConfig {
    /**
     * By default, when using exports with babel a non-enumerable `__esModule`
     * property is exported. In some cases this property is used to determine
     * if the import is the default export or if it contains the default export.
     *
     * In order to prevent the __esModule property from being exported, you
     *  can set the strict option to true.
     *
     * Defaults to `false`.
     */
    strict?: boolean;
    /**
     * Emits 'use strict' directive.
     *
     * Defaults to `true`.
     */
    strictMode?: boolean;
    /**
     * Changes Babel's compiled import statements to be lazily evaluated when their imported bindings are used for the first time.
     *
     * This can improve initial load time of your module because evaluating dependencies up
     *  front is sometimes entirely un-necessary. This is especially the case when implementing
     *  a library module.
     *
     *
     * The value of `lazy` has a few possible effects:
     *
     *  - `false` - No lazy initialization of any imported module.
     *  - `true` - Do not lazy-initialize local `./foo` imports, but lazy-init `foo` dependencies.
     *
     * Local paths are much more likely to have circular dependencies, which may break if loaded lazily,
     * so they are not lazy by default, whereas dependencies between independent modules are rarely cyclical.
     *
     *  - `Array<string>` - Lazy-initialize all imports with source matching one of the given strings.
     *
     * -----
     *
     * The two cases where imports can never be lazy are:
     *
     *  - `import "foo";`
     *
     * Side-effect imports are automatically non-lazy since their very existence means
     *  that there is no binding to later kick off initialization.
     *
     *  - `export * from "foo"`
     *
     * Re-exporting all names requires up-front execution because otherwise there is no
     * way to know what names need to be exported.
     *
     * Defaults to `false`.
     */
    lazy?: boolean | string[];
    /**
     * @deprecated  Use the `importInterop` option instead.
     *
     * By default, when using exports with swc a non-enumerable __esModule property is exported.
     * This property is then used to determine if the import is the default export or if
     *  it contains the default export.
     *
     * In cases where the auto-unwrapping of default is not needed, you can set the noInterop option
     *  to true to avoid the usage of the interopRequireDefault helper (shown in inline form above).
     *
     * Defaults to `false`.
     */
    noInterop?: boolean;
    /**
     * Defaults to `swc`.
     *
     * CommonJS modules and ECMAScript modules are not fully compatible.
     * However, compilers, bundlers and JavaScript runtimes developed different strategies
     * to make them work together as well as possible.
     *
     * - `swc` (alias: `babel`)
     *
     * When using exports with `swc` a non-enumerable `__esModule` property is exported
     * This property is then used to determine if the import is the default export
     * or if it contains the default export.
     *
     * ```javascript
     * import foo from "foo";
     * import { bar } from "bar";
     * foo;
     * bar;
     *
     * // Is compiled to ...
     *
     * "use strict";
     *
     * function _interop_require_default(obj) {
     *   return obj && obj.__esModule ? obj : { default: obj };
     * }
     *
     * var _foo = _interop_require_default(require("foo"));
     * var _bar = require("bar");
     *
     * _foo.default;
     * _bar.bar;
     * ```
     *
     * When this import interop is used, if both the imported and the importer module are compiled
     * with swc they behave as if none of them was compiled.
     *
     * This is the default behavior.
     *
     * - `node`
     *
     * When importing CommonJS files (either directly written in CommonJS, or generated with a compiler)
     * Node.js always binds the `default` export to the value of `module.exports`.
     *
     * ```javascript
     * import foo from "foo";
     * import { bar } from "bar";
     * foo;
     * bar;
     *
     * // Is compiled to ...
     *
     * "use strict";
     *
     * var _foo = require("foo");
     * var _bar = require("bar");
     *
     * _foo;
     * _bar.bar;
     * ```
     * This is not exactly the same as what Node.js does since swc allows accessing any property of `module.exports`
     * as a named export, while Node.js only allows importing statically analyzable properties of `module.exports`.
     * However, any import working in Node.js will also work when compiled with swc using `importInterop: "node"`.
     *
     * - `none`
     *
     * If you know that the imported file has been transformed with a compiler that stores the `default` export on
     * `exports.default` (such as swc or Babel), you can safely omit the `_interop_require_default` helper.
     *
     * ```javascript
     * import foo from "foo";
     * import { bar } from "bar";
     * foo;
     * bar;
     *
     * // Is compiled to ...
     *
     * "use strict";
     *
     * var _foo = require("foo");
     * var _bar = require("bar");
     *
     * _foo.default;
     * _bar.bar;
     * ```
     */
    importInterop?: "swc" | "babel" | "node" | "none";
    /**
     * Emits `cjs-module-lexer` annotation
     * `cjs-module-lexer` is used in Node.js core for detecting the named exports available when importing a CJS module into ESM.
     * swc will emit `cjs-module-lexer` detectable annotation with this option enabled.
     *
     * Defaults to `true` if import_interop is Node, else `false`
     */
    exportInteropAnnotation?: boolean;
    /**
     * If set to true, dynamic imports will be preserved.
     */
    ignoreDynamic?: boolean;
    allowTopLevelThis?: boolean;
    preserveImportMeta?: boolean;
}
interface Es6Config extends BaseModuleConfig {
    type: "es6";
}
interface NodeNextConfig extends BaseModuleConfig {
    type: "nodenext";
}
interface CommonJsConfig extends BaseModuleConfig {
    type: "commonjs";
}
interface UmdConfig extends BaseModuleConfig {
    type: "umd";
    globals?: {
        [key: string]: string;
    };
}
interface AmdConfig extends BaseModuleConfig {
    type: "amd";
    moduleId?: string;
}
interface SystemjsConfig {
    type: "systemjs";
    allowTopLevelThis?: boolean;
}
interface MatchPattern {
}
interface Span {
    start: number;
    end: number;
    ctxt: number;
}
interface Node {
    type: string;
}
interface HasSpan {
    span: Span;
}
interface HasDecorator {
    decorators?: Decorator[];
}
interface Class extends HasSpan, HasDecorator {
    body: ClassMember[];
    superClass?: Expression;
    isAbstract: boolean;
    typeParams?: TsTypeParameterDeclaration;
    superTypeParams?: TsTypeParameterInstantiation;
    implements: TsExpressionWithTypeArguments[];
}
type ClassMember = Constructor | ClassMethod | PrivateMethod | ClassProperty | PrivateProperty | TsIndexSignature | EmptyStatement | StaticBlock;
interface ClassPropertyBase extends Node, HasSpan, HasDecorator {
    value?: Expression;
    typeAnnotation?: TsTypeAnnotation;
    isStatic: boolean;
    accessibility?: Accessibility;
    isOptional: boolean;
    isOverride: boolean;
    readonly: boolean;
    definite: boolean;
}
interface ClassProperty extends ClassPropertyBase {
    type: "ClassProperty";
    key: PropertyName;
    isAbstract: boolean;
    declare: boolean;
}
interface PrivateProperty extends ClassPropertyBase {
    type: "PrivateProperty";
    key: PrivateName;
}
interface Param extends Node, HasSpan, HasDecorator {
    type: "Parameter";
    pat: Pattern;
}
interface Constructor extends Node, HasSpan {
    type: "Constructor";
    key: PropertyName;
    params: (TsParameterProperty | Param)[];
    body?: BlockStatement;
    accessibility?: Accessibility;
    isOptional: boolean;
}
interface ClassMethodBase extends Node, HasSpan {
    function: Fn;
    kind: MethodKind;
    isStatic: boolean;
    accessibility?: Accessibility;
    isAbstract: boolean;
    isOptional: boolean;
    isOverride: boolean;
}
interface ClassMethod extends ClassMethodBase {
    type: "ClassMethod";
    key: PropertyName;
}
interface PrivateMethod extends ClassMethodBase {
    type: "PrivateMethod";
    key: PrivateName;
}
interface StaticBlock extends Node, HasSpan {
    type: "StaticBlock";
    body: BlockStatement;
}
interface Decorator extends Node, HasSpan {
    type: "Decorator";
    expression: Expression;
}
type MethodKind = "method" | "getter" | "setter";
type Declaration = ClassDeclaration | FunctionDeclaration | VariableDeclaration | TsInterfaceDeclaration | TsTypeAliasDeclaration | TsEnumDeclaration | TsModuleDeclaration;
interface FunctionDeclaration extends Fn {
    type: "FunctionDeclaration";
    identifier: Identifier;
    declare: boolean;
}
interface ClassDeclaration extends Class, Node {
    type: "ClassDeclaration";
    identifier: Identifier;
    declare: boolean;
}
interface VariableDeclaration extends Node, HasSpan {
    type: "VariableDeclaration";
    kind: VariableDeclarationKind;
    declare: boolean;
    declarations: VariableDeclarator[];
}
type VariableDeclarationKind = "var" | "let" | "const";
interface VariableDeclarator extends Node, HasSpan {
    type: "VariableDeclarator";
    id: Pattern;
    init?: Expression;
    definite: boolean;
}
type Expression = ThisExpression | ArrayExpression | ObjectExpression | FunctionExpression | UnaryExpression | UpdateExpression | BinaryExpression | AssignmentExpression | MemberExpression | SuperPropExpression | ConditionalExpression | CallExpression | NewExpression | SequenceExpression | Identifier | Literal | TemplateLiteral | TaggedTemplateExpression | ArrowFunctionExpression | ClassExpression | YieldExpression | MetaProperty | AwaitExpression | ParenthesisExpression | JSXMemberExpression | JSXNamespacedName | JSXEmptyExpression | JSXElement | JSXFragment | TsTypeAssertion | TsConstAssertion | TsNonNullExpression | TsAsExpression | TsSatisfiesExpression | TsInstantiation | PrivateName | OptionalChainingExpression | Invalid;
interface ExpressionBase extends Node, HasSpan {
}
interface Identifier extends ExpressionBase {
    type: "Identifier";
    value: string;
    optional: boolean;
}
interface OptionalChainingExpression extends ExpressionBase {
    type: "OptionalChainingExpression";
    questionDotToken: Span;
    /**
     * Call expression or member expression.
     */
    base: MemberExpression | OptionalChainingCall;
}
interface OptionalChainingCall extends ExpressionBase {
    type: "CallExpression";
    callee: Expression;
    arguments: ExprOrSpread[];
    typeArguments?: TsTypeParameterInstantiation;
}
interface ThisExpression extends ExpressionBase {
    type: "ThisExpression";
}
interface ArrayExpression extends ExpressionBase {
    type: "ArrayExpression";
    elements: (ExprOrSpread | undefined)[];
}
interface ExprOrSpread {
    spread?: Span;
    expression: Expression;
}
interface ObjectExpression extends ExpressionBase {
    type: "ObjectExpression";
    properties: (SpreadElement | Property)[];
}
interface Argument {
    spread?: Span;
    expression: Expression;
}
interface SpreadElement extends Node {
    type: "SpreadElement";
    spread: Span;
    arguments: Expression;
}
interface UnaryExpression extends ExpressionBase {
    type: "UnaryExpression";
    operator: UnaryOperator;
    argument: Expression;
}
interface UpdateExpression extends ExpressionBase {
    type: "UpdateExpression";
    operator: UpdateOperator;
    prefix: boolean;
    argument: Expression;
}
interface BinaryExpression extends ExpressionBase {
    type: "BinaryExpression";
    operator: BinaryOperator;
    left: Expression;
    right: Expression;
}
interface FunctionExpression extends Fn, ExpressionBase {
    type: "FunctionExpression";
    identifier?: Identifier;
}
interface ClassExpression extends Class, ExpressionBase {
    type: "ClassExpression";
    identifier?: Identifier;
}
interface AssignmentExpression extends ExpressionBase {
    type: "AssignmentExpression";
    operator: AssignmentOperator;
    left: Expression | Pattern;
    right: Expression;
}
interface MemberExpression extends ExpressionBase {
    type: "MemberExpression";
    object: Expression;
    property: Identifier | PrivateName | ComputedPropName;
}
interface SuperPropExpression extends ExpressionBase {
    type: "SuperPropExpression";
    obj: Super;
    property: Identifier | ComputedPropName;
}
interface ConditionalExpression extends ExpressionBase {
    type: "ConditionalExpression";
    test: Expression;
    consequent: Expression;
    alternate: Expression;
}
interface Super extends Node, HasSpan {
    type: "Super";
}
interface Import extends Node, HasSpan {
    type: "Import";
}
interface CallExpression extends ExpressionBase {
    type: "CallExpression";
    callee: Super | Import | Expression;
    arguments: Argument[];
    typeArguments?: TsTypeParameterInstantiation;
}
interface NewExpression extends ExpressionBase {
    type: "NewExpression";
    callee: Expression;
    arguments?: Argument[];
    typeArguments?: TsTypeParameterInstantiation;
}
interface SequenceExpression extends ExpressionBase {
    type: "SequenceExpression";
    expressions: Expression[];
}
interface ArrowFunctionExpression extends ExpressionBase {
    type: "ArrowFunctionExpression";
    params: Pattern[];
    body: BlockStatement | Expression;
    async: boolean;
    generator: boolean;
    typeParameters?: TsTypeParameterDeclaration;
    returnType?: TsTypeAnnotation;
}
interface YieldExpression extends ExpressionBase {
    type: "YieldExpression";
    argument?: Expression;
    delegate: boolean;
}
interface MetaProperty extends Node, HasSpan {
    type: "MetaProperty";
    kind: "new.target" | "import.meta";
}
interface AwaitExpression extends ExpressionBase {
    type: "AwaitExpression";
    argument: Expression;
}
interface TemplateLiteral extends ExpressionBase {
    type: "TemplateLiteral";
    expressions: Expression[];
    quasis: TemplateElement[];
}
interface TaggedTemplateExpression extends ExpressionBase {
    type: "TaggedTemplateExpression";
    tag: Expression;
    typeParameters?: TsTypeParameterInstantiation;
    template: TemplateLiteral;
}
interface TemplateElement extends ExpressionBase {
    type: "TemplateElement";
    tail: boolean;
    cooked?: string;
    raw: string;
}
interface ParenthesisExpression extends ExpressionBase {
    type: "ParenthesisExpression";
    expression: Expression;
}
interface Fn extends HasSpan, HasDecorator {
    params: Param[];
    body?: BlockStatement;
    generator: boolean;
    async: boolean;
    typeParameters?: TsTypeParameterDeclaration;
    returnType?: TsTypeAnnotation;
}
interface PatternBase extends Node, HasSpan {
    typeAnnotation?: TsTypeAnnotation;
}
interface PrivateName extends ExpressionBase {
    type: "PrivateName";
    id: Identifier;
}
type JSXObject = JSXMemberExpression | Identifier;
interface JSXMemberExpression extends Node {
    type: "JSXMemberExpression";
    object: JSXObject;
    property: Identifier;
}
/**
 * XML-based namespace syntax:
 */
interface JSXNamespacedName extends Node {
    type: "JSXNamespacedName";
    namespace: Identifier;
    name: Identifier;
}
interface JSXEmptyExpression extends Node, HasSpan {
    type: "JSXEmptyExpression";
}
interface JSXExpressionContainer extends Node, HasSpan {
    type: "JSXExpressionContainer";
    expression: JSXExpression;
}
type JSXExpression = JSXEmptyExpression | Expression;
interface JSXSpreadChild extends Node, HasSpan {
    type: "JSXSpreadChild";
    expression: Expression;
}
type JSXElementName = Identifier | JSXMemberExpression | JSXNamespacedName;
interface JSXOpeningElement extends Node, HasSpan {
    type: "JSXOpeningElement";
    name: JSXElementName;
    attributes: JSXAttributeOrSpread[];
    selfClosing: boolean;
    typeArguments?: TsTypeParameterInstantiation;
}
type JSXAttributeOrSpread = JSXAttribute | SpreadElement;
interface JSXClosingElement extends Node, HasSpan {
    type: "JSXClosingElement";
    name: JSXElementName;
}
interface JSXAttribute extends Node, HasSpan {
    type: "JSXAttribute";
    name: JSXAttributeName;
    value?: JSXAttrValue;
}
type JSXAttributeName = Identifier | JSXNamespacedName;
type JSXAttrValue = Literal | JSXExpressionContainer | JSXElement | JSXFragment;
interface JSXText extends Node, HasSpan {
    type: "JSXText";
    value: string;
    raw: string;
}
interface JSXElement extends Node, HasSpan {
    type: "JSXElement";
    opening: JSXOpeningElement;
    children: JSXElementChild[];
    closing?: JSXClosingElement;
}
type JSXElementChild = JSXText | JSXExpressionContainer | JSXSpreadChild | JSXElement | JSXFragment;
interface JSXFragment extends Node, HasSpan {
    type: "JSXFragment";
    opening: JSXOpeningFragment;
    children: JSXElementChild[];
    closing: JSXClosingFragment;
}
interface JSXOpeningFragment extends Node, HasSpan {
    type: "JSXOpeningFragment";
}
interface JSXClosingFragment extends Node, HasSpan {
    type: "JSXClosingFragment";
}
type Literal = StringLiteral | BooleanLiteral | NullLiteral | NumericLiteral | BigIntLiteral | RegExpLiteral | JSXText;
interface StringLiteral extends Node, HasSpan {
    type: "StringLiteral";
    value: string;
    raw?: string;
}
interface BooleanLiteral extends Node, HasSpan {
    type: "BooleanLiteral";
    value: boolean;
}
interface NullLiteral extends Node, HasSpan {
    type: "NullLiteral";
}
interface RegExpLiteral extends Node, HasSpan {
    type: "RegExpLiteral";
    pattern: string;
    flags: string;
}
interface NumericLiteral extends Node, HasSpan {
    type: "NumericLiteral";
    value: number;
    raw?: string;
}
interface BigIntLiteral extends Node, HasSpan {
    type: "BigIntLiteral";
    value: bigint;
    raw?: string;
}
type ModuleDeclaration = ImportDeclaration | ExportDeclaration | ExportNamedDeclaration | ExportDefaultDeclaration | ExportDefaultExpression | ExportAllDeclaration | TsImportEqualsDeclaration | TsExportAssignment | TsNamespaceExportDeclaration;
interface ExportDefaultExpression extends Node, HasSpan {
    type: "ExportDefaultExpression";
    expression: Expression;
}
interface ExportDeclaration extends Node, HasSpan {
    type: "ExportDeclaration";
    declaration: Declaration;
}
interface ImportDeclaration extends Node, HasSpan {
    type: "ImportDeclaration";
    specifiers: ImportSpecifier[];
    source: StringLiteral;
    typeOnly: boolean;
    asserts?: ObjectExpression;
}
interface ExportAllDeclaration extends Node, HasSpan {
    type: "ExportAllDeclaration";
    source: StringLiteral;
    asserts?: ObjectExpression;
}
/**
 * - `export { foo } from 'mod'`
 * - `export { foo as bar } from 'mod'`
 */
interface ExportNamedDeclaration extends Node, HasSpan {
    type: "ExportNamedDeclaration";
    specifiers: ExportSpecifier[];
    source?: StringLiteral;
    typeOnly: boolean;
    asserts?: ObjectExpression;
}
interface ExportDefaultDeclaration extends Node, HasSpan {
    type: "ExportDefaultDeclaration";
    decl: DefaultDecl;
}
type DefaultDecl = ClassExpression | FunctionExpression | TsInterfaceDeclaration;
type ImportSpecifier = NamedImportSpecifier | ImportDefaultSpecifier | ImportNamespaceSpecifier;
/**
 * e.g. `import foo from 'mod.js'`
 */
interface ImportDefaultSpecifier extends Node, HasSpan {
    type: "ImportDefaultSpecifier";
    local: Identifier;
}
/**
 * e.g. `import * as foo from 'mod.js'`.
 */
interface ImportNamespaceSpecifier extends Node, HasSpan {
    type: "ImportNamespaceSpecifier";
    local: Identifier;
}
/**
 * e.g. - `import { foo } from 'mod.js'`
 *
 * local = foo, imported = None
 *
 * e.g. `import { foo as bar } from 'mod.js'`
 *
 * local = bar, imported = Some(foo) for
 */
interface NamedImportSpecifier extends Node, HasSpan {
    type: "ImportSpecifier";
    local: Identifier;
    imported?: ModuleExportName;
    isTypeOnly: boolean;
}
type ModuleExportName = Identifier | StringLiteral;
type ExportSpecifier = ExportNamespaceSpecifier | ExportDefaultSpecifier | NamedExportSpecifier;
/**
 * `export * as foo from 'src';`
 */
interface ExportNamespaceSpecifier extends Node, HasSpan {
    type: "ExportNamespaceSpecifier";
    name: ModuleExportName;
}
interface ExportDefaultSpecifier extends Node, HasSpan {
    type: "ExportDefaultSpecifier";
    exported: Identifier;
}
interface NamedExportSpecifier extends Node, HasSpan {
    type: "ExportSpecifier";
    orig: ModuleExportName;
    /**
     * `Some(bar)` in `export { foo as bar }`
     */
    exported?: ModuleExportName;
    isTypeOnly: boolean;
}
interface HasInterpreter {
    /**
     * e.g. `/usr/bin/node` for `#!/usr/bin/node`
     */
    interpreter: string;
}
type Program = Module | Script;
interface Module extends Node, HasSpan, HasInterpreter {
    type: "Module";
    body: ModuleItem[];
}
interface Script extends Node, HasSpan, HasInterpreter {
    type: "Script";
    body: Statement[];
}
type ModuleItem = ModuleDeclaration | Statement;
type BinaryOperator = "==" | "!=" | "===" | "!==" | "<" | "<=" | ">" | ">=" | "<<" | ">>" | ">>>" | "+" | "-" | "*" | "/" | "%" | "|" | "^" | "&" | "||" | "&&" | "in" | "instanceof" | "**" | "??";
type AssignmentOperator = "=" | "+=" | "-=" | "*=" | "/=" | "%=" | "<<=" | ">>=" | ">>>=" | "|=" | "^=" | "&=" | "**=" | "&&=" | "||=" | "??=";
type UpdateOperator = "++" | "--";
type UnaryOperator = "-" | "+" | "!" | "~" | "typeof" | "void" | "delete";
type Pattern = BindingIdentifier | ArrayPattern | RestElement | ObjectPattern | AssignmentPattern | Invalid | Expression;
interface BindingIdentifier extends PatternBase {
    type: "Identifier";
    value: string;
    optional: boolean;
}
interface ArrayPattern extends PatternBase {
    type: "ArrayPattern";
    elements: (Pattern | undefined)[];
    optional: boolean;
}
interface ObjectPattern extends PatternBase {
    type: "ObjectPattern";
    properties: ObjectPatternProperty[];
    optional: boolean;
}
interface AssignmentPattern extends PatternBase {
    type: "AssignmentPattern";
    left: Pattern;
    right: Expression;
}
interface RestElement extends PatternBase {
    type: "RestElement";
    rest: Span;
    argument: Pattern;
}
type ObjectPatternProperty = KeyValuePatternProperty | AssignmentPatternProperty | RestElement;
/**
 * `{key: value}`
 */
interface KeyValuePatternProperty extends Node {
    type: "KeyValuePatternProperty";
    key: PropertyName;
    value: Pattern;
}
/**
 * `{key}` or `{key = value}`
 */
interface AssignmentPatternProperty extends Node, HasSpan {
    type: "AssignmentPatternProperty";
    key: Identifier;
    value?: Expression;
}
/** Identifier is `a` in `{ a, }` */
type Property = Identifier | KeyValueProperty | AssignmentProperty | GetterProperty | SetterProperty | MethodProperty;
interface PropBase extends Node {
    key: PropertyName;
}
interface KeyValueProperty extends PropBase {
    type: "KeyValueProperty";
    value: Expression;
}
interface AssignmentProperty extends Node {
    type: "AssignmentProperty";
    key: Identifier;
    value: Expression;
}
interface GetterProperty extends PropBase, HasSpan {
    type: "GetterProperty";
    typeAnnotation?: TsTypeAnnotation;
    body?: BlockStatement;
}
interface SetterProperty extends PropBase, HasSpan {
    type: "SetterProperty";
    param: Pattern;
    body?: BlockStatement;
}
interface MethodProperty extends PropBase, Fn {
    type: "MethodProperty";
}
type PropertyName = Identifier | StringLiteral | NumericLiteral | ComputedPropName | BigIntLiteral;
interface ComputedPropName extends Node, HasSpan {
    type: "Computed";
    expression: Expression;
}
interface BlockStatement extends Node, HasSpan {
    type: "BlockStatement";
    stmts: Statement[];
}
interface ExpressionStatement extends Node, HasSpan {
    type: "ExpressionStatement";
    expression: Expression;
}
type Statement = BlockStatement | EmptyStatement | DebuggerStatement | WithStatement | ReturnStatement | LabeledStatement | BreakStatement | ContinueStatement | IfStatement | SwitchStatement | ThrowStatement | TryStatement | WhileStatement | DoWhileStatement | ForStatement | ForInStatement | ForOfStatement | Declaration | ExpressionStatement;
interface EmptyStatement extends Node, HasSpan {
    type: "EmptyStatement";
}
interface DebuggerStatement extends Node, HasSpan {
    type: "DebuggerStatement";
}
interface WithStatement extends Node, HasSpan {
    type: "WithStatement";
    object: Expression;
    body: Statement;
}
interface ReturnStatement extends Node, HasSpan {
    type: "ReturnStatement";
    argument?: Expression;
}
interface LabeledStatement extends Node, HasSpan {
    type: "LabeledStatement";
    label: Identifier;
    body: Statement;
}
interface BreakStatement extends Node, HasSpan {
    type: "BreakStatement";
    label?: Identifier;
}
interface ContinueStatement extends Node, HasSpan {
    type: "ContinueStatement";
    label?: Identifier;
}
interface IfStatement extends Node, HasSpan {
    type: "IfStatement";
    test: Expression;
    consequent: Statement;
    alternate?: Statement;
}
interface SwitchStatement extends Node, HasSpan {
    type: "SwitchStatement";
    discriminant: Expression;
    cases: SwitchCase[];
}
interface ThrowStatement extends Node, HasSpan {
    type: "ThrowStatement";
    argument: Expression;
}
interface TryStatement extends Node, HasSpan {
    type: "TryStatement";
    block: BlockStatement;
    handler?: CatchClause;
    finalizer?: BlockStatement;
}
interface WhileStatement extends Node, HasSpan {
    type: "WhileStatement";
    test: Expression;
    body: Statement;
}
interface DoWhileStatement extends Node, HasSpan {
    type: "DoWhileStatement";
    test: Expression;
    body: Statement;
}
interface ForStatement extends Node, HasSpan {
    type: "ForStatement";
    init?: VariableDeclaration | Expression;
    test?: Expression;
    update?: Expression;
    body: Statement;
}
interface ForInStatement extends Node, HasSpan {
    type: "ForInStatement";
    left: VariableDeclaration | Pattern;
    right: Expression;
    body: Statement;
}
interface ForOfStatement extends Node, HasSpan {
    type: "ForOfStatement";
    /**
     *  Span of the await token.
     *
     *  es2018 for-await-of statements, e.g., `for await (const x of xs) {`
     */
    await?: Span;
    left: VariableDeclaration | Pattern;
    right: Expression;
    body: Statement;
}
interface SwitchCase extends Node, HasSpan {
    type: "SwitchCase";
    /**
     * Undefined for default case
     */
    test?: Expression;
    consequent: Statement[];
}
interface CatchClause extends Node, HasSpan {
    type: "CatchClause";
    /**
     * The param is `undefined` if the catch binding is omitted. E.g., `try { foo() } catch {}`
     */
    param?: Pattern;
    body: BlockStatement;
}
interface TsTypeAnnotation extends Node, HasSpan {
    type: "TsTypeAnnotation";
    typeAnnotation: TsType;
}
interface TsTypeParameterDeclaration extends Node, HasSpan {
    type: "TsTypeParameterDeclaration";
    parameters: TsTypeParameter[];
}
interface TsTypeParameter extends Node, HasSpan {
    type: "TsTypeParameter";
    name: Identifier;
    in: boolean;
    out: boolean;
    constraint?: TsType;
    default?: TsType;
}
interface TsTypeParameterInstantiation extends Node, HasSpan {
    type: "TsTypeParameterInstantiation";
    params: TsType[];
}
interface TsParameterProperty extends Node, HasSpan, HasDecorator {
    type: "TsParameterProperty";
    accessibility?: Accessibility;
    override: boolean;
    readonly: boolean;
    param: TsParameterPropertyParameter;
}
type TsParameterPropertyParameter = BindingIdentifier | AssignmentPattern;
interface TsQualifiedName extends Node {
    type: "TsQualifiedName";
    left: TsEntityName;
    right: Identifier;
}
type TsEntityName = TsQualifiedName | Identifier;
type TsTypeElement = TsCallSignatureDeclaration | TsConstructSignatureDeclaration | TsPropertySignature | TsGetterSignature | TsSetterSignature | TsMethodSignature | TsIndexSignature;
interface TsCallSignatureDeclaration extends Node, HasSpan {
    type: "TsCallSignatureDeclaration";
    params: TsFnParameter[];
    typeAnnotation?: TsTypeAnnotation;
    typeParams?: TsTypeParameterDeclaration;
}
interface TsConstructSignatureDeclaration extends Node, HasSpan {
    type: "TsConstructSignatureDeclaration";
    params: TsFnParameter[];
    typeAnnotation?: TsTypeAnnotation;
    typeParams?: TsTypeParameterDeclaration;
}
interface TsPropertySignature extends Node, HasSpan {
    type: "TsPropertySignature";
    readonly: boolean;
    key: Expression;
    computed: boolean;
    optional: boolean;
    init?: Expression;
    params: TsFnParameter[];
    typeAnnotation?: TsTypeAnnotation;
    typeParams?: TsTypeParameterDeclaration;
}
interface TsGetterSignature extends Node, HasSpan {
    type: "TsGetterSignature";
    readonly: boolean;
    key: Expression;
    computed: boolean;
    optional: boolean;
    typeAnnotation?: TsTypeAnnotation;
}
interface TsSetterSignature extends Node, HasSpan {
    type: "TsSetterSignature";
    readonly: boolean;
    key: Expression;
    computed: boolean;
    optional: boolean;
    param: TsFnParameter;
}
interface TsMethodSignature extends Node, HasSpan {
    type: "TsMethodSignature";
    readonly: boolean;
    key: Expression;
    computed: boolean;
    optional: boolean;
    params: TsFnParameter[];
    typeAnn?: TsTypeAnnotation;
    typeParams?: TsTypeParameterDeclaration;
}
interface TsIndexSignature extends Node, HasSpan {
    type: "TsIndexSignature";
    params: TsFnParameter[];
    typeAnnotation?: TsTypeAnnotation;
    readonly: boolean;
    static: boolean;
}
type TsType = TsKeywordType | TsThisType | TsFnOrConstructorType | TsTypeReference | TsTypeQuery | TsTypeLiteral | TsArrayType | TsTupleType | TsOptionalType | TsRestType | TsUnionOrIntersectionType | TsConditionalType | TsInferType | TsParenthesizedType | TsTypeOperator | TsIndexedAccessType | TsMappedType | TsLiteralType | TsTypePredicate | TsImportType;
type TsFnOrConstructorType = TsFunctionType | TsConstructorType;
interface TsKeywordType extends Node, HasSpan {
    type: "TsKeywordType";
    kind: TsKeywordTypeKind;
}
type TsKeywordTypeKind = "any" | "unknown" | "number" | "object" | "boolean" | "bigint" | "string" | "symbol" | "void" | "undefined" | "null" | "never" | "intrinsic";
interface TsThisType extends Node, HasSpan {
    type: "TsThisType";
}
type TsFnParameter = BindingIdentifier | ArrayPattern | RestElement | ObjectPattern;
interface TsFunctionType extends Node, HasSpan {
    type: "TsFunctionType";
    params: TsFnParameter[];
    typeParams?: TsTypeParameterDeclaration;
    typeAnnotation: TsTypeAnnotation;
}
interface TsConstructorType extends Node, HasSpan {
    type: "TsConstructorType";
    params: TsFnParameter[];
    typeParams?: TsTypeParameterDeclaration;
    typeAnnotation: TsTypeAnnotation;
    isAbstract: boolean;
}
interface TsTypeReference extends Node, HasSpan {
    type: "TsTypeReference";
    typeName: TsEntityName;
    typeParams?: TsTypeParameterInstantiation;
}
interface TsTypePredicate extends Node, HasSpan {
    type: "TsTypePredicate";
    asserts: boolean;
    paramName: TsThisTypeOrIdent;
    typeAnnotation?: TsTypeAnnotation;
}
type TsThisTypeOrIdent = TsThisType | Identifier;
interface TsImportType extends Node, HasSpan {
    type: "TsImportType";
    argument: StringLiteral;
    qualifier?: TsEntityName;
    typeArguments?: TsTypeParameterInstantiation;
}
/**
 * `typeof` operator
 */
interface TsTypeQuery extends Node, HasSpan {
    type: "TsTypeQuery";
    exprName: TsTypeQueryExpr;
    typeArguments?: TsTypeParameterInstantiation;
}
type TsTypeQueryExpr = TsEntityName | TsImportType;
interface TsTypeLiteral extends Node, HasSpan {
    type: "TsTypeLiteral";
    members: TsTypeElement[];
}
interface TsArrayType extends Node, HasSpan {
    type: "TsArrayType";
    elemType: TsType;
}
interface TsTupleType extends Node, HasSpan {
    type: "TsTupleType";
    elemTypes: TsTupleElement[];
}
interface TsTupleElement extends Node, HasSpan {
    type: "TsTupleElement";
    label?: Pattern;
    ty: TsType;
}
interface TsOptionalType extends Node, HasSpan {
    type: "TsOptionalType";
    typeAnnotation: TsType;
}
interface TsRestType extends Node, HasSpan {
    type: "TsRestType";
    typeAnnotation: TsType;
}
type TsUnionOrIntersectionType = TsUnionType | TsIntersectionType;
interface TsUnionType extends Node, HasSpan {
    type: "TsUnionType";
    types: TsType[];
}
interface TsIntersectionType extends Node, HasSpan {
    type: "TsIntersectionType";
    types: TsType[];
}
interface TsConditionalType extends Node, HasSpan {
    type: "TsConditionalType";
    checkType: TsType;
    extendsType: TsType;
    trueType: TsType;
    falseType: TsType;
}
interface TsInferType extends Node, HasSpan {
    type: "TsInferType";
    typeParam: TsTypeParameter;
}
interface TsParenthesizedType extends Node, HasSpan {
    type: "TsParenthesizedType";
    typeAnnotation: TsType;
}
interface TsTypeOperator extends Node, HasSpan {
    type: "TsTypeOperator";
    op: TsTypeOperatorOp;
    typeAnnotation: TsType;
}
type TsTypeOperatorOp = "keyof" | "unique" | "readonly";
interface TsIndexedAccessType extends Node, HasSpan {
    type: "TsIndexedAccessType";
    readonly: boolean;
    objectType: TsType;
    indexType: TsType;
}
type TruePlusMinus = true | "+" | "-";
interface TsMappedType extends Node, HasSpan {
    type: "TsMappedType";
    readonly?: TruePlusMinus;
    typeParam: TsTypeParameter;
    nameType?: TsType;
    optional?: TruePlusMinus;
    typeAnnotation?: TsType;
}
interface TsLiteralType extends Node, HasSpan {
    type: "TsLiteralType";
    literal: TsLiteral;
}
type TsLiteral = NumericLiteral | StringLiteral | BooleanLiteral | BigIntLiteral | TsTemplateLiteralType;
interface TsTemplateLiteralType extends Node, HasSpan {
    type: "TemplateLiteral";
    types: TsType[];
    quasis: TemplateElement[];
}
interface TsInterfaceDeclaration extends Node, HasSpan {
    type: "TsInterfaceDeclaration";
    id: Identifier;
    declare: boolean;
    typeParams?: TsTypeParameterDeclaration;
    extends: TsExpressionWithTypeArguments[];
    body: TsInterfaceBody;
}
interface TsInterfaceBody extends Node, HasSpan {
    type: "TsInterfaceBody";
    body: TsTypeElement[];
}
interface TsExpressionWithTypeArguments extends Node, HasSpan {
    type: "TsExpressionWithTypeArguments";
    expression: Expression;
    typeArguments?: TsTypeParameterInstantiation;
}
interface TsTypeAliasDeclaration extends Node, HasSpan {
    type: "TsTypeAliasDeclaration";
    declare: boolean;
    id: Identifier;
    typeParams?: TsTypeParameterDeclaration;
    typeAnnotation: TsType;
}
interface TsEnumDeclaration extends Node, HasSpan {
    type: "TsEnumDeclaration";
    declare: boolean;
    isConst: boolean;
    id: Identifier;
    members: TsEnumMember[];
}
interface TsEnumMember extends Node, HasSpan {
    type: "TsEnumMember";
    id: TsEnumMemberId;
    init?: Expression;
}
type TsEnumMemberId = Identifier | StringLiteral;
interface TsModuleDeclaration extends Node, HasSpan {
    type: "TsModuleDeclaration";
    declare: boolean;
    global: boolean;
    id: TsModuleName;
    body?: TsNamespaceBody;
}
/**
 * `namespace A.B { }` is a namespace named `A` with another TsNamespaceDecl as its body.
 */
type TsNamespaceBody = TsModuleBlock | TsNamespaceDeclaration;
interface TsModuleBlock extends Node, HasSpan {
    type: "TsModuleBlock";
    body: ModuleItem[];
}
interface TsNamespaceDeclaration extends Node, HasSpan {
    type: "TsNamespaceDeclaration";
    declare: boolean;
    global: boolean;
    id: Identifier;
    body: TsNamespaceBody;
}
type TsModuleName = Identifier | StringLiteral;
interface TsImportEqualsDeclaration extends Node, HasSpan {
    type: "TsImportEqualsDeclaration";
    declare: boolean;
    isExport: boolean;
    isTypeOnly: boolean;
    id: Identifier;
    moduleRef: TsModuleReference;
}
type TsModuleReference = TsEntityName | TsExternalModuleReference;
interface TsExternalModuleReference extends Node, HasSpan {
    type: "TsExternalModuleReference";
    expression: StringLiteral;
}
interface TsExportAssignment extends Node, HasSpan {
    type: "TsExportAssignment";
    expression: Expression;
}
interface TsNamespaceExportDeclaration extends Node, HasSpan {
    type: "TsNamespaceExportDeclaration";
    id: Identifier;
}
interface TsAsExpression extends ExpressionBase {
    type: "TsAsExpression";
    expression: Expression;
    typeAnnotation: TsType;
}
interface TsSatisfiesExpression extends ExpressionBase {
    type: "TsSatisfiesExpression";
    expression: Expression;
    typeAnnotation: TsType;
}
interface TsInstantiation extends Node, HasSpan {
    type: "TsInstantiation";
    expression: Expression;
    typeArguments: TsTypeParameterInstantiation;
}
interface TsTypeAssertion extends ExpressionBase {
    type: "TsTypeAssertion";
    expression: Expression;
    typeAnnotation: TsType;
}
interface TsConstAssertion extends ExpressionBase {
    type: "TsConstAssertion";
    expression: Expression;
}
interface TsNonNullExpression extends ExpressionBase {
    type: "TsNonNullExpression";
    expression: Expression;
}
type Accessibility = "public" | "protected" | "private";
interface Invalid extends Node, HasSpan {
    type: "Invalid";
}

interface Options$1 {
    allowRegExp: boolean;
    allowFunction: boolean;
    allowSymbol: boolean;
    allowDate: boolean;
    allowUndefined: boolean;
    allowClass: boolean;
    allowError: boolean;
    maxDepth: number;
    space: number | undefined;
    lazyEval: boolean;
}

type ExportName = string;
type MetaId = string;
interface StoriesSpecifier {
    /**
     * When auto-titling, what to prefix all generated titles with (default: '')
     */
    titlePrefix?: string;
    /**
     * Where to start looking for story files
     */
    directory: string;
    /**
     * What does the filename of a story file look like?
     * (a glob, relative to directory, no leading `./`)
     * If unset, we use `** / *.@(mdx|stories.@(mdx|js|jsx|mjs|ts|tsx))` (no spaces)
     */
    files?: string;
}
type StoriesEntry = string | StoriesSpecifier;
interface IndexerOptions {
    makeTitle: (userTitle?: string) => string;
}
interface IndexedStory {
    id: string;
    name: string;
    tags?: Tag[];
    parameters?: Parameters;
}
interface IndexedCSFFile {
    meta: {
        id?: string;
        title?: string;
        tags?: Tag[];
    };
    stories: IndexedStory[];
}
/**
 * FIXME: This is a temporary type to allow us to deprecate the old indexer API.
 * We should remove this type and the deprecated indexer API in 8.0.
 */
type BaseIndexer = {
    /**
     * A regular expression that should match all files to be handled by this indexer
     */
    test: RegExp;
};
/**
 * An indexer describes which filenames it handles, and how to index each individual file - turning it into an entry in the index.
 */
type Indexer = BaseIndexer & {
    /**
     * Indexes a file containing stories or docs.
     * @param fileName The name of the file to index.
     * @param options {@link IndexerOptions} for indexing the file.
     * @returns A promise that resolves to an array of {@link IndexInput} objects.
     */
    createIndex: (fileName: string, options: IndexerOptions) => Promise<IndexInput[]>;
    /**
     * @deprecated Use {@link index} instead
     */
    indexer?: never;
};
type DeprecatedIndexer = BaseIndexer & {
    indexer: (fileName: string, options: IndexerOptions) => Promise<IndexedCSFFile>;
    createIndex?: never;
};
/**
 * @deprecated Use {@link Indexer} instead
 */
type StoryIndexer = Indexer | DeprecatedIndexer;
/**
 * The base input for indexing a story or docs entry.
 */
type BaseIndexInput = {
    /** The file to import from e.g. the story file. */
    importPath: Path;
    /** The name of the export to import. */
    exportName: ExportName;
    /** The name of the entry, auto-generated from {@link exportName} if unspecified. */
    name?: StoryName;
    /** The location in the sidebar, auto-generated from {@link importPath} if unspecified. */
    title?: ComponentTitle;
    /**
     * The custom id optionally set at `meta.id` if it needs to differ from the id generated via {@link title}.
     * If unspecified, the meta id will be auto-generated from {@link title}.
     * If specified, the meta in the CSF file _must_ have a matching id set at `meta.id`, to be correctly matched.
     */
    metaId?: MetaId;
    /** Tags for filtering entries in Storybook and its tools. */
    tags?: Tag[];
    /**
     * The id of the entry, auto-generated from {@link title}/{@link metaId} and {@link exportName} if unspecified.
     * If specified, the story in the CSF file _must_ have a matching id set at `parameters.__id`, to be correctly matched.
     * Only use this if you need to override the auto-generated id.
     */
    __id?: StoryId;
};
/**
 * The input for indexing a story entry.
 */
type StoryIndexInput = BaseIndexInput & {
    type: 'story';
};
/**
 * The input for indexing a docs entry.
 */
type DocsIndexInput = BaseIndexInput & {
    type: 'docs';
    /** Paths to story files that must be pre-loaded for this docs entry. */
    storiesImports?: Path[];
};
type IndexInput = StoryIndexInput | DocsIndexInput;

/**
 * ⚠️ This file contains internal WIP types they MUST NOT be exported outside this package for now!
 */
type BuilderName = 'webpack5' | '@storybook/builder-webpack5' | string;
type RendererName = string;
interface CoreConfig {
    builder?: BuilderName | {
        name: BuilderName;
        options?: Record<string, any>;
    };
    renderer?: RendererName;
    disableWebpackDefaults?: boolean;
    channelOptions?: Partial<Options$1>;
    /**
     * Disables the generation of project.json, a file containing Storybook metadata
     */
    disableProjectJson?: boolean;
    /**
     * Disables Storybook telemetry
     * @see https://storybook.js.org/telemetry
     */
    disableTelemetry?: boolean;
    /**
     * Disables notifications for Storybook updates.
     */
    disableWhatsNewNotifications?: boolean;
    /**
     * Enable crash reports to be sent to Storybook telemetry
     * @see https://storybook.js.org/telemetry
     */
    enableCrashReports?: boolean;
    /**
     * enable CORS headings to run document in a "secure context"
     * see: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/SharedArrayBuffer#security_requirements
     * This enables these headers in development-mode:
     *   Cross-Origin-Opener-Policy: same-origin
     *   Cross-Origin-Embedder-Policy: require-corp
     */
    crossOriginIsolated?: boolean;
}
interface DirectoryMapping {
    from: string;
    to: string;
}
interface Presets {
    apply(extension: 'typescript', config: Partial<TypescriptOptions>, args?: Options$3): Promise<Partial<TypescriptOptions>>;
    apply(extension: 'framework', config?: {}, args?: any): Promise<Preset>;
    apply(extension: 'babel', config?: {}, args?: any): Promise<TransformOptions>;
    apply(extension: 'swc', config?: {}, args?: any): Promise<Options$2>;
    apply(extension: 'entries', config?: [], args?: any): Promise<unknown>;
    apply(extension: 'stories', config?: [], args?: any): Promise<StoriesEntry[]>;
    apply(extension: 'managerEntries', config: [], args?: any): Promise<string[]>;
    apply(extension: 'refs', config?: [], args?: any): Promise<unknown>;
    apply(extension: 'core', config?: {}, args?: any): Promise<CoreConfig>;
    apply(extension: 'build', config?: {}, args?: any): Promise<StorybookConfig['build']>;
    apply<T>(extension: string, config?: T, args?: unknown): Promise<T>;
}
interface LoadedPreset {
    name: string;
    preset: any;
    options: any;
}
interface VersionCheck {
    success: boolean;
    cached: boolean;
    data?: any;
    error?: any;
    time: number;
}
type PackageJson = PackageJson$1 & Record<string, any>;
interface LoadOptions {
    packageJson: PackageJson;
    outputDir?: string;
    configDir?: string;
    ignorePreview?: boolean;
    extendServer?: (server: Server) => void;
}
interface CLIOptions {
    port?: number;
    ignorePreview?: boolean;
    previewUrl?: string;
    forceBuildPreview?: boolean;
    disableTelemetry?: boolean;
    enableCrashReports?: boolean;
    host?: string;
    initialPath?: string;
    /**
     * @deprecated Use 'staticDirs' Storybook Configuration option instead
     */
    staticDir?: string[];
    configDir?: string;
    https?: boolean;
    sslCa?: string[];
    sslCert?: string;
    sslKey?: string;
    smokeTest?: boolean;
    managerCache?: boolean;
    open?: boolean;
    ci?: boolean;
    loglevel?: string;
    quiet?: boolean;
    versionUpdates?: boolean;
    docs?: boolean;
    test?: boolean;
    debugWebpack?: boolean;
    webpackStatsJson?: string | boolean;
    outputDir?: string;
}
interface BuilderOptions {
    configType?: 'DEVELOPMENT' | 'PRODUCTION';
    ignorePreview?: boolean;
    cache?: FileSystemCache;
    configDir: string;
    docsMode?: boolean;
    features?: StorybookConfig['features'];
    versionCheck?: VersionCheck;
    disableWebpackDefaults?: boolean;
    serverChannelUrl?: string;
}
interface StorybookConfigOptions {
    presets: Presets;
    presetsList?: LoadedPreset[];
}
type Options$3 = LoadOptions & StorybookConfigOptions & CLIOptions & BuilderOptions & {
    build?: TestBuildConfig;
};
/**
 * Options for TypeScript usage within Storybook.
 */
interface TypescriptOptions {
    /**
     * Enables type checking within Storybook.
     *
     * @default `false`
     */
    check: boolean;
    /**
     * Disable parsing typescript files through babel.
     *
     * @default `false`
     * @deprecated use `skipCompiler` instead
     */
    skipBabel: boolean;
    /**
     * Disable parsing typescript files through compiler.
     *
     * @default `false`
     */
    skipCompiler: boolean;
}
type Preset = string | {
    name: string;
    options?: any;
};
/**
 * An additional script that gets injected into the
 * preview or the manager,
 */
type Entry = string;
type CoreCommon_StorybookRefs = Record<string, {
    title: string;
    url: string;
} | {
    disable: boolean;
    expanded?: boolean;
}>;
type DocsOptions = {
    /**
     * What should we call the generated docs entries?
     */
    defaultName?: string;
    /**
     * Should we generate a docs entry per CSF file?
     * Set to 'tag' (the default) to generate an entry for every CSF file with the
     * 'autodocs' tag.
     */
    autodocs?: boolean | 'tag';
    /**
     * Only show doc entries in the side bar (usually set with the `--docs` CLI flag)
     */
    docsMode?: boolean;
};
interface TestBuildFlags {
    /**
     * The package @storybook/blocks will be excluded from the bundle, even when imported in e.g. the preview.
     */
    disableBlocks?: boolean;
    /**
     * Disable specific addons
     */
    disabledAddons?: string[];
    /**
     * Filter out .mdx stories entries
     */
    disableMDXEntries?: boolean;
    /**
     * Override autodocs to be disabled
     */
    disableAutoDocs?: boolean;
    /**
     * Override docgen to be disabled.
     */
    disableDocgen?: boolean;
    /**
     * Override sourcemaps generation to be disabled.
     */
    disableSourcemaps?: boolean;
    /**
     * Override tree-shaking (dead code elimination) to be disabled.
     */
    disableTreeShaking?: boolean;
    /**
     * Minify with ESBuild when using webpack.
     */
    esbuildMinify?: boolean;
}
interface TestBuildConfig {
    test?: TestBuildFlags;
}
/**
 * The interface for Storybook configuration in `main.ts` files.
 */
interface StorybookConfig {
    /**
     * Sets the addons you want to use with Storybook.
     *
     * @example `['@storybook/addon-essentials']` or `[{ name: '@storybook/addon-essentials', options: { backgrounds: false } }]`
     */
    addons?: Preset[];
    core?: CoreConfig;
    /**
     * Sets a list of directories of static files to be loaded by Storybook server
     *
     * @example `['./public']` or `[{from: './public', 'to': '/assets'}]`
     */
    staticDirs?: (DirectoryMapping | string)[];
    logLevel?: string;
    features?: {
        /**
         * Build stories.json automatically on start/build
         */
        buildStoriesJson?: boolean;
        /**
         * Activate on demand story store
         */
        storyStoreV7?: boolean;
        /**
         * Do not throw errors if using `.mdx` files in SSv7
         * (for internal use in sandboxes)
         */
        storyStoreV7MdxErrors?: boolean;
        /**
         * Filter args with a "target" on the type from the render function (EXPERIMENTAL)
         */
        argTypeTargetsV7?: boolean;
        /**
         * Warn when there is a pre-6.0 hierarchy separator ('.' / '|') in the story title.
         * Will be removed in 7.0.
         */
        warnOnLegacyHierarchySeparator?: boolean;
        /**
         * Use legacy MDX1, to help smooth migration to 7.0
         */
        legacyMdx1?: boolean;
        /**
         * Apply decorators from preview.js before decorators from addons or frameworks
         */
        legacyDecoratorFileOrder?: boolean;
        /**
         * Disallow implicit actions during rendering. This will be the default in Storybook 8.
         *
         * This will make sure that your story renders the same no matter if docgen is enabled or not.
         */
        disallowImplicitActionsInRenderV8?: boolean;
    };
    build?: TestBuildConfig;
    /**
     * Tells Storybook where to find stories.
     *
     * @example `['./src/*.stories.@(j|t)sx?']`
     */
    stories: StoriesEntry[];
    /**
     * Framework, e.g. '@storybook/react-vite', required in v7
     */
    framework?: Preset;
    /**
     * Controls how Storybook handles TypeScript files.
     */
    typescript?: Partial<TypescriptOptions>;
    /**
     * References external Storybooks
     */
    refs?: PresetValue<CoreCommon_StorybookRefs>;
    /**
     * Modify or return babel config.
     */
    babel?: (config: TransformOptions, options: Options$3) => TransformOptions | Promise<TransformOptions>;
    /**
     * Modify or return swc config.
     */
    swc?: (config: Options$2, options: Options$3) => Options$2 | Promise<Options$2>;
    /**
     * Modify or return env config.
     */
    env?: PresetValue<Record<string, string>>;
    /**
     * Modify or return babel config.
     */
    babelDefault?: (config: TransformOptions, options: Options$3) => TransformOptions | Promise<TransformOptions>;
    /**
     * Add additional scripts to run in the preview a la `.storybook/preview.js`
     *
     * @deprecated use `previewAnnotations` or `/preview.js` file instead
     */
    config?: PresetValue<Entry[]>;
    /**
     * Add additional scripts to run in the preview a la `.storybook/preview.js`
     */
    previewAnnotations?: PresetValue<Entry[]>;
    /**
     * Process CSF files for the story index.
     * @deprecated use {@link experimental_indexers} instead
     */
    storyIndexers?: PresetValue<StoryIndexer[]>;
    /**
     * Process CSF files for the story index.
     */
    experimental_indexers?: PresetValue<Indexer[]>;
    /**
     * Docs related features in index generation
     */
    docs?: DocsOptions;
    /**
     * Programmatically modify the preview head/body HTML.
     * The previewHead and previewBody functions accept a string,
     * which is the existing head/body, and return a modified string.
     */
    previewHead?: PresetValue<string>;
    previewBody?: PresetValue<string>;
    /**
     * Programmatically override the preview's main page template.
     * This should return a reference to a file containing an `.ejs` template
     * that will be interpolated with environment variables.
     *
     * @example '.storybook/index.ejs'
     */
    previewMainTemplate?: string;
    /**
     * Programmatically modify the preview head/body HTML.
     * The managerHead function accept a string,
     * which is the existing head content, and return a modified string.
     */
    managerHead?: PresetValue<string>;
}
type PresetValue<T> = T | ((config: T, options: Options$3) => T | Promise<T>);
type Path = string;

declare const monorepoConfigs: {
    readonly Nx: "nx.json";
    readonly Turborepo: "turbo.json";
    readonly Lerna: "lerna.json";
    readonly Rush: "rush.json";
    readonly Lage: "lage.config.json";
};
type MonorepoType = keyof typeof monorepoConfigs | 'Workspaces' | undefined;

type EventType = 'boot' | 'dev' | 'build' | 'upgrade' | 'init' | 'browser' | 'canceled' | 'error' | 'error-metadata' | 'version-update' | 'core-config';
interface Dependency {
    version: string | undefined;
    versionSpecifier?: string;
}
interface StorybookAddon extends Dependency {
    options: any;
}
type StorybookMetadata = {
    storybookVersion?: string;
    storybookVersionSpecifier: string;
    generatedAt?: number;
    language: 'typescript' | 'javascript';
    framework?: {
        name: string;
        options?: any;
    };
    builder?: string;
    renderer?: string;
    monorepo?: MonorepoType;
    packageManager?: {
        type: PM;
        version: string;
    };
    typescriptOptions?: Partial<TypescriptOptions>;
    addons?: Record<string, StorybookAddon>;
    storybookPackages?: Record<string, Dependency>;
    metaFramework?: {
        name: string;
        packageName: string;
        version: string;
    };
    hasStorybookEslint?: boolean;
    hasStaticDirs?: boolean;
    hasCustomWebpack?: boolean;
    hasCustomBabel?: boolean;
    features?: StorybookConfig['features'];
    refCount?: number;
    preview?: {
        usesGlobals?: boolean;
    };
};
interface Payload {
    [key: string]: any;
}
interface Options {
    retryDelay: number;
    immediate: boolean;
    configDir?: string;
    enableCrashReports?: boolean;
    stripMetadata?: boolean;
}
interface TelemetryData {
    eventType: EventType;
    payload: Payload;
    metadata?: StorybookMetadata;
}

declare const oneWayHash: (payload: BinaryLike) => string;

declare const metaFrameworks: Record<string, string>;
declare const sanitizeAddonName: (name: string) => string;
declare const computeStorybookMetadata: ({ packageJson, mainConfig, }: {
    packageJson: PackageJson;
    mainConfig: StorybookConfig & Record<string, any>;
}) => Promise<StorybookMetadata>;
declare const getStorybookMetadata: (_configDir?: string) => Promise<StorybookMetadata>;

interface UpgradeSummary {
    timestamp: number;
    eventType?: EventType;
    eventId?: string;
    sessionId?: string;
}
declare const getPrecedingUpgrade: (events?: any) => Promise<UpgradeSummary | undefined>;

declare const addToGlobalContext: (key: string, value: any) => void;

declare const telemetry: (eventType: EventType, payload?: Payload, options?: Partial<Options>) => Promise<void>;

export { Dependency, EventType, Options, Payload, StorybookAddon, StorybookMetadata, TelemetryData, addToGlobalContext, computeStorybookMetadata, getPrecedingUpgrade, getStorybookMetadata, metaFrameworks, oneWayHash, sanitizeAddonName, telemetry };
