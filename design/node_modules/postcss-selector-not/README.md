# PostCSS Selector Not [<img src="https://postcss.github.io/postcss/logo.svg" alt="PostCSS Logo" width="90" height="90" align="right">][postcss]

[<img alt="npm version" src="https://img.shields.io/npm/v/postcss-selector-not.svg" height="20">][npm-url] [<img alt="CSS Standard Status" src="https://cssdb.org/images/badges/not-pseudo-class.svg" height="20">][css-url] [<img alt="Build Status" src="https://github.com/csstools/postcss-plugins/workflows/test/badge.svg" height="20">][cli-url] [<img alt="Discord" src="https://shields.io/badge/Discord-5865F2?logo=discord&logoColor=white">][discord]

[PostCSS Selector Not] transforms :not() W3C CSS level 4 pseudo classes to :not() CSS level 3 selectors following the [Selectors 4 Specification].

```pcss
p:not(:first-child, .special) {
	color: red;
}

/* becomes */

p:not(:first-child):not(.special) {
	color: red;
}
```

⚠️ Only lists of simple selectors (`:not(.a, .b)`) will work as expected.
Complex selectors (`:not(.a > .b, .c ~ .d)`) can not be downgraded.

## Usage

Add [PostCSS Selector Not] to your project:

```bash
npm install postcss postcss-selector-not --save-dev
```

Use it as a [PostCSS] plugin:

```js
const postcss = require('postcss');
const postcssSelectorNot = require('postcss-selector-not');

postcss([
	postcssSelectorNot(/* pluginOptions */)
]).process(YOUR_CSS /*, processOptions */);
```

[PostCSS Selector Not] runs in all Node environments, with special
instructions for:

| [Node](INSTALL.md#node) | [PostCSS CLI](INSTALL.md#postcss-cli) | [Webpack](INSTALL.md#webpack) | [Create React App](INSTALL.md#create-react-app) | [Gulp](INSTALL.md#gulp) | [Grunt](INSTALL.md#grunt) |
| --- | --- | --- | --- | --- | --- |

[cli-url]: https://github.com/csstools/postcss-plugins/actions/workflows/test.yml?query=workflow/test
[css-url]: https://cssdb.org/#not-pseudo-class
[discord]: https://discord.gg/bUadyRwkJS
[npm-url]: https://www.npmjs.com/package/postcss-selector-not

[Gulp PostCSS]: https://github.com/postcss/gulp-postcss
[Grunt PostCSS]: https://github.com/nDmitry/grunt-postcss
[PostCSS]: https://github.com/postcss/postcss
[PostCSS Loader]: https://github.com/postcss/postcss-loader
[PostCSS Selector Not]: https://github.com/csstools/postcss-plugins/tree/main/plugins/postcss-selector-not
[Selectors 4 Specification]: https://www.w3.org/TR/selectors-4/#negation-pseudo
