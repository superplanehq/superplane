# Storybook CLI

Storybook CLI (_Command Line Interface_) is the easiest way to add [Storybook](https://github.com/storybookjs/storybook) to your project.

![Screenshot](docs/getstorybook.png)

Go to your project and run:

```sh
cd my-app
npx sb init
```

In addition to `init`, the CLI also has other commands:

- `add` - add an addon and register it
- `info` - print out system information for bug reports
- `migrate` - run codemods to migrate your code

See the command-line help with `-h` for details.

---

## [Yarn](https://github.com/yarnpkg/yarn) support

The CLI supports yarn. If you have installed yarn in your system and your project has a `yarn.lock` file, it'll detect it and use `yarn` to install dependencies.

If you don't want to use `yarn` always you can use the `--package-manager` option like this:

```sh
npx sb init --package-manager=npm
```

If you would like to force a particular version of yarn, you can use the `--package-manager` flag with a value of `yarn1` or `yarn2`.

---

## [PNPM](https://pnpm.io/) support

The CLI supports pnpm. If you have installed pnpm in your system and your project has a `pnpm-lock.yaml` file, it'll detect it and use `pnpm` to install dependencies.

If you don't have a lock file and would like to force pnpm to be used, you can use the `--package-manager` option like this:

```sh
npx sb init --package-manager=pnpm
```

---

## [Flow](https://flow.org/) support

It also supports flow files. By default, [jscodeshift](https://github.com/facebook/jscodeshift), the tool used to transform the source files, uses babel to read the files. To be able to transform any flow annotated file, you need to use the flow parser.

```sh
npx sb init --parser flow
```

For more information visit: [storybook.js.org](https://storybook.js.org)

---

## Manually specify project type

If the CLI cannot detect your project type, it will ask you. You can also force it to use a particular project type:

```sh
npx sb init --type <type>
```

Where type is one of the project types defined in [project_types.ts](https://github.com/storybookjs/storybook/blob/next/code/lib/cli/src/project_types.ts)
