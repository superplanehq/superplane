const path = require('path');
const { ScriptTransformer } = require('@jest/transform');
const { dedent } = require('ts-dedent');

const { compile } = require('@storybook/mdx2-csf');

module.exports = {
  async processAsync(src, filename, config, { instrument }) {
    const code = await compile(src, { skipCsf: false });
    const result = dedent`
      /* @jsx mdx */
      import React from 'react'
      import { mdx } from '@mdx-js/react'
      ${code}
    `;

    const extension = path.extname(filename);
    const jsFileName = `${filename.slice(0, -extension.length)}.js`;

    return new ScriptTransformer(config).transformSource(jsFileName, result, instrument);
  },
};
