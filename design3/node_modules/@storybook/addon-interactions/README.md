# Storybook Addon Interactions

Storybook Addon Interactions enables visual debugging of interactions and tests in [Storybook](https://storybook.js.org).

![Screenshot](https://user-images.githubusercontent.com/321738/135628189-3d101cba-50bc-49dc-bba0-776586fedaf3.png)

## Installation

Install this addon by adding the `@storybook/addon-interactions` dependency:

```sh
yarn add -D @storybook/addon-interactions @storybook/jest @storybook/testing-library
```

within `.storybook/main.js`:

```js
export default {
  addons: ['@storybook/addon-interactions'],
};
```

Note that `@storybook/addon-interactions` must be listed **after** `@storybook/addon-actions` or `@storybook/addon-essentials`.

## Usage

Interactions relies on "instrumented" versions of Jest and Testing Library, that you import from `@storybook/jest` and
`@storybook/testing-library` instead of their original package. You can then use these libraries in your `play` function.

```js
import { Button } from './Button';
import { expect } from '@storybook/jest';
import { within, userEvent } from '@storybook/testing-library';

export default {
  title: 'Button',
  component: Button,
  argTypes: {
    onClick: { action: true },
  },
};

const Template = (args) => <Button {...args} />;

export const Demo = Template.bind({});
Demo.play = async ({ args, canvasElement }) => {
  const canvas = within(canvasElement);
  await userEvent.click(canvas.getByRole('button'));
  await expect(args.onClick).toHaveBeenCalled();
};
```

In order to enable step-through debugging, calls to `userEvent.*`, `fireEvent`, `findBy*`, `waitFor*` and `expect` have to
be `await`-ed. While debugging, these functions return a Promise that won't resolve until you continue to the next step.

While you can technically use `screen`, it's recommended to use `within(canvasElement)`. Besides giving you a better error
message when a DOM element can't be found, it will also ensure your play function is compatible with Storybook Docs.

Any `args` that are marked as an `action` (typically via `argTypes` or `argTypesRegex`) will be automatically wrapped in
a [Jest mock function](https://jestjs.io/docs/jest-object#jestfnimplementation) so you can assert invocations. See
[addon-actions](https://storybook.js.org/docs/react/essentials/actions) for how to setup actions.
