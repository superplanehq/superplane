import { global as globalThis } from '@storybook/global';

import { Button } from './Button.jsx';
import { Pre } from './Pre.jsx';
import { Form } from './Form.jsx';
import { Html } from './Html.jsx';

globalThis.Components = { Button, Pre, Form, Html };
globalThis.storybookRenderer = 'react';
