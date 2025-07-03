import { ReactElement, ReactNode, WeakValidationMap, ValidationMap, FC, PropsWithChildren } from 'react';
import { FileSystemCache } from 'file-system-cache';
import { TransformOptions } from '@babel/core';
import { Router } from 'express';
import { Server } from 'http';
import { Channel as Channel$1 } from '@storybook/channels';

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
@see Simplify
*/
interface SimplifyOptions {
	/**
	Do the simplification recursively.

	@default false
	*/
	deep?: boolean;
}

// Flatten a type without worrying about the result.
type Flatten<
	AnyType,
	Options extends SimplifyOptions = {},
> = Options['deep'] extends true
	? {[KeyType in keyof AnyType]: Simplify<AnyType[KeyType], Options>}
	: {[KeyType in keyof AnyType]: AnyType[KeyType]};

/**
Useful to flatten the type output to improve type hints shown in editors. And also to transform an interface into a type to aide with assignability.

@example
```
import type {Simplify} from 'type-fest';

type PositionProps = {
	top: number;
	left: number;
};

type SizeProps = {
	width: number;
	height: number;
};

// In your editor, hovering over `Props` will show a flattened object with all the properties.
type Props = Simplify<PositionProps & SizeProps>;
```

Sometimes it is desired to pass a value as a function argument that has a different type. At first inspection it may seem assignable, and then you discover it is not because the `value`'s type definition was defined as an interface. In the following example, `fn` requires an argument of type `Record<string, unknown>`. If the value is defined as a literal, then it is assignable. And if the `value` is defined as type using the `Simplify` utility the value is assignable.  But if the `value` is defined as an interface, it is not assignable because the interface is not sealed and elsewhere a non-string property could be added to the interface.

If the type definition must be an interface (perhaps it was defined in a third-party npm package), then the `value` can be defined as `const value: Simplify<SomeInterface> = ...`. Then `value` will be assignable to the `fn` argument.  Or the `value` can be cast as `Simplify<SomeInterface>` if you can't re-declare the `value`.

@example
```
import type {Simplify} from 'type-fest';

interface SomeInterface {
	foo: number;
	bar?: string;
	baz: number | undefined;
}

type SomeType = {
	foo: number;
	bar?: string;
	baz: number | undefined;
};

const literal = {foo: 123, bar: 'hello', baz: 456};
const someType: SomeType = literal;
const someInterface: SomeInterface = literal;

function fn(object: Record<string, unknown>): void {}

fn(literal); // Good: literal object type is sealed
fn(someType); // Good: type is sealed
fn(someInterface); // Error: Index signature for type 'string' is missing in type 'someInterface'. Because `interface` can be re-opened
fn(someInterface as Simplify<SomeInterface>); // Good: transform an `interface` into a `type`
```

@link https://github.com/microsoft/TypeScript/issues/15300

@category Object
*/
type Simplify<
	AnyType,
	Options extends SimplifyOptions = {},
> = Flatten<AnyType> extends AnyType
	? Flatten<AnyType, Options>
	: AnyType;

/**
Remove any index signatures from the given object type, so that only explicitly defined properties remain.

Use-cases:
- Remove overly permissive signatures from third-party types.

This type was taken from this [StackOverflow answer](https://stackoverflow.com/a/68261113/420747).

It relies on the fact that an empty object (`{}`) is assignable to an object with just an index signature, like `Record<string, unknown>`, but not to an object with explicitly defined keys, like `Record<'foo' | 'bar', unknown>`.

(The actual value type, `unknown`, is irrelevant and could be any type. Only the key type matters.)

```
const indexed: Record<string, unknown> = {}; // Allowed

const keyed: Record<'foo', unknown> = {}; // Error
// => TS2739: Type '{}' is missing the following properties from type 'Record<"foo" | "bar", unknown>': foo, bar
```

Instead of causing a type error like the above, you can also use a [conditional type](https://www.typescriptlang.org/docs/handbook/2/conditional-types.html) to test whether a type is assignable to another:

```
type Indexed = {} extends Record<string, unknown>
	? '✅ `{}` is assignable to `Record<string, unknown>`'
	: '❌ `{}` is NOT assignable to `Record<string, unknown>`';
// => '✅ `{}` is assignable to `Record<string, unknown>`'

type Keyed = {} extends Record<'foo' | 'bar', unknown>
	? "✅ `{}` is assignable to `Record<'foo' | 'bar', unknown>`"
	: "❌ `{}` is NOT assignable to `Record<'foo' | 'bar', unknown>`";
// => "❌ `{}` is NOT assignable to `Record<'foo' | 'bar', unknown>`"
```

Using a [mapped type](https://www.typescriptlang.org/docs/handbook/2/mapped-types.html#further-exploration), you can then check for each `KeyType` of `ObjectType`...

```
import type {RemoveIndexSignature} from 'type-fest';

type RemoveIndexSignature<ObjectType> = {
	[KeyType in keyof ObjectType // Map each key of `ObjectType`...
	]: ObjectType[KeyType]; // ...to its original value, i.e. `RemoveIndexSignature<Foo> == Foo`.
};
```

...whether an empty object (`{}`) would be assignable to an object with that `KeyType` (`Record<KeyType, unknown>`)...

```
import type {RemoveIndexSignature} from 'type-fest';

type RemoveIndexSignature<ObjectType> = {
	[KeyType in keyof ObjectType
		// Is `{}` assignable to `Record<KeyType, unknown>`?
		as {} extends Record<KeyType, unknown>
			? ... // ✅ `{}` is assignable to `Record<KeyType, unknown>`
			: ... // ❌ `{}` is NOT assignable to `Record<KeyType, unknown>`
	]: ObjectType[KeyType];
};
```

If `{}` is assignable, it means that `KeyType` is an index signature and we want to remove it. If it is not assignable, `KeyType` is a "real" key and we want to keep it.

```
import type {RemoveIndexSignature} from 'type-fest';

type RemoveIndexSignature<ObjectType> = {
	[KeyType in keyof ObjectType
		as {} extends Record<KeyType, unknown>
			? never // => Remove this `KeyType`.
			: KeyType // => Keep this `KeyType` as it is.
	]: ObjectType[KeyType];
};
```

@example
```
import type {RemoveIndexSignature} from 'type-fest';

interface Example {
	// These index signatures will be removed.
	[x: string]: any
	[x: number]: any
	[x: symbol]: any
	[x: `head-${string}`]: string
	[x: `${string}-tail`]: string
	[x: `head-${string}-tail`]: string
	[x: `${bigint}`]: string
	[x: `embedded-${number}`]: string

	// These explicitly defined keys will remain.
	foo: 'bar';
	qux?: 'baz';
}

type ExampleWithoutIndexSignatures = RemoveIndexSignature<Example>;
// => { foo: 'bar'; qux?: 'baz' | undefined; }
```

@category Object
*/
type RemoveIndexSignature<ObjectType> = {
	[KeyType in keyof ObjectType as {} extends Record<KeyType, unknown>
		? never
		: KeyType]: ObjectType[KeyType];
};

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

/**
Convert a union type to an intersection type using [distributive conditional types](https://www.typescriptlang.org/docs/handbook/release-notes/typescript-2-8.html#distributive-conditional-types).

Inspired by [this Stack Overflow answer](https://stackoverflow.com/a/50375286/2172153).

@example
```
import type {UnionToIntersection} from 'type-fest';

type Union = {the(): void} | {great(arg: string): void} | {escape: boolean};

type Intersection = UnionToIntersection<Union>;
//=> {the(): void; great(arg: string): void; escape: boolean};
```

A more applicable example which could make its way into your library code follows.

@example
```
import type {UnionToIntersection} from 'type-fest';

class CommandOne {
	commands: {
		a1: () => undefined,
		b1: () => undefined,
	}
}

class CommandTwo {
	commands: {
		a2: (argA: string) => undefined,
		b2: (argB: string) => undefined,
	}
}

const union = [new CommandOne(), new CommandTwo()].map(instance => instance.commands);
type Union = typeof union;
//=> {a1(): void; b1(): void} | {a2(argA: string): void; b2(argB: string): void}

type Intersection = UnionToIntersection<Union>;
//=> {a1(): void; b1(): void; a2(argA: string): void; b2(argB: string): void}
```

@category Type
*/
type UnionToIntersection<Union> = (
	// `extends unknown` is always going to be the case and is used to convert the
	// `Union` into a [distributive conditional
	// type](https://www.typescriptlang.org/docs/handbook/release-notes/typescript-2-8.html#distributive-conditional-types).
	Union extends unknown
		// The union type is used as the only argument to a function since the union
		// of function arguments is an intersection.
		? (distributedUnion: Union) => void
		// This won't happen.
		: never
		// Infer the `Intersection` type since TypeScript represents the positional
		// arguments of unions of functions as an intersection of the union.
	) extends ((mergedIntersection: infer Intersection) => void)
		? Intersection
		: never;

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

interface SBBaseType {
    required?: boolean;
    raw?: string;
}
type SBScalarType = SBBaseType & {
    name: 'boolean' | 'string' | 'number' | 'function' | 'symbol';
};
type SBArrayType = SBBaseType & {
    name: 'array';
    value: SBType;
};
type SBObjectType = SBBaseType & {
    name: 'object';
    value: Record<string, SBType>;
};
type SBEnumType = SBBaseType & {
    name: 'enum';
    value: (string | number)[];
};
type SBIntersectionType = SBBaseType & {
    name: 'intersection';
    value: SBType[];
};
type SBUnionType = SBBaseType & {
    name: 'union';
    value: SBType[];
};
type SBOtherType = SBBaseType & {
    name: 'other';
    value: string;
};
type SBType = SBScalarType | SBEnumType | SBArrayType | SBObjectType | SBIntersectionType | SBUnionType | SBOtherType;

