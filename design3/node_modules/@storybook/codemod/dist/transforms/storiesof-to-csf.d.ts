/**
 * Convert a legacy story API to component story format
 *
 * For example:
 *
 * ```
 * input { Button } from './Button';
 * storiesOf('Button', module).add('story', () => <Button label="The Button" />);
 * ```
 *
 * Becomes:
 *
 * ```
 * input { Button } from './Button';
 * export default {
 *   title: 'Button'
 * }
 * export const story = () => <Button label="The Button" />;
 *
 * NOTES: only support chained storiesOf() calls
 */
declare function transformer(file: any, api: any, options: any): any;

export { transformer as default };
