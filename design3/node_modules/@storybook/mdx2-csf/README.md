## @storybook/mdx2-csf

Storybook's `mdx2-csf` is a compiler that turns MDXv2 input into CSF output.

For example, the following input:

```mdx
import { Meta, Story } from '@storybook/addon-docs';

<Meta title="atoms/Button" />

<Story name="Bar">
  <Button>hello</Button>
</Story>
```

Might be transformed into the following CSF (over-simplified):

```js
export default {
  title: 'atoms/Button',
};

export const Bar = () => <Button>hello</Button>;
```

## API

This library exports an async function to compile MDX, `compile`.
It does not support a synchronous compiler because it uses asynchronous
imports to bridge the ESM/CJS gap. The underlying MDXv2 libraries only
support pure ESM, but this library is used in CJS environments.

### compile

Asynchronously compile a string:

```js
import { compile } from '@storybook/mdx2-csf';

const code = '# hello\n\nworld';
const output = await compile(code);
```

## Loader

In addition, this library supports a simple Webpack loader that mirrors MDXv1's loader, but adds Webpack5 support. It doesn't use MDXv2's loader because it is prohibitively complex, and we want this to be interchangeable with the `@storybook/mdx1-csf`'s loader which is also based on the MDXv1 loader.

The loader takes two options:

- `skipCsf` don't generate CSF stories for the MDX file
- `mdxCompileOptions` full options for the [MDX compile function](https://mdxjs.com/packages/mdx/#api)

For example, to add [GFM support](https://mdxjs.com/guides/gfm/):

```js
import remarkGfm from 'remark-gfm';

module.exports = {
  module: {
    rules: [
      {
        test: /\.(stories|story)\.mdx$/,
        use: [
          {
            loader: require.resolve('@storybook/mdx2-csf/loader'),
            options: {
              skipCsf: false,
              mdxCompileOptions: {
                remarkPlugins: [remarkGfm],
              },
            },
          },
        ],
      },
    ],
  },
};
```

## Contributing

We welcome contributions to Storybook!

- 📥 Pull requests and 🌟 Stars are always welcome.
- Read our [contributing guide](CONTRIBUTING.md) to get started,
  or find us on [Discord](https://discord.gg/storybook), we will take the time to guide you

## License

[MIT](https://github.com/storybookjs/csf-mdx2/blob/main/LICENSE)