type StoryId = string;
type ComponentId = string;
type ComponentTitle = string;
type StoryName = string;
/** @deprecated */
type StoryKind = ComponentTitle;
type Tag = string;
interface StoryIdentifier {
    componentId: ComponentId;
    title: ComponentTitle;
    /** @deprecated */
    kind: ComponentTitle;
    id: StoryId;
    name: StoryName;
    /** @deprecated */
    story: StoryName;
    tags: Tag[];
}
interface Parameters {
    [name: string]: any;
}
type ConditionalTest = {
    truthy?: boolean;
} | {
    exists: boolean;
} | {
    eq: any;
} | {
    neq: any;
};
type ConditionalValue = {
    arg: string;
} | {
    global: string;
};
type Conditional = ConditionalValue & ConditionalTest;
interface InputType {
    name?: string;
    description?: string;
    defaultValue?: any;
    type?: SBType | SBScalarType['name'];
    if?: Conditional;
    [key: string]: any;
}
interface StrictInputType extends InputType {
    name: string;
    type?: SBType;
}
interface Args {
    [name: string]: any;
}
interface StrictArgs {
    [name: string]: unknown;
}
type ArgTypes<TArgs = Args> = {
    [name in keyof TArgs]: InputType;
};
type StrictArgTypes<TArgs = Args> = {
    [name in keyof TArgs]: StrictInputType;
};
interface Globals {
    [name: string]: any;
}
interface GlobalTypes {
    [name: string]: InputType;
}
interface StrictGlobalTypes {
    [name: string]: StrictInputType;
}
type Renderer = {
    /** What is the type of the `component` annotation in this renderer? */
    component: unknown;
    /** What does the story function return in this renderer? */
    storyResult: unknown;
    /** What type of element does this renderer render to? */
    canvasElement: unknown;
    T?: unknown;
};
interface StoryContextForEnhancers<TRenderer extends Renderer = Renderer, TArgs = Args> extends StoryIdentifier {
    component?: (TRenderer & {
        T: any;
    })['component'];
    subcomponents?: Record<string, (TRenderer & {
        T: any;
    })['component']>;
    parameters: Parameters;
    initialArgs: TArgs;
    argTypes: StrictArgTypes<TArgs>;
}
type ArgsEnhancer<TRenderer extends Renderer = Renderer, TArgs = Args> = (context: StoryContextForEnhancers<TRenderer, TArgs>) => TArgs;
type ArgTypesEnhancer<TRenderer extends Renderer = Renderer, TArgs = Args> = ((context: StoryContextForEnhancers<TRenderer, TArgs>) => StrictArgTypes<TArgs>) & {
    secondPass?: boolean;
};
interface StoryContextUpdate<TArgs = Args> {
    args?: TArgs;
    globals?: Globals;
    [key: string]: any;
}
type ViewMode$1 = 'story' | 'docs';
interface StoryContextForLoaders<TRenderer extends Renderer = Renderer, TArgs = Args> extends StoryContextForEnhancers<TRenderer, TArgs>, Required<StoryContextUpdate<TArgs>> {
    hooks: unknown;
    viewMode: ViewMode$1;
    originalStoryFn: StoryFn<TRenderer>;
}
type LoaderFunction<TRenderer extends Renderer = Renderer, TArgs = Args> = (context: StoryContextForLoaders<TRenderer, TArgs>) => Promise<Record<string, any> | void> | Record<string, any> | void;
interface StoryContext<TRenderer extends Renderer = Renderer, TArgs = Args> extends StoryContextForLoaders<TRenderer, TArgs> {
    loaded: Record<string, any>;
    abortSignal: AbortSignal;
    canvasElement: TRenderer['canvasElement'];
}
type StepLabel = string;
type StepFunction<TRenderer extends Renderer = Renderer, TArgs = Args> = (label: StepLabel, play: PlayFunction<TRenderer, TArgs>) => Promise<void> | void;
type PlayFunctionContext<TRenderer extends Renderer = Renderer, TArgs = Args> = StoryContext<TRenderer, TArgs> & {
    step: StepFunction<TRenderer, TArgs>;
};
type PlayFunction<TRenderer extends Renderer = Renderer, TArgs = Args> = (context: PlayFunctionContext<TRenderer, TArgs>) => Promise<void> | void;
type PartialStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = (update?: StoryContextUpdate<Partial<TArgs>>) => TRenderer['storyResult'];
type LegacyStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = (context: StoryContext<TRenderer, TArgs>) => TRenderer['storyResult'];
type ArgsStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = (args: TArgs, context: StoryContext<TRenderer, TArgs>) => (TRenderer & {
    T: TArgs;
})['storyResult'];
type StoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = LegacyStoryFn<TRenderer, TArgs> | ArgsStoryFn<TRenderer, TArgs>;
type DecoratorFunction<TRenderer extends Renderer = Renderer, TArgs = Args> = (fn: PartialStoryFn<TRenderer, TArgs>, c: StoryContext<TRenderer, TArgs>) => TRenderer['storyResult'];
type DecoratorApplicator<TRenderer extends Renderer = Renderer, TArgs = Args> = (storyFn: LegacyStoryFn<TRenderer, TArgs>, decorators: DecoratorFunction<TRenderer, TArgs>[]) => LegacyStoryFn<TRenderer, TArgs>;
type StepRunner<TRenderer extends Renderer = Renderer, TArgs = Args> = (label: StepLabel, play: PlayFunction<TRenderer, TArgs>, context: PlayFunctionContext<TRenderer, TArgs>) => Promise<void>;
type BaseAnnotations<TRenderer extends Renderer = Renderer, TArgs = Args> = {
    /**
     * Wrapper components or Storybook decorators that wrap a story.
     *
     * Decorators defined in Meta will be applied to every story variation.
     * @see [Decorators](https://storybook.js.org/docs/addons/introduction/#1-decorators)
     */
    decorators?: DecoratorFunction<TRenderer, Simplify<TArgs>>[] | DecoratorFunction<TRenderer, Simplify<TArgs>>;
    /**
     * Custom metadata for a story.
     * @see [Parameters](https://storybook.js.org/docs/basics/writing-stories/#parameters)
     */
    parameters?: Parameters;
    /**
     * Dynamic data that are provided (and possibly updated by) Storybook and its addons.
     * @see [Arg story inputs](https://storybook.js.org/docs/react/api/csf#args-story-inputs)
     */
    args?: Partial<TArgs>;
    /**
     * ArgTypes encode basic metadata for args, such as `name`, `description`, `defaultValue` for an arg. These get automatically filled in by Storybook Docs.
     * @see [Control annotations](https://github.com/storybookjs/storybook/blob/91e9dee33faa8eff0b342a366845de7100415367/addons/controls/README.md#control-annotations)
     */
    argTypes?: Partial<ArgTypes<TArgs>>;
    /**
     * Asynchronous functions which provide data for a story.
     * @see [Loaders](https://storybook.js.org/docs/react/writing-stories/loaders)
     */
    loaders?: LoaderFunction<TRenderer, TArgs>[] | LoaderFunction<TRenderer, TArgs>;
    /**
     * Define a custom render function for the story(ies). If not passed, a default render function by the renderer will be used.
     */
    render?: ArgsStoryFn<TRenderer, TArgs>;
};
type ProjectAnnotations$1<TRenderer extends Renderer = Renderer, TArgs = Args> = BaseAnnotations<TRenderer, TArgs> & {
    argsEnhancers?: ArgsEnhancer<TRenderer, Args>[];
    argTypesEnhancers?: ArgTypesEnhancer<TRenderer, Args>[];
    globals?: Globals;
    globalTypes?: GlobalTypes;
    applyDecorators?: DecoratorApplicator<TRenderer, Args>;
    runStep?: StepRunner<TRenderer, TArgs>;
};
type StoryDescriptor$1 = string[] | RegExp;
interface ComponentAnnotations<TRenderer extends Renderer = Renderer, TArgs = Args> extends BaseAnnotations<TRenderer, TArgs> {
    /**
     * Title of the component which will be presented in the navigation. **Should be unique.**
     *
     * Components can be organized in a nested structure using "/" as a separator.
     *
     * Since CSF 3.0 this property is optional -- it can be inferred from the filesystem path
     *
     * @example
     * export default {
     *   ...
     *   title: 'Design System/Atoms/Button'
     * }
     *
     * @see [Story Hierarchy](https://storybook.js.org/docs/basics/writing-stories/#story-hierarchy)
     */
    title?: ComponentTitle;
    /**
     * Id of the component (prefix of the story id) which is used for URLs.
     *
     * By default is inferred from sanitizing the title
     *
     * @see [Story Hierarchy](https://storybook.js.org/docs/basics/writing-stories/#story-hierarchy)
     */
    id?: ComponentId;
    /**
     * Used to only include certain named exports as stories. Useful when you want to have non-story exports such as mock data or ignore a few stories.
     * @example
     * includeStories: ['SimpleStory', 'ComplexStory']
     * includeStories: /.*Story$/
     *
     * @see [Non-story exports](https://storybook.js.org/docs/formats/component-story-format/#non-story-exports)
     */
    includeStories?: StoryDescriptor$1;
    /**
     * Used to exclude certain named exports. Useful when you want to have non-story exports such as mock data or ignore a few stories.
     * @example
     * excludeStories: ['simpleData', 'complexData']
     * excludeStories: /.*Data$/
     *
     * @see [Non-story exports](https://storybook.js.org/docs/formats/component-story-format/#non-story-exports)
     */
    excludeStories?: StoryDescriptor$1;
    /**
     * The primary component for your story.
     *
     * Used by addons for automatic prop table generation and display of other component metadata.
     */
    component?: (TRenderer & {
        T: Record<string, unknown> extends Required<TArgs> ? any : TArgs;
    })['component'];
    /**
     * Auxiliary subcomponents that are part of the stories.
     *
     * Used by addons for automatic prop table generation and display of other component metadata.
     *
     * @example
     * import { Button, ButtonGroup } from './components';
     *
     * export default {
     *   ...
     *   subcomponents: { Button, ButtonGroup }
     * }
     *
     * By defining them each component will have its tab in the args table.
     */
    subcomponents?: Record<string, TRenderer['component']>;
    /**
     * Function that is executed after the story is rendered.
     */
    play?: PlayFunction<TRenderer, TArgs>;
    /**
     * Named tags for a story, used to filter stories in different contexts.
     */
    tags?: Tag[];
}
type StoryAnnotations<TRenderer extends Renderer = Renderer, TArgs = Args, TRequiredArgs = Partial<TArgs>> = BaseAnnotations<TRenderer, TArgs> & {
    /**
     * Override the display name in the UI (CSF v3)
     */
    name?: StoryName;
    /**
     * Override the display name in the UI (CSF v2)
     */
    storyName?: StoryName;
    /**
     * Function that is executed after the story is rendered.
     */
    play?: PlayFunction<TRenderer, TArgs>;
    /**
     * Named tags for a story, used to filter stories in different contexts.
     */
    tags?: Tag[];
    /** @deprecated */
    story?: Omit<StoryAnnotations<TRenderer, TArgs>, 'story'>;
} & ({} extends TRequiredArgs ? {
    args?: TRequiredArgs;
} : {
    args: TRequiredArgs;
});
type LegacyAnnotatedStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = StoryFn<TRenderer, TArgs> & StoryAnnotations<TRenderer, TArgs>;
type LegacyStoryAnnotationsOrFn<TRenderer extends Renderer = Renderer, TArgs = Args> = LegacyAnnotatedStoryFn<TRenderer, TArgs> | StoryAnnotations<TRenderer, TArgs>;
type AnnotatedStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = ArgsStoryFn<TRenderer, TArgs> & StoryAnnotations<TRenderer, TArgs>;
type StoryAnnotationsOrFn<TRenderer extends Renderer = Renderer, TArgs = Args> = AnnotatedStoryFn<TRenderer, TArgs> | StoryAnnotations<TRenderer, TArgs>;
type ArgsFromMeta<TRenderer extends Renderer, Meta> = Meta extends {
    render?: ArgsStoryFn<TRenderer, infer RArgs>;
    loaders?: (infer Loaders)[] | infer Loaders;
    decorators?: (infer Decorators)[] | infer Decorators;
} ? Simplify<RemoveIndexSignature<RArgs & DecoratorsArgs<TRenderer, Decorators> & LoaderArgs<TRenderer, Loaders>>> : unknown;
type DecoratorsArgs<TRenderer extends Renderer, Decorators> = UnionToIntersection<Decorators extends DecoratorFunction<TRenderer, infer TArgs> ? TArgs : unknown>;
type LoaderArgs<TRenderer extends Renderer, Loaders> = UnionToIntersection<Loaders extends LoaderFunction<TRenderer, infer TArgs> ? TArgs : unknown>;
type StoryDescriptor = string[] | RegExp;
interface IncludeExcludeOptions {
    includeStories?: StoryDescriptor;
    excludeStories?: StoryDescriptor;
}
interface SeparatorOptions {
    rootSeparator: string | RegExp;
    groupSeparator: string | RegExp;
}

