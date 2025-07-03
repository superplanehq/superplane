const { compile } = require('./dist/index');

// FIXME: we shouldn't be doing this, but we need it
// for react MDX story definitions, e.g.
//
// <Story name="foo"><div>hi></div></Story>
//
// Which generates the code:
//
// export const foo = () => <div>hi</div>;
const DEFAULT_RENDERER = `
import React from 'react';
`;

// Lifted from MDXv1 loader
// https://github.com/mdx-js/mdx/blob/v1/packages/loader/index.js
//
// Added
// - webpack5 support
// - MDX compiler built in
const loader = async function (content) {
  const callback = this.async();
  const options = Object.assign({}, this.getOptions(), {
    filepath: this.resourcePath,
  });

  let result;
  try {
    result = await compile(content, options);
  } catch (err) {
    console.error('Error loading:', this.resourcePath)
    return callback(err);
  }

  const code = `${DEFAULT_RENDERER}\n${result}`;
  return callback(null, code);
};

module.exports = loader;
