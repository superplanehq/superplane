/**
 * Adds a `component` parameter for each storiesOf(...) call.
 *
 * For example:
 *
 * input { Button } from './Button';
 * storiesOf('Button', module).add('story', () => <Button label="The Button" />);
 *
 * Becomes:
 *
 * input { Button } from './Button';
 * storiesOf('Button', module)
 *   .addParameters({ component: Button })
 *   .add('story', () => <Button label="The Button" />);
 *
 * Heuristics:
 * - The storiesOf "kind" name must be Button
 * - Button must be imported in the file
 */
declare function transformer(file: any, api: any): any;

export { transformer as default };