/**
 * A URL pathname, beginning with a /.
 *
 * @see https://github.com/remix-run/history/tree/main/docs/api-reference.md#location.pathname
 */
declare type Pathname = string;
/**
 * A URL search string, beginning with a ?.
 *
 * @see https://github.com/remix-run/history/tree/main/docs/api-reference.md#location.search
 */
declare type Search = string;
/**
 * A URL fragment identifier, beginning with a #.
 *
 * @see https://github.com/remix-run/history/tree/main/docs/api-reference.md#location.hash
 */
declare type Hash = string;
/**
 * A unique string associated with a location. May be used to safely store
 * and retrieve data in some other storage API, like `localStorage`.
 *
 * @see https://github.com/remix-run/history/tree/main/docs/api-reference.md#location.key
 */
declare type Key = string;
/**
 * The pathname, search, and hash values of a URL.
 */
interface Path$1 {
    /**
     * A URL pathname, beginning with a /.
     *
     * @see https://github.com/remix-run/history/tree/main/docs/api-reference.md#location.pathname
     */
    pathname: Pathname;
    /**
     * A URL search string, beginning with a ?.
     *
     * @see https://github.com/remix-run/history/tree/main/docs/api-reference.md#location.search
     */
    search: Search;
    /**
     * A URL fragment identifier, beginning with a #.
     *
     * @see https://github.com/remix-run/history/tree/main/docs/api-reference.md#location.hash
     */
    hash: Hash;
}
/**
 * An entry in a history stack. A location contains information about the
 * URL path, as well as possibly some arbitrary state and a key.
 *
 * @see https://github.com/remix-run/history/tree/main/docs/api-reference.md#location
 */
interface Location extends Path$1 {
    /**
     * A value of arbitrary data associated with this location.
     *
     * @see https://github.com/remix-run/history/tree/main/docs/api-reference.md#location.state
     */
    state: unknown;
    /**
     * A unique string associated with this location. May be used to safely store
     * and retrieve data in some other storage API, like `localStorage`.
     *
     * Note: This value is always "default" on the initial location.
     *
     * @see https://github.com/remix-run/history/tree/main/docs/api-reference.md#location.key
     */
    key: Key;
}
/**
 * Describes a location that is the destination of some navigation, either via
 * `history.push` or `history.replace`. May be either a URL or the pieces of a
 * URL path.
 */
declare type To = string | Partial<Path$1>;

interface NavigateOptions$1 {
    replace?: boolean;
    state?: any;
}

interface StoryData {
    viewMode?: string;
    storyId?: string;
    refId?: string;
}

interface Other extends StoryData {
    path: string;
    singleStory?: boolean;
}
type NavigateOptions = NavigateOptions$1 & {
    plain?: boolean;
};
type NavigateFunction = (to: To | number, options?: NavigateOptions) => void;
type RouterData = {
    location: Partial<Location>;
    navigate: NavigateFunction;
} & Other;
type RenderData = Pick<RouterData, 'location'> & Other;

interface ThemeVars extends ThemeVarsBase, ThemeVarsColors {
}
interface ThemeVarsBase {
    base: 'light' | 'dark';
}
interface ThemeVarsColors {
    colorPrimary: string;
    colorSecondary: string;
    appBg: string;
    appContentBg: string;
    appPreviewBg: string;
    appBorderColor: string;
    appBorderRadius: number;
    fontBase: string;
    fontCode: string;
    textColor: string;
    textInverseColor: string;
    textMutedColor: string;
    barTextColor: string;
    barHoverColor: string;
    barSelectedColor: string;
    barBg: string;
    buttonBg: string;
    buttonBorder: string;
    booleanBg: string;
    booleanSelectedBg: string;
    inputBg: string;
    inputBorder: string;
    inputTextColor: string;
    inputBorderRadius: number;
    brandTitle?: string;
    brandUrl?: string;
    brandImage?: string;
    brandTarget?: string;
    gridCellSize?: number;
}

type ChannelHandler = (event: ChannelEvent) => void;
interface ChannelTransport {
    send(event: ChannelEvent, options?: any): void;
    setHandler(handler: ChannelHandler): void;
}
interface ChannelEvent {
    type: string;
    from: string;
    args: any[];
}
interface Listener {
    (...args: any[]): void;
}
interface ChannelArgsSingle {
    transport?: ChannelTransport;
    async?: boolean;
}
interface ChannelArgsMulti {
    transports: ChannelTransport[];
    async?: boolean;
}

declare class Channel {
    readonly isAsync: boolean;
    private sender;
    private events;
    private data;
    private readonly transports;
    constructor(input: ChannelArgsMulti);
    constructor(input: ChannelArgsSingle);
    get hasTransport(): boolean;
    addListener(eventName: string, listener: Listener): void;
    emit(eventName: string, ...args: any): void;
    last(eventName: string): any;
    eventNames(): string[];
    listenerCount(eventName: string): number;
    listeners(eventName: string): Listener[] | undefined;
    once(eventName: string, listener: Listener): void;
    removeAllListeners(eventName?: string): void;
    removeListener(eventName: string, listener: Listener): void;
    on(eventName: string, listener: Listener): void;
    off(eventName: string, listener: Listener): void;
    private handleEvent;
    private onceListener;
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
type NormalizedStoriesSpecifier = Required<StoriesSpecifier> & {
    importPathMatcher: RegExp;
};
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
interface BaseIndexEntry {
    id: StoryId;
    name: StoryName;
    title: ComponentTitle;
    tags?: Tag[];
    importPath: Path;
}
type StoryIndexEntry = BaseIndexEntry & {
    type: 'story';
};
type DocsIndexEntry = BaseIndexEntry & {
    storiesImports: Path[];
    type: 'docs';
};
type IndexEntry = StoryIndexEntry | DocsIndexEntry;
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
interface V3CompatIndexEntry extends Omit<StoryIndexEntry, 'type' | 'tags'> {
    kind: ComponentTitle;
    story: StoryName;
    parameters: Parameters;
}
interface StoryIndexV2 {
    v: number;
    stories: Record<StoryId, Omit<V3CompatIndexEntry, 'title' | 'name' | 'importPath'> & {
        name?: StoryName;
    }>;
}
interface StoryIndexV3 {
    v: number;
    stories: Record<StoryId, V3CompatIndexEntry>;
}
interface StoryIndex {
    v: number;
    entries: Record<StoryId, IndexEntry>;
}

/**
 * ⚠️ This file contains internal WIP types they MUST NOT be exported outside this package for now!
 */
type BuilderName = 'webpack5' | '@storybook/builder-webpack5' | string;
type RendererName = string;
interface ServerChannel {
    emit(type: string, args?: any): void;
}
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
    apply(extension: 'typescript', config: Partial<TypescriptOptions>, args?: Options): Promise<Partial<TypescriptOptions>>;
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
type PresetConfig = string | {
    name: string;
    options?: unknown;
};
interface Ref {
    id: string;
    url: string;
    title: string;
    version: string;
    type?: string;
    disable?: boolean;
}
interface VersionCheck {
    success: boolean;
    cached: boolean;
    data?: any;
    error?: any;
    time: number;
}
interface Stats {
    toJson: () => any;
}
interface BuilderResult {
    totalTime?: ReturnType<typeof process.hrtime>;
    stats?: Stats;
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
type Options = LoadOptions & StorybookConfigOptions & CLIOptions & BuilderOptions & {
    build?: TestBuildConfig;
};
interface Builder<Config, BuilderStats extends Stats = Stats> {
    getConfig: (options: Options) => Promise<Config>;
    start: (args: {
        options: Options;
        startTime: ReturnType<typeof process.hrtime>;
        router: Router;
        server: Server;
        channel: ServerChannel;
    }) => Promise<void | {
        stats?: BuilderStats;
        totalTime: ReturnType<typeof process.hrtime>;
        bail: (e?: Error) => Promise<void>;
    }>;
    build: (arg: {
        options: Options;
        startTime: ReturnType<typeof process.hrtime>;
    }) => Promise<void | BuilderStats>;
    bail: (e?: Error) => Promise<void>;
    corePresets?: string[];
    overridePresets?: string[];
}
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
    babel?: (config: TransformOptions, options: Options) => TransformOptions | Promise<TransformOptions>;
    /**
     * Modify or return swc config.
     */
    swc?: (config: Options$2, options: Options) => Options$2 | Promise<Options$2>;
    /**
     * Modify or return env config.
     */
    env?: PresetValue<Record<string, string>>;
    /**
     * Modify or return babel config.
     */
    babelDefault?: (config: TransformOptions, options: Options) => TransformOptions | Promise<TransformOptions>;
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
type PresetValue<T> = T | ((config: T, options: Options) => T | Promise<T>);
type PresetProperty<K, TStorybookConfig = StorybookConfig> = TStorybookConfig[K extends keyof TStorybookConfig ? K : never] | PresetPropertyFn<K, TStorybookConfig>;
type PresetPropertyFn<K, TStorybookConfig = StorybookConfig, TOptions = {}> = (config: TStorybookConfig[K extends keyof TStorybookConfig ? K : never], options: Options & TOptions) => TStorybookConfig[K extends keyof TStorybookConfig ? K : never] | Promise<TStorybookConfig[K extends keyof TStorybookConfig ? K : never]>;
interface CoreCommon_ResolvedAddonPreset {
    type: 'presets';
    name: string;
}
type PreviewAnnotation = string | {
    bare: string;
    absolute: string;
};
interface CoreCommon_ResolvedAddonVirtual {
    type: 'virtual';
    name: string;
    managerEntries?: string[];
    previewAnnotations?: PreviewAnnotation[];
    presets?: (string | {
        name: string;
        options?: any;
    })[];
}
type CoreCommon_OptionsEntry = {
    name: string;
};
type CoreCommon_AddonEntry = string | CoreCommon_OptionsEntry;
type CoreCommon_AddonInfo = {
    name: string;
    inEssentials: boolean;
};
interface CoreCommon_StorybookInfo {
    version: string;
    framework: string;
    frameworkPackage: string;
    renderer: string;
    rendererPackage: string;
    configDir?: string;
    mainConfig?: string;
    previewConfig?: string;
    managerConfig?: string;
}

interface API_BaseEntry {
    id: StoryId;
    depth: number;
    name: string;
    refId?: string;
    renderLabel?: (item: API_BaseEntry) => any;
    /** @deprecated */
    isRoot: boolean;
    /** @deprecated */
    isComponent: boolean;
    /** @deprecated */
    isLeaf: boolean;
}
interface API_RootEntry extends API_BaseEntry {
    type: 'root';
    startCollapsed?: boolean;
    children: StoryId[];
    /** @deprecated */
    isRoot: true;
    /** @deprecated */
    isComponent: false;
    /** @deprecated */
    isLeaf: false;
}
interface API_GroupEntry extends API_BaseEntry {
    type: 'group';
    parent?: StoryId;
    children: StoryId[];
    /** @deprecated */
    isRoot: false;
    /** @deprecated */
    isComponent: false;
    /** @deprecated */
    isLeaf: false;
}
interface API_ComponentEntry extends API_BaseEntry {
    type: 'component';
    parent?: StoryId;
    children: StoryId[];
    /** @deprecated */
    isRoot: false;
    /** @deprecated */
    isComponent: true;
    /** @deprecated */
    isLeaf: false;
}
interface API_DocsEntry extends API_BaseEntry {
    type: 'docs';
    parent: StoryId;
    title: ComponentTitle;
    /** @deprecated */
    kind: ComponentTitle;
    importPath: Path;
    tags: Tag[];
    prepared: boolean;
    parameters?: {
        [parameterName: string]: any;
    };
    /** @deprecated */
    isRoot: false;
    /** @deprecated */
    isComponent: false;
    /** @deprecated */
    isLeaf: true;
}
interface API_StoryEntry extends API_BaseEntry {
    type: 'story';
    parent: StoryId;
    title: ComponentTitle;
    /** @deprecated */
    kind: ComponentTitle;
    importPath: Path;
    tags: Tag[];
    prepared: boolean;
    parameters?: {
        [parameterName: string]: any;
    };
    args?: Args;
    argTypes?: ArgTypes;
    initialArgs?: Args;
    /** @deprecated */
    isRoot: false;
    /** @deprecated */
    isComponent: false;
    /** @deprecated */
    isLeaf: true;
}
type API_LeafEntry = API_DocsEntry | API_StoryEntry;
type API_HashEntry = API_RootEntry | API_GroupEntry | API_ComponentEntry | API_DocsEntry | API_StoryEntry;
/** @deprecated */
type API_Root = API_RootEntry;
/** @deprecated */
type API_Group = API_GroupEntry | API_ComponentEntry;
/** @deprecated */
type API_Story = API_LeafEntry;
/**
 * The `IndexHash` is our manager-side representation of the `StoryIndex`.
 * We create entries in the hash not only for each story or docs entry, but
 * also for each "group" of the component (split on '/'), as that's how things
 * are manipulated in the manager (i.e. in the sidebar)
 */
interface API_IndexHash {
    [id: string]: API_HashEntry;
}
type API_PreparedIndexEntry = IndexEntry & {
    parameters?: Parameters;
    argTypes?: ArgTypes;
    args?: Args;
    initialArgs?: Args;
};
interface API_PreparedStoryIndex {
    v: number;
    entries: Record<StoryId, API_PreparedIndexEntry>;
}
type API_OptionsData = {
    docsOptions: DocsOptions;
};
interface API_ReleaseNotes {
    success?: boolean;
    currentVersion?: string;
    showOnFirstLaunch?: boolean;
}
interface API_Settings {
    lastTrackedStoryId: string;
}
interface API_Version {
    version: string;
    info?: {
        plain: string;
    };
    [key: string]: any;
}
interface API_UnknownEntries {
    [key: string]: {
        [key: string]: any;
    };
}
interface API_Versions$1 {
    latest?: API_Version;
    next?: API_Version;
    current?: API_Version;
}
type API_StatusValue = 'pending' | 'success' | 'error' | 'warn' | 'unknown';
interface API_StatusObject {
    status: API_StatusValue;
    title: string;
    description: string;
    data?: any;
}
type API_StatusState = Record<StoryId, Record<string, API_StatusObject>>;
type API_StatusUpdate = Record<StoryId, API_StatusObject | null>;
type API_FilterFunction = (item: API_PreparedIndexEntry & {
    status: Record<string, API_StatusObject | null>;
}) => boolean;

interface SetStoriesStory {
    id: StoryId;
    name: string;
    refId?: string;
    componentId?: ComponentId;
    kind: StoryKind;
    parameters: {
        fileName: string;
        options: {
            [optionName: string]: any;
        };
        docsOnly?: boolean;
        viewMode?: API_ViewMode;
        [parameterName: string]: any;
    };
    argTypes?: ArgTypes;
    args?: Args;
    initialArgs?: Args;
}
interface SetStoriesStoryData {
    [id: string]: SetStoriesStory;
}
type SetStoriesPayload = {
    v: 2;
    error?: Error;
    globals: Args;
    globalParameters: Parameters;
    stories: SetStoriesStoryData;
    kindParameters: {
        [kind: string]: Parameters;
    };
} | ({
    v?: number;
    stories: SetStoriesStoryData;
} & Record<string, never>);
interface SetGlobalsPayload {
    globals: Globals;
    globalTypes: GlobalTypes;
}
interface StoryPreparedPayload {
    id: StoryId;
    parameters: Parameters;
    argTypes: ArgTypes;
    initialArgs: Args;
    args: Args;
}
interface DocsPreparedPayload {
    id: StoryId;
    parameters: Parameters;
}

type OrString$1<T extends string> = T | (string & {});
type API_ViewMode = OrString$1<'story' | 'docs' | 'settings'> | undefined;
type API_RenderOptions = Addon_RenderOptions;
interface API_RouteOptions {
    storyId: string;
    viewMode: API_ViewMode;
    location: RenderData['location'];
    path: string;
}
interface API_MatchOptions {
    storyId: string;
    viewMode: API_ViewMode;
    location: RenderData['location'];
    path: string;
}
/**
 * @deprecated this is synonymous with `Addon_Type`. This interface will be removed in 8.0
 */
type API_Addon = Addon_Type;
/**
 * @deprecated this is synonymous with `Addon_Collection`. This interface will be removed in 8.0
 */
type API_Collection<T = Addon_Type> = Addon_Collection<T>;
/**
 * @deprecated This interface will be removed in 8.0
 */
type API_Panels = Addon_Collection<Addon_BaseType>;
type API_StateMerger<S> = (input: S) => S;
interface API_ProviderData<API> {
    provider: API_Provider<API>;
    docsOptions: DocsOptions;
}
interface API_Provider<API> {
    channel?: Channel;
    /**
     * @deprecated will be removed in 8.0, please use channel instead
     */
    serverChannel?: Channel;
    renderPreview?: API_IframeRenderer;
    handleAPI(api: API): void;
    getConfig(): {
        sidebar?: API_SidebarOptions;
        theme?: ThemeVars;
        StoryMapper?: API_StoryMapper;
        [k: string]: any;
    } & Partial<API_UIOptions>;
    [key: string]: any;
}
type API_IframeRenderer = (storyId: string, viewMode: API_ViewMode, id: string, baseUrl: string, scale: number, queryParams: Record<string, any>) => ReactElement<any, any> | null;
interface API_UIOptions {
    name?: string;
    url?: string;
    goFullScreen: boolean;
    showStoriesPanel: boolean;
    showAddonPanel: boolean;
    addonPanelInRight: boolean;
    theme?: ThemeVars;
    selectedPanel?: string;
}
interface API_Layout {
    initialActive: API_ActiveTabsType;
    isFullscreen: boolean;
    showPanel: boolean;
    panelPosition: API_PanelPositions;
    showNav: boolean;
    showTabs: boolean;
    showToolbar: boolean;
    /**
     * @deprecated, will be removed in 8.0 - this API no longer works
     */
    isToolshown?: boolean;
}
interface API_UI {
    name?: string;
    url?: string;
    enableShortcuts: boolean;
}
type API_PanelPositions = 'bottom' | 'right';
type API_ActiveTabsType = 'sidebar' | 'canvas' | 'addons';
interface API_SidebarOptions {
    showRoots?: boolean;
    filters?: Record<string, API_FilterFunction>;
    collapsedRoots?: string[];
    renderLabel?: (item: API_HashEntry) => any;
}
interface OnClearOptions {
    /**
     *  True when the user dismissed the notification.
     */
    dismissed: boolean;
}
interface API_Notification {
    id: string;
    link: string;
    content: {
        headline: string;
        subHeadline?: string | any;
    };
    icon?: {
        name: string;
        color?: string;
    };
    onClear?: (options: OnClearOptions) => void;
}
type API_Versions = Record<string, string>;
type API_SetRefData = Partial<API_ComposedRef & {
    setStoriesData: SetStoriesStoryData;
    storyIndex: StoryIndex;
}>;
type API_StoryMapper = (ref: API_ComposedRef, story: SetStoriesStory) => SetStoriesStory;
interface API_LoadedRefData {
    index?: API_IndexHash;
    indexError?: Error;
    previewInitialized: boolean;
}
interface API_ComposedRef extends API_LoadedRefData {
    id: string;
    title?: string;
    url: string;
    type?: 'auto-inject' | 'unknown' | 'lazy' | 'server-checked';
    expanded?: boolean;
    versions?: API_Versions;
    loginUrl?: string;
    version?: string;
}
type API_ComposedRefUpdate = Partial<Pick<API_ComposedRef, 'title' | 'type' | 'expanded' | 'index' | 'versions' | 'loginUrl' | 'version' | 'indexError' | 'previewInitialized'>>;
type API_Refs = Record<string, API_ComposedRef>;
type API_RefId = string;
type API_RefUrl = string;

type Addon_Types = Exclude<Addon_TypesEnum, Addon_TypesEnum.experimental_PAGE | Addon_TypesEnum.experimental_SIDEBAR_BOTTOM | Addon_TypesEnum.experimental_SIDEBAR_TOP>;
interface Addon_ArgType<TArg = unknown> extends InputType {
    defaultValue?: TArg;
}
type Addons_ArgTypes<TArgs = Args> = {
    [key in keyof Partial<TArgs>]: Addon_ArgType<TArgs[key]>;
} & {
    [key in string]: Addon_ArgType<unknown>;
};
type Addon_Comparator<T> = ((a: T, b: T) => boolean) | ((a: T, b: T) => number);
type Addon_StorySortMethod = 'configure' | 'alphabetical';
interface Addon_StorySortObjectParameter {
    method?: Addon_StorySortMethod;
    order?: any[];
    locales?: string;
    includeNames?: boolean;
}
type IndexEntryLegacy = [StoryId, any, Parameters, Parameters];
type Addon_StorySortComparator = Addon_Comparator<IndexEntryLegacy>;
type Addon_StorySortParameter = Addon_StorySortComparator | Addon_StorySortObjectParameter;
type Addon_StorySortComparatorV7 = Addon_Comparator<IndexEntry>;
type Addon_StorySortParameterV7 = Addon_StorySortComparatorV7 | Addon_StorySortObjectParameter;
interface Addon_OptionsParameter extends Object {
    storySort?: Addon_StorySortParameter;
    theme?: {
        base: string;
        brandTitle?: string;
    };
    [key: string]: any;
}
interface Addon_OptionsParameterV7 extends Object {
    storySort?: Addon_StorySortParameterV7;
    theme?: {
        base: string;
        brandTitle?: string;
    };
    [key: string]: any;
}
type Addon_StoryContext<TRenderer extends Renderer = Renderer> = StoryContext<TRenderer>;
type Addon_StoryContextUpdate = Partial<Addon_StoryContext>;
type Addon_ReturnTypeFramework<ReturnType> = {
    component: any;
    storyResult: ReturnType;
    canvasElement: any;
};
type Addon_PartialStoryFn<ReturnType = unknown> = PartialStoryFn<Addon_ReturnTypeFramework<ReturnType>>;
type Addon_LegacyStoryFn<ReturnType = unknown> = LegacyStoryFn<Addon_ReturnTypeFramework<ReturnType>>;
type Addon_ArgsStoryFn<ReturnType = unknown> = ArgsStoryFn<Addon_ReturnTypeFramework<ReturnType>>;
type Addon_StoryFn<ReturnType = unknown> = StoryFn<Addon_ReturnTypeFramework<ReturnType>>;
type Addon_DecoratorFunction<StoryFnReturnType = unknown> = DecoratorFunction<Addon_ReturnTypeFramework<StoryFnReturnType>>;
type Addon_LoaderFunction = LoaderFunction<Addon_ReturnTypeFramework<unknown>>;
interface Addon_WrapperSettings {
    options: object;
    parameters: {
        [key: string]: any;
    };
}
type Addon_StoryWrapper = (storyFn: Addon_LegacyStoryFn, context: Addon_StoryContext, settings: Addon_WrapperSettings) => any;
type Addon_MakeDecoratorResult = (...args: any) => any;
interface Addon_AddStoryArgs<StoryFnReturnType = unknown> {
    id: StoryId;
    kind: StoryKind;
    name: StoryName;
    storyFn: Addon_StoryFn<StoryFnReturnType>;
    parameters: Parameters;
}
type Addon_ClientApiAddon<StoryFnReturnType = unknown> = Addon_Type & {
    apply: (a: Addon_StoryApi<StoryFnReturnType>, b: any[]) => any;
};
interface Addon_ClientApiAddons<StoryFnReturnType> {
    [key: string]: Addon_ClientApiAddon<StoryFnReturnType>;
}
type Addon_ClientApiReturnFn<StoryFnReturnType = unknown> = (...args: any[]) => Addon_StoryApi<StoryFnReturnType>;
interface Addon_StoryApi<StoryFnReturnType = unknown> {
    kind: StoryKind;
    add: (storyName: StoryName, storyFn: Addon_StoryFn<StoryFnReturnType>, parameters?: Parameters) => Addon_StoryApi<StoryFnReturnType>;
    addDecorator: (decorator: Addon_DecoratorFunction<StoryFnReturnType>) => Addon_StoryApi<StoryFnReturnType>;
    addLoader: (decorator: Addon_LoaderFunction) => Addon_StoryApi<StoryFnReturnType>;
    addParameters: (parameters: Parameters) => Addon_StoryApi<StoryFnReturnType>;
    [k: string]: string | Addon_ClientApiReturnFn<StoryFnReturnType>;
}
interface Addon_ClientStoryApi<StoryFnReturnType = unknown> {
    storiesOf(kind: StoryKind, module: any): Addon_StoryApi<StoryFnReturnType>;
}
type Addon_LoadFn = () => any;
type Addon_RequireContext = any;
type Addon_Loadable = Addon_RequireContext | [Addon_RequireContext] | Addon_LoadFn;
type Addon_BaseDecorators<StoryFnReturnType> = Array<(story: () => StoryFnReturnType, context: Addon_StoryContext) => StoryFnReturnType>;
interface Addon_BaseAnnotations<TArgs, StoryFnReturnType, TRenderer extends Renderer = Renderer> {
    /**
     * Dynamic data that are provided (and possibly updated by) Storybook and its addons.
     * @see [Arg story inputs](https://storybook.js.org/docs/react/api/csf#args-story-inputs)
     */
    args?: Partial<TArgs>;
    /**
     * ArgTypes encode basic metadata for args, such as `name`, `description`, `defaultValue` for an arg. These get automatically filled in by Storybook Docs.
     * @see [Arg types](https://storybook.js.org/docs/react/api/arg-types)
     */
    argTypes?: Addons_ArgTypes<TArgs>;
    /**
     * Custom metadata for a story.
     * @see [Parameters](https://storybook.js.org/docs/react/writing-stories/parameters)
     */
    parameters?: Parameters;
    /**
     * Wrapper components or Storybook decorators that wrap a story.
     *
     * Decorators defined in Meta will be applied to every story variation.
     * @see [Decorators](https://storybook.js.org/docs/addons/#1-decorators)
     */
    decorators?: Addon_BaseDecorators<StoryFnReturnType>;
    /**
     * Define a custom render function for the story(ies). If not passed, a default render function by the framework will be used.
     */
    render?: (args: TArgs, context: Addon_StoryContext<TRenderer>) => StoryFnReturnType;
    /**
     * Function that is executed after the story is rendered.
     */
    play?: (context: Addon_StoryContext<TRenderer>) => Promise<void> | void;
}
interface Addon_Annotations<TArgs, StoryFnReturnType> extends Addon_BaseAnnotations<TArgs, StoryFnReturnType> {
    /**
     * Used to only include certain named exports as stories. Useful when you want to have non-story exports such as mock data or ignore a few stories.
     * @example
     * includeStories: ['SimpleStory', 'ComplexStory']
     * includeStories: /.*Story$/
     *
     * @see [Non-story exports](https://storybook.js.org/docs/formats/component-story-format/#non-story-exports)
     */
    includeStories?: string[] | RegExp;
    /**
     * Used to exclude certain named exports. Useful when you want to have non-story exports such as mock data or ignore a few stories.
     * @example
     * excludeStories: ['simpleData', 'complexData']
     * excludeStories: /.*Data$/
     *
     * @see [Non-story exports](https://storybook.js.org/docs/formats/component-story-format/#non-story-exports)
     */
    excludeStories?: string[] | RegExp;
}
interface Addon_BaseMeta<ComponentType> {
    /**
     * Title of the story which will be presented in the navigation. **Should be unique.**
     *
     * Stories can be organized in a nested structure using "/" as a separator.
     *
     * Since CSF 3.0 this property is optional.
     *
     * @example
     * export default {
     *   ...
     *   title: 'Design System/Atoms/Button'
     * }
     *
     * @see [Story Hierarchy](https://storybook.js.org/docs/basics/writing-stories/#story-hierarchy)
     */
    title?: string;
    /**
     * Manually set the id of a story, which in particular is useful if you want to rename stories without breaking permalinks.
     *
     * Storybook will prioritize the id over the title for ID generation, if provided, and will prioritize the story.storyName over the export key for display.
     *
     * @see [Sidebar and URLs](https://storybook.js.org/docs/react/configure/sidebar-and-urls#permalinking-to-stories)
     */
    id?: string;
    /**
     * The primary component for your story.
     *
     * Used by addons for automatic prop table generation and display of other component metadata.
     */
    component?: ComponentType;
    /**
     * Auxiliary subcomponents that are part of the stories.
     *
     * Used by addons for automatic prop table generation and display of other component metadata.
     *
     * @example
     * import { Button, ButtonGroup } from './components';
     *
     * export default {
     *   ...
     *   subcomponents: { Button, ButtonGroup }
     * }
     *
     * By defining them each component will have its tab in the args table.
     *
     * @deprecated
     */
    subcomponents?: Record<string, ComponentType>;
}
type Addon_BaseStoryObject<TArgs, StoryFnReturnType> = {
    /**
     * Override the display name in the UI
     */
    storyName?: string;
};
type Addon_BaseStoryFn<TArgs, StoryFnReturnType> = {
    (args: TArgs, context: Addon_StoryContext): StoryFnReturnType;
} & Addon_BaseStoryObject<TArgs, StoryFnReturnType>;
type BaseStory<TArgs, StoryFnReturnType> = Addon_BaseStoryFn<TArgs, StoryFnReturnType> | Addon_BaseStoryObject<TArgs, StoryFnReturnType>;
interface Addon_RenderOptions {
    active: boolean;
    /**
     * @deprecated You should not use key anymore as of Storybook 7.2 this render method is invoked as a React component.
     * This property will be removed in 8.0.
     * */
    key?: unknown;
}
/**
 * @deprecated This type is deprecated and will be removed in 8.0.
 */
type ReactJSXElement = {
    type: any;
    props: any;
    key: any;
};
type Addon_Type = Addon_BaseType | Addon_PageType | Addon_WrapperType | Addon_SidebarBottomType | Addon_SidebarTopType;
interface Addon_BaseType {
    /**
     * The title of the addon.
     * This can be a simple string, but it can also be a React.FunctionComponent or a React.ReactElement.
     */
    title: FCWithoutChildren | ReactNode;
    /**
     * The type of the addon.
     * @example Addon_TypesEnum.PANEL
     */
    type: Exclude<Addon_Types, Addon_TypesEnum.PREVIEW | Addon_TypesEnum.experimental_PAGE | Addon_TypesEnum.experimental_SIDEBAR_BOTTOM | Addon_TypesEnum.experimental_SIDEBAR_TOP>;
    /**
     * The unique id of the addon.
     * @warn This will become non-optional in 8.0
     *
     * This needs to be globally unique, so we recommend prefixing it with your org name or npm package name.
     *
     * Do not prefix with `storybook`, this is reserved for core storybook feature and core addons.
     *
     * @example 'my-org-name/my-addon-name'
     */
    id?: string;
    /**
     * This component will wrap your `render` function.
     *
     * With it you can determine if you want your addon to be rendered or not.
     *
     * This is to facilitate addons keeping state, and keep listening for events even when they are not currently on screen/rendered.
     */
    route?: (routeOptions: RenderData) => string;
    /**
     * This will determine the value of `active` prop of your render function.
     */
    match?: (matchOptions: RenderData) => boolean;
    /**
     * The actual contents of your addon.
     *
     * This is called as a function, so if you want to use hooks,
     * your function needs to return a JSX.Element within which components are rendered
     */
    render: (renderOptions: Partial<Addon_RenderOptions>) => ReactElement<any, any> | null;
    /**
     * @unstable
     */
    paramKey?: string;
    /**
     * @unstable
     */
    disabled?: boolean;
    /**
     * @unstable
     */
    hidden?: boolean;
}
/**
 * This is a copy of FC from react/index.d.ts, but has the PropsWithChildren type removed
 * this is correct and more type strict, and future compatible with React.FC in React 18+
 *
 * @deprecated This type is deprecated and will be removed in 8.0. (assuming the manager uses React 18 is out by then)
 */
interface FCWithoutChildren<P = {}> {
    (props: P, context?: any): ReactElement<any, any> | null;
    propTypes?: WeakValidationMap<P> | undefined;
    contextTypes?: ValidationMap<any> | undefined;
    defaultProps?: Partial<P> | undefined;
    displayName?: string | undefined;
}
interface Addon_PageType {
    type: Addon_TypesEnum.experimental_PAGE;
    /**
     * The unique id of the page.
     */
    id: string;
    /**
     * The URL to navigate to when Storybook needs to navigate to this page.
     */
    url: string;
    /**
     * The title is used in mobile mode to represent the page in the navigation.
     */
    title: FCWithoutChildren | string | ReactElement | ReactNode;
    /**
     * The main content of the addon, a function component without any props.
     * Storybook will render your component always.
     *
     * If you want to render your component only when the URL matches, use the `Route` component.
     * @example
     * import { Route } from '@storybook/router';
     *
     * render: () => {
     *   return (
     *     <Route path="/my-addon">
     *       <MyAddonContent />
     *     </Route>
     *   );
     * };
     */
    render: FCWithoutChildren;
}
interface Addon_WrapperType {
    type: Addon_TypesEnum.PREVIEW;
    /**
     * The unique id of the page.
     */
    id: string;
    /**
     * A React.FunctionComponent that wraps the story.
     *
     * This component must accept a children prop, and render it.
     */
    render: FC<PropsWithChildren<{
        index: number;
        children: ReactNode;
        id: string;
        storyId: StoryId;
        active: boolean;
    }>>;
}
interface Addon_SidebarBottomType {
    type: Addon_TypesEnum.experimental_SIDEBAR_BOTTOM;
    /**
     * The unique id of the tool.
     */
    id: string;
    /**
     * A React.FunctionComponent.
     */
    render: FCWithoutChildren;
}
interface Addon_SidebarTopType {
    type: Addon_TypesEnum.experimental_SIDEBAR_TOP;
    /**
     * The unique id of the tool.
     */
    id: string;
    /**
     * A React.FunctionComponent.
     */
    render: FCWithoutChildren;
}
type Addon_TypeBaseNames = Exclude<Addon_TypesEnum, Addon_TypesEnum.PREVIEW | Addon_TypesEnum.experimental_PAGE | Addon_TypesEnum.experimental_SIDEBAR_BOTTOM | Addon_TypesEnum.experimental_SIDEBAR_TOP>;
interface Addon_TypesMapping extends Record<Addon_TypeBaseNames, Addon_BaseType> {
    [Addon_TypesEnum.PREVIEW]: Addon_WrapperType;
    [Addon_TypesEnum.experimental_PAGE]: Addon_PageType;
    [Addon_TypesEnum.experimental_SIDEBAR_BOTTOM]: Addon_SidebarBottomType;
    [Addon_TypesEnum.experimental_SIDEBAR_TOP]: Addon_SidebarTopType;
}
type Addon_Loader<API> = (api: API) => void;
interface Addon_Loaders<API> {
    [key: string]: Addon_Loader<API>;
}
interface Addon_Collection<T = Addon_Type> {
    [key: string]: T;
}
interface Addon_Elements {
    [key: string]: Addon_Collection;
}
interface Addon_ToolbarConfig {
    hidden?: boolean;
}
interface Addon_Config {
    theme?: ThemeVars;
    toolbar?: {
        [id: string]: Addon_ToolbarConfig;
    };
    sidebar?: API_SidebarOptions;
    [key: string]: any;
}
declare enum Addon_TypesEnum {
    /**
     * This API is used to create a tab the toolbar above the canvas, This API might be removed in the future.
     * @unstable
     */
    TAB = "tab",
    /**
     * This adds panels to the addons side panel.
     */
    PANEL = "panel",
    /**
     * This adds items in the toolbar above the canvas - on the left side.
     */
    TOOL = "tool",
    /**
     * This adds items in the toolbar above the canvas - on the right side.
     */
    TOOLEXTRA = "toolextra",
    /**
     * This adds wrapper components around the canvas/iframe component storybook renders.
     * @unstable this API is not stable yet, and is likely to change in 8.0.
     */
    PREVIEW = "preview",
    /**
     * This adds pages that render instead of the canvas.
     * @unstable
     */
    experimental_PAGE = "page",
    /**
     * This adds items in the bottom of the sidebar.
     * @unstable
     */
    experimental_SIDEBAR_BOTTOM = "sidebar-bottom",
    /**
     * This adds items in the top of the sidebar.
     * @unstable This will get replaced with a new API in 8.0, use at your own risk.
     */
    experimental_SIDEBAR_TOP = "sidebar-top",
    /**
     * @deprecated This property does nothing, and will be removed in Storybook 8.0.
     */
    NOTES_ELEMENT = "notes-element"
}

type OrString<T extends string> = T | (string & {});
type ViewMode = OrString<ViewMode$1 | 'settings'> | undefined;
type Layout = 'centered' | 'fullscreen' | 'padded' | 'none';
interface StorybookParameters {
    options?: Addon_OptionsParameter;
    /**
     * The layout property defines basic styles added to the preview body where the story is rendered.
     *
     * If you pass `none`, no styles are applied.
     */
    layout?: Layout;
}
interface StorybookInternalParameters extends StorybookParameters {
    fileName?: string;
    docsOnly?: true;
}
type Path = string;

interface WebRenderer extends Renderer {
    canvasElement: HTMLElement;
}
type ModuleExport = any;
type ModuleExports = Record<string, ModuleExport>;
type ModuleImportFn = (path: Path) => Promise<ModuleExports>;
type MaybePromise<T> = Promise<T> | T;
type TeardownRenderToCanvas = () => MaybePromise<void>;
type RenderToCanvas<TRenderer extends Renderer> = (context: RenderContext<TRenderer>, element: TRenderer['canvasElement']) => MaybePromise<void | TeardownRenderToCanvas>;
type ProjectAnnotations<TRenderer extends Renderer> = ProjectAnnotations$1<TRenderer> & {
    renderToCanvas?: RenderToCanvas<TRenderer>;
    renderToDOM?: RenderToCanvas<TRenderer>;
};
type NormalizedProjectAnnotations<TRenderer extends Renderer = Renderer> = Omit<ProjectAnnotations<TRenderer>, 'decorators' | 'loaders'> & {
    argTypes?: StrictArgTypes;
    globalTypes?: StrictGlobalTypes;
    decorators?: DecoratorFunction<TRenderer>[];
    loaders?: LoaderFunction<TRenderer>[];
};
type NormalizedComponentAnnotations<TRenderer extends Renderer = Renderer> = Omit<ComponentAnnotations<TRenderer>, 'decorators' | 'loaders'> & {
    id: ComponentId;
    title: ComponentTitle;
    argTypes?: StrictArgTypes;
    decorators?: DecoratorFunction<TRenderer>[];
    loaders?: LoaderFunction<TRenderer>[];
};
type NormalizedStoryAnnotations<TRenderer extends Renderer = Renderer> = Omit<StoryAnnotations<TRenderer>, 'storyName' | 'story' | 'decorators' | 'loaders'> & {
    moduleExport: ModuleExport;
    id: StoryId;
    argTypes?: StrictArgTypes;
    name: StoryName;
    userStoryFn?: StoryFn<TRenderer>;
    decorators?: DecoratorFunction<TRenderer>[];
    loaders?: LoaderFunction<TRenderer>[];
};
type CSFFile<TRenderer extends Renderer = Renderer> = {
    meta: NormalizedComponentAnnotations<TRenderer>;
    stories: Record<StoryId, NormalizedStoryAnnotations<TRenderer>>;
    moduleExports: ModuleExports;
};
type PreparedStory<TRenderer extends Renderer = Renderer> = StoryContextForEnhancers<TRenderer> & {
    moduleExport: ModuleExport;
    originalStoryFn: StoryFn<TRenderer>;
    undecoratedStoryFn: LegacyStoryFn<TRenderer>;
    unboundStoryFn: LegacyStoryFn<TRenderer>;
    applyLoaders: (context: StoryContextForLoaders<TRenderer>) => Promise<StoryContextForLoaders<TRenderer> & {
        loaded: StoryContext<TRenderer>['loaded'];
    }>;
    playFunction?: (context: StoryContext<TRenderer>) => Promise<void> | void;
};
type PreparedMeta<TRenderer extends Renderer = Renderer> = Omit<StoryContextForEnhancers<TRenderer>, 'name' | 'story'> & {
    moduleExport: ModuleExport;
};
type BoundStory<TRenderer extends Renderer = Renderer> = PreparedStory<TRenderer> & {
    storyFn: PartialStoryFn<TRenderer>;
};
declare type RenderContext<TRenderer extends Renderer = Renderer> = StoryIdentifier & {
    showMain: () => void;
    showError: (error: {
        title: string;
        description: string;
    }) => void;
    showException: (err: Error) => void;
    forceRemount: boolean;
    storyContext: StoryContext<TRenderer>;
    storyFn: PartialStoryFn<TRenderer>;
    unboundStoryFn: LegacyStoryFn<TRenderer>;
};

interface BuilderStats {
    toJson: () => any;
}
type Builder_WithRequiredProperty<Type, Key extends keyof Type> = Type & {
    [Property in Key]-?: Type[Property];
};
type Builder_Unpromise<T extends Promise<any>> = T extends Promise<infer U> ? U : never;
type Builder_EnvsRaw = Record<string, string>;

type RenderContextCallbacks<TRenderer extends Renderer> = Pick<RenderContext<TRenderer>, 'showMain' | 'showError' | 'showException'>;
type StoryRenderOptions = {
    autoplay?: boolean;
    forceInitialArgs?: boolean;
};
type ResolvedModuleExportType = 'component' | 'meta' | 'story';
/**
 * What do we know about an of={} call?
 *
 * Technically, the type names aren't super accurate:
 *   - meta === `CSFFile`
 *   - story === `PreparedStory`
 * But these shorthands capture the idea of what is being talked about
 */
type ResolvedModuleExportFromType<TType extends ResolvedModuleExportType, TRenderer extends Renderer = Renderer> = TType extends 'component' ? {
    type: 'component';
    component: TRenderer['component'];
    projectAnnotations: NormalizedProjectAnnotations<Renderer>;
} : TType extends 'meta' ? {
    type: 'meta';
    csfFile: CSFFile<TRenderer>;
    preparedMeta: PreparedMeta;
} : {
    type: 'story';
    story: PreparedStory<TRenderer>;
};
type ResolvedModuleExport<TRenderer extends Renderer = Renderer> = {
    type: ResolvedModuleExportType;
} & (ResolvedModuleExportFromType<'component', TRenderer> | ResolvedModuleExportFromType<'meta', TRenderer> | ResolvedModuleExportFromType<'story', TRenderer>);
interface DocsContextProps<TRenderer extends Renderer = Renderer> {
    /**
     * Register a CSF file that this docs entry uses.
     * Used by the `<Meta of={} />` block to attach, and the `<Story meta={} />` bloc to reference
     */
    referenceMeta: (metaExports: ModuleExports, attach: boolean) => void;
    /**
     * Find a component, meta or story object from the direct export(s) from the CSF file.
     * This is the API that drives the `of={}` syntax.
     */
    resolveOf<TType extends ResolvedModuleExportType>(moduleExportOrType: ModuleExport | TType, validTypes?: TType[]): ResolvedModuleExportFromType<TType, TRenderer>;
    /**
     * Find a story's id from the name of the story.
     * This is primarily used by the `<Story name={} /> block.
     * Note that the story must be part of the primary CSF file of the docs entry.
     */
    storyIdByName: (storyName: StoryName) => StoryId;
    /**
     * Syncronously find a story by id (if the id is not provided, this will look up the primary
     * story in the CSF file, if such a file exists).
     */
    storyById: (id?: StoryId) => PreparedStory<TRenderer>;
    /**
     * Syncronously find all stories of the component referenced by the CSF file.
     */
    componentStories: () => PreparedStory<TRenderer>[];
    /**
     * Get the story context of the referenced story.
     */
    getStoryContext: (story: PreparedStory<TRenderer>) => StoryContextForLoaders<TRenderer>;
    /**
     * Asyncronously load an arbitrary story by id.
     */
    loadStory: (id: StoryId) => Promise<PreparedStory<TRenderer>>;
    /**
     * Render a story to a given HTML element and keep it up to date across context changes
     */
    renderStoryToElement: (story: PreparedStory<TRenderer>, element: HTMLElement, callbacks: RenderContextCallbacks<TRenderer>, options: StoryRenderOptions) => () => Promise<void>;
    /**
     * Storybook channel -- use for low level event watching/emitting
     */
    channel: Channel$1;
    /**
     * Project annotations -- can be read to get the project's global annotations
     */
    projectAnnotations: NormalizedProjectAnnotations<TRenderer>;
}
type DocsRenderFunction<TRenderer extends Renderer> = (docsContext: DocsContextProps<TRenderer>, docsParameters: Parameters, element: HTMLElement) => Promise<void>;

type Store_CSFExports<TRenderer extends Renderer = Renderer, TArgs extends Args = Args> = {
    default: ComponentAnnotations<TRenderer, TArgs>;
    __esModule?: boolean;
    __namedExportsOrder?: string[];
};
/**
 * Type for the play function returned by a composed story, which will contain everything needed in the context,
 * except the canvasElement, which should be passed by the user.
 * It's useful for scenarios where the user wants to execute the play function in test environments, e.g.
 *
 * const { PrimaryButton } = composeStories(stories)
 * const { container } = render(<PrimaryButton />) // or PrimaryButton()
 * PrimaryButton.play({ canvasElement: container })
 */
type ComposedStoryPlayContext<TRenderer extends Renderer = Renderer, TArgs = Args> = Partial<StoryContext<TRenderer, TArgs> & Pick<StoryContext<TRenderer, TArgs>, 'canvasElement'>>;
type ComposedStoryPlayFn<TRenderer extends Renderer = Renderer, TArgs = Args> = (context: ComposedStoryPlayContext<TRenderer, TArgs>) => Promise<void> | void;
/**
 * A story function with partial args, used internally by composeStory
 */
type PartialArgsStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = (args?: TArgs) => (TRenderer & {
    T: TArgs;
})['storyResult'];
/**
 * A story that got recomposed for portable stories, containing all the necessary data to be rendered in external environments
 */
type ComposedStoryFn<TRenderer extends Renderer = Renderer, TArgs = Args> = PartialArgsStoryFn<TRenderer, TArgs> & {
    play: ComposedStoryPlayFn<TRenderer, TArgs>;
    args: TArgs;
    id: StoryId;
    storyName: string;
    parameters: Parameters;
};
/**
 * Based on a module of stories, it returns all stories within it, filtering non-stories
 * Each story will have partial props, as their props should be handled when composing stories
 */
type StoriesWithPartialProps<TRenderer extends Renderer, TModule> = {
    [K in keyof TModule as TModule[K] extends StoryAnnotationsOrFn<infer _, infer _TProps> ? K : never]: TModule[K] extends StoryAnnotationsOrFn<infer _, infer TProps> ? ComposedStoryFn<TRenderer, Partial<TProps>> : unknown;
};
/**
 * Type used for integrators of portable stories, as reference when creating their own composeStory function
 */
interface ComposeStoryFn<TRenderer extends Renderer = Renderer, TArgs extends Args = Args> {
    (storyAnnotations: AnnotatedStoryFn<TRenderer, TArgs> | StoryAnnotations<TRenderer, TArgs>, componentAnnotations: ComponentAnnotations<TRenderer, TArgs>, projectAnnotations: ProjectAnnotations<TRenderer>, exportsName?: string): ComposedStoryFn;
}

export { API_ActiveTabsType, API_Addon, API_BaseEntry, API_Collection, API_ComponentEntry, API_ComposedRef, API_ComposedRefUpdate, API_DocsEntry, API_FilterFunction, API_Group, API_GroupEntry, API_HashEntry, API_IframeRenderer, API_IndexHash, API_Layout, API_LeafEntry, API_LoadedRefData, API_MatchOptions, API_Notification, API_OptionsData, API_PanelPositions, API_Panels, API_PreparedIndexEntry, API_PreparedStoryIndex, API_Provider, API_ProviderData, API_RefId, API_RefUrl, API_Refs, API_ReleaseNotes, API_RenderOptions, API_Root, API_RootEntry, API_RouteOptions, API_SetRefData, API_Settings, API_SidebarOptions, API_StateMerger, API_StatusObject, API_StatusState, API_StatusUpdate, API_StatusValue, API_Story, API_StoryEntry, API_StoryMapper, API_UI, API_UIOptions, API_UnknownEntries, API_Version, API_Versions$1 as API_Versions, API_ViewMode, Addon_AddStoryArgs, Addon_Annotations, Addon_ArgType, Addon_ArgsStoryFn, Addon_BaseAnnotations, Addon_BaseDecorators, Addon_BaseMeta, Addon_BaseStoryFn, Addon_BaseStoryObject, Addon_BaseType, Addon_ClientApiAddon, Addon_ClientApiAddons, Addon_ClientApiReturnFn, Addon_ClientStoryApi, Addon_Collection, Addon_Comparator, Addon_Config, Addon_DecoratorFunction, Addon_Elements, Addon_LegacyStoryFn, Addon_LoadFn, Addon_Loadable, Addon_Loader, Addon_LoaderFunction, Addon_Loaders, Addon_MakeDecoratorResult, Addon_OptionsParameter, Addon_OptionsParameterV7, Addon_PageType, Addon_PartialStoryFn, Addon_RenderOptions, Addon_RequireContext, Addon_SidebarBottomType, Addon_SidebarTopType, Addon_StoryApi, Addon_StoryContext, Addon_StoryContextUpdate, Addon_StoryFn, Addon_StorySortComparator, Addon_StorySortComparatorV7, Addon_StorySortMethod, Addon_StorySortObjectParameter, Addon_StorySortParameter, Addon_StorySortParameterV7, Addon_StoryWrapper, Addon_ToolbarConfig, Addon_Type, Addon_Types, Addon_TypesEnum, Addon_TypesMapping, Addon_WrapperSettings, Addon_WrapperType, Addons_ArgTypes, AnnotatedStoryFn, ArgTypes, ArgTypesEnhancer, Args, ArgsEnhancer, ArgsFromMeta, ArgsStoryFn, BaseAnnotations, BaseIndexEntry, BaseIndexInput, BaseStory, BoundStory, Builder, BuilderName, BuilderOptions, BuilderResult, BuilderStats, Builder_EnvsRaw, Builder_Unpromise, Builder_WithRequiredProperty, CLIOptions, CSFFile, ComponentAnnotations, ComponentId, ComponentTitle, ComposeStoryFn, ComposedStoryFn, ComposedStoryPlayContext, ComposedStoryPlayFn, Conditional, CoreCommon_AddonEntry, CoreCommon_AddonInfo, CoreCommon_OptionsEntry, CoreCommon_ResolvedAddonPreset, CoreCommon_ResolvedAddonVirtual, CoreCommon_StorybookInfo, CoreConfig, DecoratorApplicator, DecoratorFunction, DeprecatedIndexer, DocsContextProps, DocsIndexEntry, DocsIndexInput, DocsOptions, DocsPreparedPayload, DocsRenderFunction, Entry, GlobalTypes, Globals, IncludeExcludeOptions, IndexEntry, IndexEntryLegacy, IndexInput, IndexedCSFFile, IndexedStory, Indexer, IndexerOptions, InputType, LegacyAnnotatedStoryFn, LegacyStoryAnnotationsOrFn, LegacyStoryFn, LoadOptions, LoadedPreset, LoaderFunction, ModuleExport, ModuleExports, ModuleImportFn, NormalizedComponentAnnotations, NormalizedProjectAnnotations, NormalizedStoriesSpecifier, NormalizedStoryAnnotations, Options, PackageJson, Parameters, PartialArgsStoryFn, PartialStoryFn, Path, PlayFunction, PlayFunctionContext, PreparedMeta, PreparedStory, Preset, PresetConfig, PresetProperty, PresetPropertyFn, PresetValue, Presets, PreviewAnnotation, ProjectAnnotations, ReactJSXElement, Ref, RenderContext, RenderContextCallbacks, RenderToCanvas, Renderer, RendererName, ResolvedModuleExport, ResolvedModuleExportFromType, ResolvedModuleExportType, SBArrayType, SBEnumType, SBIntersectionType, SBObjectType, SBOtherType, SBScalarType, SBType, SBUnionType, SeparatorOptions, SetGlobalsPayload, SetStoriesPayload, SetStoriesStory, SetStoriesStoryData, Stats, StepFunction, StepLabel, StepRunner, Store_CSFExports, StoriesEntry, StoriesWithPartialProps, StoryAnnotations, StoryAnnotationsOrFn, StoryContext, StoryContextForEnhancers, StoryContextForLoaders, StoryContextUpdate, StoryFn, StoryId, StoryIdentifier, StoryIndex, StoryIndexEntry, StoryIndexInput, StoryIndexV2, StoryIndexV3, StoryIndexer, StoryKind, StoryName, StoryPreparedPayload, StoryRenderOptions, StorybookConfig, StorybookConfigOptions, StorybookInternalParameters, StorybookParameters, StrictArgTypes, StrictArgs, StrictGlobalTypes, StrictInputType, Tag, TeardownRenderToCanvas, TestBuildConfig, TestBuildFlags, TypescriptOptions, V3CompatIndexEntry, VersionCheck, ViewMode, WebRenderer };
