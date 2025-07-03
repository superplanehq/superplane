declare const wrapperJs: string;

// Type definitions for non-npm package estree 1.0
// Project: https://github.com/estree/estree
// Definitions by: RReverser <https://github.com/RReverser>
// Definitions: https://github.com/DefinitelyTyped/DefinitelyTyped

// This definition file follows a somewhat unusual format. ESTree allows
// runtime type checks based on the `type` parameter. In order to explain this
// to typescript we want to use discriminated union types:
// https://github.com/Microsoft/TypeScript/pull/9163
//
// For ESTree this is a bit tricky because the high level interfaces like
// Node or Function are pulling double duty. We want to pass common fields down
// to the interfaces that extend them (like Identifier or
// ArrowFunctionExpression), but you can't extend a type union or enforce
// common fields on them. So we've split the high level interfaces into two
// types, a base type which passes down inherited fields, and a type union of
// all types which extend the base type. Only the type union is exported, and
// the union is how other types refer to the collection of inheriting types.
//
// This makes the definitions file here somewhat more difficult to maintain,
// but it has the notable advantage of making ESTree much easier to use as
// an end user.

interface BaseNodeWithoutComments {
    // Every leaf interface that extends BaseNode must specify a type property.
    // The type property should be a string literal. For example, Identifier
    // has: `type: "Identifier"`
    type: string;
    loc?: SourceLocation | null | undefined;
    range?: [number, number] | undefined;
}

interface BaseNode extends BaseNodeWithoutComments {
    leadingComments?: Comment$1[] | undefined;
    trailingComments?: Comment$1[] | undefined;
}

interface Comment$1 extends BaseNodeWithoutComments {
    type: 'Line' | 'Block';
    value: string;
}

interface SourceLocation {
    source?: string | null | undefined;
    start: Position$4;
    end: Position$4;
}

interface Position$4 {
    /** >= 1 */
    line: number;
    /** >= 0 */
    column: number;
}

interface BaseFunction extends BaseNode {
    params: Pattern[];
    generator?: boolean | undefined;
    async?: boolean | undefined;
    // The body is either BlockStatement or Expression because arrow functions
    // can have a body that's either. FunctionDeclarations and
    // FunctionExpressions have only BlockStatement bodies.
    body: BlockStatement | Expression;
}

type Statement =
    | ExpressionStatement
    | BlockStatement
    | StaticBlock
    | EmptyStatement
    | DebuggerStatement
    | WithStatement
    | ReturnStatement
    | LabeledStatement
    | BreakStatement
    | ContinueStatement
    | IfStatement
    | SwitchStatement
    | ThrowStatement
    | TryStatement
    | WhileStatement
    | DoWhileStatement
    | ForStatement
    | ForInStatement
    | ForOfStatement
    | Declaration;

interface BaseStatement extends BaseNode {}

interface EmptyStatement extends BaseStatement {
    type: 'EmptyStatement';
}

interface BlockStatement extends BaseStatement {
    type: 'BlockStatement';
    body: Statement[];
    innerComments?: Comment$1[] | undefined;
}

interface StaticBlock extends Omit<BlockStatement, 'type'> {
    type: 'StaticBlock';
}

interface ExpressionStatement extends BaseStatement {
    type: 'ExpressionStatement';
    expression: Expression;
}

interface IfStatement extends BaseStatement {
    type: 'IfStatement';
    test: Expression;
    consequent: Statement;
    alternate?: Statement | null | undefined;
}

interface LabeledStatement extends BaseStatement {
    type: 'LabeledStatement';
    label: Identifier;
    body: Statement;
}

interface BreakStatement extends BaseStatement {
    type: 'BreakStatement';
    label?: Identifier | null | undefined;
}

interface ContinueStatement extends BaseStatement {
    type: 'ContinueStatement';
    label?: Identifier | null | undefined;
}

interface WithStatement extends BaseStatement {
    type: 'WithStatement';
    object: Expression;
    body: Statement;
}

interface SwitchStatement extends BaseStatement {
    type: 'SwitchStatement';
    discriminant: Expression;
    cases: SwitchCase[];
}

interface ReturnStatement extends BaseStatement {
    type: 'ReturnStatement';
    argument?: Expression | null | undefined;
}

interface ThrowStatement extends BaseStatement {
    type: 'ThrowStatement';
    argument: Expression;
}

interface TryStatement extends BaseStatement {
    type: 'TryStatement';
    block: BlockStatement;
    handler?: CatchClause | null | undefined;
    finalizer?: BlockStatement | null | undefined;
}

interface WhileStatement extends BaseStatement {
    type: 'WhileStatement';
    test: Expression;
    body: Statement;
}

interface DoWhileStatement extends BaseStatement {
    type: 'DoWhileStatement';
    body: Statement;
    test: Expression;
}

interface ForStatement extends BaseStatement {
    type: 'ForStatement';
    init?: VariableDeclaration | Expression | null | undefined;
    test?: Expression | null | undefined;
    update?: Expression | null | undefined;
    body: Statement;
}

interface BaseForXStatement extends BaseStatement {
    left: VariableDeclaration | Pattern;
    right: Expression;
    body: Statement;
}

interface ForInStatement extends BaseForXStatement {
    type: 'ForInStatement';
}

interface DebuggerStatement extends BaseStatement {
    type: 'DebuggerStatement';
}

type Declaration = FunctionDeclaration | VariableDeclaration | ClassDeclaration;

interface BaseDeclaration extends BaseStatement {}

interface FunctionDeclaration extends BaseFunction, BaseDeclaration {
    type: 'FunctionDeclaration';
    /** It is null when a function declaration is a part of the `export default function` statement */
    id: Identifier | null;
    body: BlockStatement;
}

interface VariableDeclaration extends BaseDeclaration {
    type: 'VariableDeclaration';
    declarations: VariableDeclarator[];
    kind: 'var' | 'let' | 'const';
}

interface VariableDeclarator extends BaseNode {
    type: 'VariableDeclarator';
    id: Pattern;
    init?: Expression | null | undefined;
}

interface ExpressionMap {
    ArrayExpression: ArrayExpression;
    ArrowFunctionExpression: ArrowFunctionExpression;
    AssignmentExpression: AssignmentExpression;
    AwaitExpression: AwaitExpression;
    BinaryExpression: BinaryExpression;
    CallExpression: CallExpression;
    ChainExpression: ChainExpression;
    ClassExpression: ClassExpression;
    ConditionalExpression: ConditionalExpression;
    FunctionExpression: FunctionExpression;
    Identifier: Identifier;
    ImportExpression: ImportExpression;
    Literal: Literal$3;
    LogicalExpression: LogicalExpression;
    MemberExpression: MemberExpression;
    MetaProperty: MetaProperty;
    NewExpression: NewExpression;
    ObjectExpression: ObjectExpression;
    SequenceExpression: SequenceExpression;
    TaggedTemplateExpression: TaggedTemplateExpression;
    TemplateLiteral: TemplateLiteral;
    ThisExpression: ThisExpression;
    UnaryExpression: UnaryExpression;
    UpdateExpression: UpdateExpression;
    YieldExpression: YieldExpression;
}

type Expression = ExpressionMap[keyof ExpressionMap];

interface BaseExpression extends BaseNode {}

type ChainElement = SimpleCallExpression | MemberExpression;

interface ChainExpression extends BaseExpression {
    type: 'ChainExpression';
    expression: ChainElement;
}

interface ThisExpression extends BaseExpression {
    type: 'ThisExpression';
}

interface ArrayExpression extends BaseExpression {
    type: 'ArrayExpression';
    elements: Array<Expression | SpreadElement | null>;
}

interface ObjectExpression extends BaseExpression {
    type: 'ObjectExpression';
    properties: Array<Property | SpreadElement>;
}

interface PrivateIdentifier extends BaseNode {
    type: 'PrivateIdentifier';
    name: string;
}

interface Property extends BaseNode {
    type: 'Property';
    key: Expression | PrivateIdentifier;
    value: Expression | Pattern; // Could be an AssignmentProperty
    kind: 'init' | 'get' | 'set';
    method: boolean;
    shorthand: boolean;
    computed: boolean;
}

interface PropertyDefinition extends BaseNode {
    type: 'PropertyDefinition';
    key: Expression | PrivateIdentifier;
    value?: Expression | null | undefined;
    computed: boolean;
    static: boolean;
}

interface FunctionExpression extends BaseFunction, BaseExpression {
    id?: Identifier | null | undefined;
    type: 'FunctionExpression';
    body: BlockStatement;
}

interface SequenceExpression extends BaseExpression {
    type: 'SequenceExpression';
    expressions: Expression[];
}

interface UnaryExpression extends BaseExpression {
    type: 'UnaryExpression';
    operator: UnaryOperator;
    prefix: true;
    argument: Expression;
}

interface BinaryExpression extends BaseExpression {
    type: 'BinaryExpression';
    operator: BinaryOperator;
    left: Expression;
    right: Expression;
}

interface AssignmentExpression extends BaseExpression {
    type: 'AssignmentExpression';
    operator: AssignmentOperator;
    left: Pattern | MemberExpression;
    right: Expression;
}

interface UpdateExpression extends BaseExpression {
    type: 'UpdateExpression';
    operator: UpdateOperator;
    argument: Expression;
    prefix: boolean;
}

interface LogicalExpression extends BaseExpression {
    type: 'LogicalExpression';
    operator: LogicalOperator;
    left: Expression;
    right: Expression;
}

interface ConditionalExpression extends BaseExpression {
    type: 'ConditionalExpression';
    test: Expression;
    alternate: Expression;
    consequent: Expression;
}

interface BaseCallExpression extends BaseExpression {
    callee: Expression | Super;
    arguments: Array<Expression | SpreadElement>;
}
type CallExpression = SimpleCallExpression | NewExpression;

interface SimpleCallExpression extends BaseCallExpression {
    type: 'CallExpression';
    optional: boolean;
}

interface NewExpression extends BaseCallExpression {
    type: 'NewExpression';
}

interface MemberExpression extends BaseExpression, BasePattern {
    type: 'MemberExpression';
    object: Expression | Super;
    property: Expression | PrivateIdentifier;
    computed: boolean;
    optional: boolean;
}

type Pattern = Identifier | ObjectPattern | ArrayPattern | RestElement | AssignmentPattern | MemberExpression;

interface BasePattern extends BaseNode {}

interface SwitchCase extends BaseNode {
    type: 'SwitchCase';
    test?: Expression | null | undefined;
    consequent: Statement[];
}

interface CatchClause extends BaseNode {
    type: 'CatchClause';
    param: Pattern | null;
    body: BlockStatement;
}

interface Identifier extends BaseNode, BaseExpression, BasePattern {
    type: 'Identifier';
    name: string;
}

type Literal$3 = SimpleLiteral | RegExpLiteral | BigIntLiteral;

interface SimpleLiteral extends BaseNode, BaseExpression {
    type: 'Literal';
    value: string | boolean | number | null;
    raw?: string | undefined;
}

interface RegExpLiteral extends BaseNode, BaseExpression {
    type: 'Literal';
    value?: RegExp | null | undefined;
    regex: {
        pattern: string;
        flags: string;
    };
    raw?: string | undefined;
}

interface BigIntLiteral extends BaseNode, BaseExpression {
    type: 'Literal';
    value?: bigint | null | undefined;
    bigint: string;
    raw?: string | undefined;
}

type UnaryOperator = '-' | '+' | '!' | '~' | 'typeof' | 'void' | 'delete';

type BinaryOperator =
    | '=='
    | '!='
    | '==='
    | '!=='
    | '<'
    | '<='
    | '>'
    | '>='
    | '<<'
    | '>>'
    | '>>>'
    | '+'
    | '-'
    | '*'
    | '/'
    | '%'
    | '**'
    | '|'
    | '^'
    | '&'
    | 'in'
    | 'instanceof';

type LogicalOperator = '||' | '&&' | '??';

type AssignmentOperator =
    | '='
    | '+='
    | '-='
    | '*='
    | '/='
    | '%='
    | '**='
    | '<<='
    | '>>='
    | '>>>='
    | '|='
    | '^='
    | '&=';

type UpdateOperator = '++' | '--';

interface ForOfStatement extends BaseForXStatement {
    type: 'ForOfStatement';
    await: boolean;
}

interface Super extends BaseNode {
    type: 'Super';
}

interface SpreadElement extends BaseNode {
    type: 'SpreadElement';
    argument: Expression;
}

interface ArrowFunctionExpression extends BaseExpression, BaseFunction {
    type: 'ArrowFunctionExpression';
    expression: boolean;
    body: BlockStatement | Expression;
}

interface YieldExpression extends BaseExpression {
    type: 'YieldExpression';
    argument?: Expression | null | undefined;
    delegate: boolean;
}

interface TemplateLiteral extends BaseExpression {
    type: 'TemplateLiteral';
    quasis: TemplateElement[];
    expressions: Expression[];
}

interface TaggedTemplateExpression extends BaseExpression {
    type: 'TaggedTemplateExpression';
    tag: Expression;
    quasi: TemplateLiteral;
}

interface TemplateElement extends BaseNode {
    type: 'TemplateElement';
    tail: boolean;
    value: {
        /** It is null when the template literal is tagged and the text has an invalid escape (e.g. - tag`\unicode and \u{55}`) */
        cooked?: string | null | undefined;
        raw: string;
    };
}

interface AssignmentProperty extends Property {
    value: Pattern;
    kind: 'init';
    method: boolean; // false
}

interface ObjectPattern extends BasePattern {
    type: 'ObjectPattern';
    properties: Array<AssignmentProperty | RestElement>;
}

interface ArrayPattern extends BasePattern {
    type: 'ArrayPattern';
    elements: Array<Pattern | null>;
}

interface RestElement extends BasePattern {
    type: 'RestElement';
    argument: Pattern;
}

interface AssignmentPattern extends BasePattern {
    type: 'AssignmentPattern';
    left: Pattern;
    right: Expression;
}
interface BaseClass extends BaseNode {
    superClass?: Expression | null | undefined;
    body: ClassBody;
}

interface ClassBody extends BaseNode {
    type: 'ClassBody';
    body: Array<MethodDefinition | PropertyDefinition | StaticBlock>;
}

interface MethodDefinition extends BaseNode {
    type: 'MethodDefinition';
    key: Expression | PrivateIdentifier;
    value: FunctionExpression;
    kind: 'constructor' | 'method' | 'get' | 'set';
    computed: boolean;
    static: boolean;
}

interface ClassDeclaration extends BaseClass, BaseDeclaration {
    type: 'ClassDeclaration';
    /** It is null when a class declaration is a part of the `export default class` statement */
    id: Identifier | null;
}

interface ClassExpression extends BaseClass, BaseExpression {
    type: 'ClassExpression';
    id?: Identifier | null | undefined;
}

interface MetaProperty extends BaseExpression {
    type: 'MetaProperty';
    meta: Identifier;
    property: Identifier;
}

interface ImportExpression extends BaseExpression {
    type: 'ImportExpression';
    source: Expression;
}

interface AwaitExpression extends BaseExpression {
    type: 'AwaitExpression';
    argument: Expression;
}

// Type definitions for non-npm package estree-jsx 1.0


declare module 'estree' {
    interface ExpressionMap {
        JSXElement: JSXElement;
    }

    interface NodeMap {
        JSXIdentifier: JSXIdentifier;
        JSXNamespacedName: JSXNamespacedName;
        JSXMemberExpression: JSXMemberExpression;
        JSXEmptyExpression: JSXEmptyExpression;
        JSXExpressionContainer: JSXExpressionContainer;
        JSXSpreadAttribute: JSXSpreadAttribute;
        JSXAttribute: JSXAttribute;
        JSXOpeningElement: JSXOpeningElement;
        JSXOpeningFragment: JSXOpeningFragment;
        JSXClosingElement: JSXClosingElement;
        JSXClosingFragment: JSXClosingFragment;
        JSXElement: JSXElement;
        JSXFragment: JSXFragment;
        JSXText: JSXText;
    }
}

interface JSXIdentifier extends BaseNode {
    type: 'JSXIdentifier';
    name: string;
}

interface JSXMemberExpression extends BaseExpression {
    type: 'JSXMemberExpression';
    object: JSXMemberExpression | JSXIdentifier;
    property: JSXIdentifier;
}

interface JSXNamespacedName extends BaseExpression {
    type: 'JSXNamespacedName';
    namespace: JSXIdentifier;
    name: JSXIdentifier;
}

interface JSXEmptyExpression extends BaseNode {
    type: 'JSXEmptyExpression';
}

interface JSXExpressionContainer extends BaseNode {
    type: 'JSXExpressionContainer';
    expression: Expression | JSXEmptyExpression;
}

interface JSXSpreadChild extends BaseNode {
    type: 'JSXSpreadChild';
    expression: Expression;
}

interface JSXBoundaryElement extends BaseNode {
    name: JSXIdentifier | JSXMemberExpression | JSXNamespacedName;
}

interface JSXOpeningElement extends JSXBoundaryElement {
    type: 'JSXOpeningElement';
    attributes: Array<JSXAttribute | JSXSpreadAttribute>;
    selfClosing: boolean;
}

interface JSXClosingElement extends JSXBoundaryElement {
    type: 'JSXClosingElement';
}

interface JSXAttribute extends BaseNode {
    type: 'JSXAttribute';
    name: JSXIdentifier | JSXNamespacedName;
    value: Literal$3 | JSXExpressionContainer | JSXElement | JSXFragment | null;
}

interface JSXSpreadAttribute extends BaseNode {
    type: 'JSXSpreadAttribute';
    argument: Expression;
}

interface JSXText extends BaseNode {
    type: 'JSXText';
    value: string;
    raw: string;
}

interface JSXElement extends BaseExpression {
    type: 'JSXElement';
    openingElement: JSXOpeningElement;
    children: Array<JSXText | JSXExpressionContainer | JSXSpreadChild | JSXElement | JSXFragment>;
    closingElement: JSXClosingElement | null;
}

interface JSXFragment extends BaseExpression {
    type: 'JSXFragment';
    openingFragment: JSXOpeningFragment;
    children: Array<JSXText | JSXExpressionContainer | JSXSpreadChild | JSXElement | JSXFragment>;
    closingFragment: JSXClosingFragment;
}

interface JSXOpeningFragment extends BaseNode {
    type: 'JSXOpeningFragment';
}

interface JSXClosingFragment extends BaseNode {
    type: 'JSXClosingFragment';
}

// Type definitions for non-npm package Unist 2.0
// Project: https://github.com/syntax-tree/unist
// Definitions by: bizen241 <https://github.com/bizen241>
//                 Jun Lu <https://github.com/lujun2>
//                 Hernan Rajchert <https://github.com/hrajchert>
//                 Titus Wormer <https://github.com/wooorm>
//                 Junyoung Choi <https://github.com/rokt33r>
//                 Ben Moon <https://github.com/GuiltyDolphin>
//                 JounQin <https://github.com/JounQin>
// Definitions: https://github.com/DefinitelyTyped/DefinitelyTyped
// TypeScript Version: 3.0

/**
 * Syntactic units in unist syntax trees are called nodes.
 *
 * @typeParam TData Information from the ecosystem. Useful for more specific {@link Node.data}.
 */
interface Node$2<TData extends object = Data$2> {
    /**
     * The variant of a node.
     */
    type: string;

    /**
     * Information from the ecosystem.
     */
    data?: TData | undefined;

    /**
     * Location of a node in a source document.
     * Must not be present if a node is generated.
     */
    position?: Position$3 | undefined;
}

/**
 * Information associated by the ecosystem with the node.
 * Space is guaranteed to never be specified by unist or specifications
 * implementing unist.
 */
interface Data$2 {
    [key: string]: unknown;
}

/**
 * Location of a node in a source file.
 */
interface Position$3 {
    /**
     * Place of the first character of the parsed source region.
     */
    start: Point$2;

    /**
     * Place of the first character after the parsed source region.
     */
    end: Point$2;

    /**
     * Start column at each index (plus start line) in the source region,
     * for elements that span multiple lines.
     */
    indent?: number[] | undefined;
}

/**
 * One place in a source file.
 */
interface Point$2 {
    /**
     * Line in a source file (1-indexed integer).
     */
    line: number;

    /**
     * Column in a source file (1-indexed integer).
     */
    column: number;
    /**
     * Character in a source file (0-indexed integer).
     */
    offset?: number | undefined;
}

/**
 * Util for extracting type of {@link Node.data}
 *
 * @typeParam TNode Specific node type such as {@link Node} with {@link Data}, {@link Literal}, etc.
 *
 * @example `NodeData<Node<{ key: string }>>` -> `{ key: string }`
 */
type NodeData<TNode extends Node$2<object>> = TNode extends Node$2<infer TData> ? TData : never;

/**
 * Nodes containing other nodes.
 *
 * @typeParam ChildNode Node item of {@link Parent.children}
 */
interface Parent$2<ChildNode extends Node$2<object> = Node$2, TData extends object = NodeData<ChildNode>>
    extends Node$2<TData> {
    /**
     * List representing the children of a node.
     */
    children: ChildNode[];
}

/**
 * Nodes containing a value.
 *
 * @typeParam Value Specific value type of {@link Literal.value} such as `string` for `Text` node
 */
interface Literal$2<Value = unknown, TData extends object = Data$2> extends Node$2<TData> {
    value: Value;
}

/**
 * This is the same as `Buffer` if node types are included, `never` otherwise.
 */
// eslint-disable-next-line @typescript-eslint/ban-ts-comment, @typescript-eslint/prefer-ts-expect-error
// @ts-ignore It’s important to preserve this ignore statement. This makes sure
// it works both with and without node types.
// eslint-disable-next-line n/prefer-global/buffer
type MaybeBuffer = any extends Buffer ? never : Buffer

/**
 * Contents of the file.
 *
 * Can either be text or a `Buffer` structure.
 */
// Note: this does not directly use type `Buffer`, because it can also be used
// in a browser context.
// Instead this leverages `Uint8Array` which is the base type for `Buffer`,
// and a native JavaScript construct.
type Value$1 = string | MaybeBuffer

/**
 * This map registers the type of the `data` key of a `VFile`.
 *
 * This type can be augmented to register custom `data` types.
 *
 * @example
 * declare module 'vfile' {
 *   interface DataMap {
 *     // `file.data.name` is typed as `string`
 *     name: string
 *   }
 * }
 */

// eslint-disable-next-line @typescript-eslint/consistent-type-definitions, @typescript-eslint/no-empty-interface
interface DataMap {}

/**
 * Custom information.
 *
 * Known attributes can be added to @see {@link DataMap}
 */
type Data$1 = Record<string, unknown> & Partial<DataMap>

type URL$1 = {
  hash: string
  host: string
  hostname: string
  href: string
  origin: string
  password: string
  pathname: string
  port: string
  protocol: string
  search: string
  searchParams: any
  username: string
  toString: () => string
  toJSON: () => string
}

/**
 * Message.
 */
declare class VFileMessage extends Error {
  /**
   * Create a message for `reason` at `place` from `origin`.
   *
   * When an error is passed in as `reason`, the `stack` is copied.
   *
   * @param {string | Error | VFileMessage} reason
   *   Reason for message, uses the stack and message of the error if given.
   *
   *   > 👉 **Note**: you should use markdown.
   * @param {Node | NodeLike | Position | Point | null | undefined} [place]
   *   Place in file where the message occurred.
   * @param {string | null | undefined} [origin]
   *   Place in code where the message originates (example:
   *   `'my-package:my-rule'` or `'my-rule'`).
   * @returns
   *   Instance of `VFileMessage`.
   */
  constructor(
    reason: string | Error | VFileMessage,
    place?: Node$1 | NodeLike$1 | Position$2 | Point$1 | null | undefined,
    origin?: string | null | undefined
  )
  /**
   * Stack of message.
   *
   * This is used by normal errors to show where something happened in
   * programming code, irrelevant for `VFile` messages,
   *
   * @type {string}
   */
  stack: string
  /**
   * Reason for message.
   *
   * > 👉 **Note**: you should use markdown.
   *
   * @type {string}
   */
  reason: string
  /**
   * State of problem.
   *
   * * `true` — marks associated file as no longer processable (error)
   * * `false` — necessitates a (potential) change (warning)
   * * `null | undefined` — for things that might not need changing (info)
   *
   * @type {boolean | null | undefined}
   */
  fatal: boolean | null | undefined
  /**
   * Starting line of error.
   *
   * @type {number | null}
   */
  line: number | null
  /**
   * Starting column of error.
   *
   * @type {number | null}
   */
  column: number | null
  /**
   * Full unist position.
   *
   * @type {Position | null}
   */
  position: Position$2 | null
  /**
   * Namespace of message (example: `'my-package'`).
   *
   * @type {string | null}
   */
  source: string | null
  /**
   * Category of message (example: `'my-rule'`).
   *
   * @type {string | null}
   */
  ruleId: string | null
  /**
   * Path of a file (used throughout the `VFile` ecosystem).
   *
   * @type {string | null}
   */
  file: string | null
  /**
   * Specify the source value that’s being reported, which is deemed
   * incorrect.
   *
   * @type {string | null}
   */
  actual: string | null
  /**
   * Suggest acceptable values that can be used instead of `actual`.
   *
   * @type {Array<string> | null}
   */
  expected: Array<string> | null
  /**
   * Link to docs for the message.
   *
   * > 👉 **Note**: this must be an absolute URL that can be passed as `x`
   * > to `new URL(x)`.
   *
   * @type {string | null}
   */
  url: string | null
  /**
   * Long form description of the message (you should use markdown).
   *
   * @type {string | null}
   */
  note: string | null
}
type Node$1 = Node$2
type Position$2 = Position$3
type Point$1 = Point$2
type NodeLike$1 = object & {
  type: string
  position?: Position$2 | undefined
}

declare class VFile$1 {
  /**
   * Create a new virtual file.
   *
   * `options` is treated as:
   *
   * *   `string` or `Buffer` — `{value: options}`
   * *   `URL` — `{path: options}`
   * *   `VFile` — shallow copies its data over to the new file
   * *   `object` — all fields are shallow copied over to the new file
   *
   * Path related fields are set in the following order (least specific to
   * most specific): `history`, `path`, `basename`, `stem`, `extname`,
   * `dirname`.
   *
   * You cannot set `dirname` or `extname` without setting either `history`,
   * `path`, `basename`, or `stem` too.
   *
   * @param {Compatible | null | undefined} [value]
   *   File value.
   * @returns
   *   New instance.
   */
  constructor(value?: Compatible | null | undefined)
  /**
   * Place to store custom information (default: `{}`).
   *
   * It’s OK to store custom data directly on the file but moving it to
   * `data` is recommended.
   *
   * @type {Data}
   */
  data: Data
  /**
   * List of messages associated with the file.
   *
   * @type {Array<VFileMessage>}
   */
  messages: Array<VFileMessage>
  /**
   * List of filepaths the file moved between.
   *
   * The first is the original path and the last is the current path.
   *
   * @type {Array<string>}
   */
  history: Array<string>
  /**
   * Base of `path` (default: `process.cwd()` or `'/'` in browsers).
   *
   * @type {string}
   */
  cwd: string
  /**
   * Raw value.
   *
   * @type {Value}
   */
  value: Value
  /**
   * Whether a file was saved to disk.
   *
   * This is used by vfile reporters.
   *
   * @type {boolean}
   */
  stored: boolean
  /**
   * Custom, non-string, compiled, representation.
   *
   * This is used by unified to store non-string results.
   * One example is when turning markdown into React nodes.
   *
   * @type {unknown}
   */
  result: unknown
  /**
   * Source map.
   *
   * This type is equivalent to the `RawSourceMap` type from the `source-map`
   * module.
   *
   * @type {Map | null | undefined}
   */
  map: Map | null | undefined
  /**
   * Set the full path (example: `'~/index.min.js'`).
   *
   * Cannot be nullified.
   * You can set a file URL (a `URL` object with a `file:` protocol) which will
   * be turned into a path with `url.fileURLToPath`.
   *
   * @param {string | URL} path
   */
  set path(arg: string)
  /**
   * Get the full path (example: `'~/index.min.js'`).
   *
   * @returns {string}
   */
  get path(): string
  /**
   * Set the parent path (example: `'~'`).
   *
   * Cannot be set if there’s no `path` yet.
   */
  set dirname(arg: string | undefined)
  /**
   * Get the parent path (example: `'~'`).
   */
  get dirname(): string | undefined
  /**
   * Set basename (including extname) (`'index.min.js'`).
   *
   * Cannot contain path separators (`'/'` on unix, macOS, and browsers, `'\'`
   * on windows).
   * Cannot be nullified (use `file.path = file.dirname` instead).
   */
  set basename(arg: string | undefined)
  /**
   * Get the basename (including extname) (example: `'index.min.js'`).
   */
  get basename(): string | undefined
  /**
   * Set the extname (including dot) (example: `'.js'`).
   *
   * Cannot contain path separators (`'/'` on unix, macOS, and browsers, `'\'`
   * on windows).
   * Cannot be set if there’s no `path` yet.
   */
  set extname(arg: string | undefined)
  /**
   * Get the extname (including dot) (example: `'.js'`).
   */
  get extname(): string | undefined
  /**
   * Set the stem (basename w/o extname) (example: `'index.min'`).
   *
   * Cannot contain path separators (`'/'` on unix, macOS, and browsers, `'\'`
   * on windows).
   * Cannot be nullified (use `file.path = file.dirname` instead).
   */
  set stem(arg: string | undefined)
  /**
   * Get the stem (basename w/o extname) (example: `'index.min'`).
   */
  get stem(): string | undefined
  /**
   * Serialize the file.
   *
   * @param {BufferEncoding | null | undefined} [encoding='utf8']
   *   Character encoding to understand `value` as when it’s a `Buffer`
   *   (default: `'utf8'`).
   * @returns {string}
   *   Serialized file.
   */
  toString(encoding?: BufferEncoding | null | undefined): string
  /**
   * Create a warning message associated with the file.
   *
   * Its `fatal` is set to `false` and `file` is set to the current file path.
   * Its added to `file.messages`.
   *
   * @param {string | Error | VFileMessage} reason
   *   Reason for message, uses the stack and message of the error if given.
   * @param {Node | NodeLike | Position | Point | null | undefined} [place]
   *   Place in file where the message occurred.
   * @param {string | null | undefined} [origin]
   *   Place in code where the message originates (example:
   *   `'my-package:my-rule'` or `'my-rule'`).
   * @returns {VFileMessage}
   *   Message.
   */
  message(
    reason: string | Error | VFileMessage,
    place?: Node | NodeLike | Position$1 | Point | null | undefined,
    origin?: string | null | undefined
  ): VFileMessage
  /**
   * Create an info message associated with the file.
   *
   * Its `fatal` is set to `null` and `file` is set to the current file path.
   * Its added to `file.messages`.
   *
   * @param {string | Error | VFileMessage} reason
   *   Reason for message, uses the stack and message of the error if given.
   * @param {Node | NodeLike | Position | Point | null | undefined} [place]
   *   Place in file where the message occurred.
   * @param {string | null | undefined} [origin]
   *   Place in code where the message originates (example:
   *   `'my-package:my-rule'` or `'my-rule'`).
   * @returns {VFileMessage}
   *   Message.
   */
  info(
    reason: string | Error | VFileMessage,
    place?: Node | NodeLike | Position$1 | Point | null | undefined,
    origin?: string | null | undefined
  ): VFileMessage
  /**
   * Create a fatal error associated with the file.
   *
   * Its `fatal` is set to `true` and `file` is set to the current file path.
   * Its added to `file.messages`.
   *
   * > 👉 **Note**: a fatal error means that a file is no longer processable.
   *
   * @param {string | Error | VFileMessage} reason
   *   Reason for message, uses the stack and message of the error if given.
   * @param {Node | NodeLike | Position | Point | null | undefined} [place]
   *   Place in file where the message occurred.
   * @param {string | null | undefined} [origin]
   *   Place in code where the message originates (example:
   *   `'my-package:my-rule'` or `'my-rule'`).
   * @returns {never}
   *   Message.
   * @throws {VFileMessage}
   *   Message.
   */
  fail(
    reason: string | Error | VFileMessage,
    place?: Node | NodeLike | Position$1 | Point | null | undefined,
    origin?: string | null | undefined
  ): never
}
type Node = Node$2
type Position$1 = Position$3
type Point = Point$2
type URL = URL$1
type Data = Data$1
type Value = Value$1
type NodeLike = Record<string, unknown> & {
  type: string
  position?: Position$1 | undefined
}
/**
 * Encodings supported by the buffer class.
 *
 * This is a copy of the types from Node, copied to prevent Node globals from
 * being needed.
 * Copied from: <https://github.com/DefinitelyTyped/DefinitelyTyped/blob/90a4ec8/types/node/buffer.d.ts#L170>
 */
type BufferEncoding =
  | 'ascii'
  | 'utf8'
  | 'utf-8'
  | 'utf16le'
  | 'ucs2'
  | 'ucs-2'
  | 'base64'
  | 'base64url'
  | 'latin1'
  | 'binary'
  | 'hex'
/**
 * Things that can be passed to the constructor.
 */
type Compatible = Options$4 | URL | Value | VFile$1
/**
 * Set multiple values.
 */
type VFileCoreOptions = {
  /**
   * Set `value`.
   */
  value?: Value | null | undefined
  /**
   * Set `cwd`.
   */
  cwd?: string | null | undefined
  /**
   * Set `history`.
   */
  history?: Array<string> | null | undefined
  /**
   * Set `path`.
   */
  path?: URL | string | null | undefined
  /**
   * Set `basename`.
   */
  basename?: string | null | undefined
  /**
   * Set `stem`.
   */
  stem?: string | null | undefined
  /**
   * Set `extname`.
   */
  extname?: string | null | undefined
  /**
   * Set `dirname`.
   */
  dirname?: string | null | undefined
  /**
   * Set `data`.
   */
  data?: Data | null | undefined
}
/**
 * Raw source map.
 *
 * See:
 * <https://github.com/mozilla/source-map/blob/58819f0/source-map.d.ts#L15-L23>.
 */
type Map = {
  /**
   *  Which version of the source map spec this map is following.
   */
  version: number
  /**
   *  An array of URLs to the original source files.
   */
  sources: Array<string>
  /**
   *  An array of identifiers which can be referenced by individual mappings.
   */
  names: Array<string>
  /**
   * The URL root from which all sources are relative.
   */
  sourceRoot?: string | undefined
  /**
   * An array of contents of the original source files.
   */
  sourcesContent?: Array<string> | undefined
  /**
   *  A string of base64 VLQs which contain the actual mappings.
   */
  mappings: string
  /**
   *  The generated file this source map is associated with.
   */
  file: string
}
/**
 * Configuration.
 *
 * A bunch of keys that will be shallow copied over to the new file.
 */
type Options$4 = {
  [key: string]: unknown
} & VFileCoreOptions

// TypeScript Version: 4.0


/* eslint-disable @typescript-eslint/naming-convention */

type VFileWithOutput<Result> = Result extends Uint8Array // Buffer.
  ? VFile$1
  : Result extends object // Custom result type
  ? VFile$1 & {result: Result}
  : VFile$1

// Get the right most non-void thing.
type Specific<Left = void, Right = void> = Right extends void ? Left : Right

// Create a processor based on the input/output of a plugin.
type UsePlugin<
  ParseTree extends Node$2 | void = void,
  CurrentTree extends Node$2 | void = void,
  CompileTree extends Node$2 | void = void,
  CompileResult = void,
  Input = void,
  Output = void
> = Output extends Node$2
  ? Input extends string
    ? // If `Input` is `string` and `Output` is `Node`, then this plugin
      // defines a parser, so set `ParseTree`.
      Processor<
        Output,
        Specific<Output, CurrentTree>,
        Specific<Output, CompileTree>,
        CompileResult
      >
    : Input extends Node$2
    ? // If `Input` is `Node` and `Output` is `Node`, then this plugin defines a
      // transformer, its output defines the input of the next, so set
      // `CurrentTree`.
      Processor<
        Specific<Input, ParseTree>,
        Output,
        Specific<CompileTree, Output>,
        CompileResult
      >
    : // Else, `Input` is something else and `Output` is `Node`:
      never
  : Input extends Node$2
  ? // If `Input` is `Node` and `Output` is not a `Node`, then this plugin
    // defines a compiler, so set `CompileTree` and `CompileResult`
    Processor<
      Specific<Input, ParseTree>,
      Specific<Input, CurrentTree>,
      Input,
      Output
    >
  : // Else, `Input` is not a `Node` and `Output` is not a `Node`.
    // Maybe it’s untyped, or the plugin throws an error (`never`), so lets
    // just keep it as it was.
    Processor<ParseTree, CurrentTree, CompileTree, CompileResult>

/* eslint-enable @typescript-eslint/naming-convention */

/**
 * Processor allows plugins to be chained together to transform content.
 * The chain of plugins defines how content flows through it.
 *
 * @typeParam ParseTree
 *   The node that the parser yields (and `run` receives).
 * @typeParam CurrentTree
 *   The node that the last attached plugin yields.
 * @typeParam CompileTree
 *   The node that the compiler receives (and `run` yields).
 * @typeParam CompileResult
 *   The thing that the compiler yields.
 */
interface Processor<
  ParseTree extends Node$2 | void = void,
  CurrentTree extends Node$2 | void = void,
  CompileTree extends Node$2 | void = void,
  CompileResult = void
> extends FrozenProcessor<ParseTree, CurrentTree, CompileTree, CompileResult> {
  /**
   * Configure the processor to use a plugin.
   *
   * @typeParam PluginParameters
   *   Plugin settings.
   * @typeParam Input
   *   Value that is accepted by the plugin.
   *
   *   *   If the plugin returns a transformer, then this should be the node
   *       type that the transformer expects.
   *   *   If the plugin sets a parser, then this should be `string`.
   *   *   If the plugin sets a compiler, then this should be the node type that
   *       the compiler expects.
   * @typeParam Output
   *   Value that the plugin yields.
   *
   *   *   If the plugin returns a transformer, then this should be the node
   *       type that the transformer yields, and defaults to `Input`.
   *   *   If the plugin sets a parser, then this should be the node type that
   *       the parser yields.
   *   *   If the plugin sets a compiler, then this should be the result that
   *       the compiler yields (`string`, `Buffer`, or something else).
   * @param plugin
   *   Plugin (function) to use.
   *   Plugins are deduped based on identity: passing a function in twice will
   *   cause it to run only once.
   * @param settings
   *   Configuration for plugin, optional.
   *   Plugins typically receive one options object, but could receive other and
   *   more values.
   *   It’s also possible to pass a boolean instead of settings: `true` (to turn
   *   a plugin on) or `false` (to turn a plugin off).
   * @returns
   *   Current processor.
   */
  use<
    PluginParameters extends any[] = any[],
    Input = Specific<Node$2, CurrentTree>,
    Output = Input
  >(
    plugin: Plugin<PluginParameters, Input, Output>,
    ...settings: PluginParameters | [boolean]
  ): UsePlugin<
    ParseTree,
    CurrentTree,
    CompileTree,
    CompileResult,
    Input,
    Output
  >

  /**
   * Configure the processor with a tuple of a plugin and setting(s).
   *
   * @typeParam PluginParameters
   *   Plugin settings.
   * @typeParam Input
   *   Value that is accepted by the plugin.
   *
   *   *   If the plugin returns a transformer, then this should be the node
   *       type that the transformer expects.
   *   *   If the plugin sets a parser, then this should be `string`.
   *   *   If the plugin sets a compiler, then this should be the node type that
   *       the compiler expects.
   * @typeParam Output
   *   Value that the plugin yields.
   *
   *   *   If the plugin returns a transformer, then this should be the node
   *       type that the transformer yields, and defaults to `Input`.
   *   *   If the plugin sets a parser, then this should be the node type that
   *       the parser yields.
   *   *   If the plugin sets a compiler, then this should be the result that
   *       the compiler yields (`string`, `Buffer`, or something else).
   * @param tuple
   *   A tuple where the first item is a plugin (function) to use and other
   *   items are options.
   *   Plugins are deduped based on identity: passing a function in twice will
   *   cause it to run only once.
   *   It’s also possible to pass a boolean instead of settings: `true` (to turn
   *   a plugin on) or `false` (to turn a plugin off).
   * @returns
   *   Current processor.
   */
  use<
    PluginParameters extends any[] = any[],
    Input = Specific<Node$2, CurrentTree>,
    Output = Input
  >(
    tuple:
      | PluginTuple<PluginParameters, Input, Output>
      | [Plugin<PluginParameters, Input, Output>, boolean]
  ): UsePlugin<
    ParseTree,
    CurrentTree,
    CompileTree,
    CompileResult,
    Input,
    Output
  >

  /**
   * Configure the processor with a preset or list of plugins and presets.
   *
   * @param presetOrList
   *   Either a list of plugins, presets, and tuples, or a single preset: an
   *   object with a `plugins` (list) and/or `settings`
   *   (`Record<string, unknown>`).
   * @returns
   *   Current processor.
   */
  use(
    presetOrList: Preset | PluggableList
  ): Processor<ParseTree, CurrentTree, CompileTree, CompileResult>
}

/**
 * A frozen processor is just like a regular processor, except no additional
 * plugins can be added.
 * A frozen processor can be created by calling `.freeze()` on a processor.
 * An unfrozen processor can be created by calling a processor.
 */
interface FrozenProcessor<
  ParseTree extends Node$2 | void = void,
  CurrentTree extends Node$2 | void = void,
  CompileTree extends Node$2 | void = void,
  CompileResult = void
> {
  /**
   * Clone current processor
   *
   * @returns
   *   New unfrozen processor that is configured to function the same as its
   *   ancestor.
   *   But when the descendant processor is configured it does not affect the
   *   ancestral processor.
   */
  (): Processor<ParseTree, CurrentTree, CompileTree, CompileResult>

  /**
   * Internal list of configured plugins.
   *
   * @private
   */
  attachers: Array<[Plugin, ...unknown[]]>

  Parser?: Parser<Specific<Node$2, ParseTree>> | undefined
  Compiler?:
    | Compiler<Specific<Node$2, CompileTree>, Specific<unknown, CompileResult>>
    | undefined

  /**
   * Parse a file.
   *
   * @param file
   *   File to parse.
   *   `VFile` or anything that can be given to `new VFile()`, optional.
   * @returns
   *   Resulting tree.
   */
  parse(file?: Compatible | undefined): Specific<Node$2, ParseTree>

  /**
   * Compile a file.
   *
   * @param node
   *   Node to compile.
   * @param file
   *   `VFile` or anything that can be given to `new VFile()`, optional.
   * @returns
   *   New content: compiled text (`string` or `Buffer`) or something else.
   *   This depends on which plugins you use: typically text, but could for
   *   example be a React node.
   */
  stringify(
    node: Specific<Node$2, CompileTree>,
    file?: Compatible | undefined
  ): CompileTree extends Node$2 ? CompileResult : unknown

  /**
   * Run transforms on the given tree.
   *
   * @param node
   *   Tree to transform.
   * @param callback
   *   Callback called with an error or the resulting node.
   * @returns
   *   Nothing.
   */
  run(
    node: Specific<Node$2, ParseTree>,
    callback: RunCallback<Specific<Node$2, CompileTree>>
  ): void

  /**
   * Run transforms on the given node.
   *
   * @param node
   *   Tree to transform.
   * @param file
   *   File associated with `node`.
   *   `VFile` or anything that can be given to `new VFile()`.
   * @param callback
   *   Callback called with an error or the resulting node.
   * @returns
   *   Nothing.
   */
  run(
    node: Specific<Node$2, ParseTree>,
    file: Compatible | undefined,
    callback: RunCallback<Specific<Node$2, CompileTree>>
  ): void

  /**
   * Run transforms on the given node.
   *
   * @param node
   *   Tree to transform.
   * @param file
   *   File associated with `node`.
   *   `VFile` or anything that can be given to `new VFile()`.
   * @returns
   *   Promise that resolves to the resulting tree.
   */
  run(
    node: Specific<Node$2, ParseTree>,
    file?: Compatible | undefined
  ): Promise<Specific<Node$2, CompileTree>>

  /**
   * Run transforms on the given node, synchronously.
   * Throws when asynchronous transforms are configured.
   *
   * @param node
   *   Tree to transform.
   * @param file
   *   File associated with `node`.
   *   `VFile` or anything that can be given to `new VFile()`, optional.
   * @returns
   *   Resulting tree.
   */
  runSync(
    node: Specific<Node$2, ParseTree>,
    file?: Compatible | undefined
  ): Specific<Node$2, CompileTree>

  /**
   * Process a file.
   *
   * This performs all phases of the processor:
   *
   * 1.  Parse a file into a unist node using the configured `Parser`
   * 2.  Run transforms on that node
   * 3.  Compile the resulting node using the `Compiler`
   *
   * The result from the compiler is stored on the file.
   * What the result is depends on which plugins you use.
   * The result is typically text (`string` or `Buffer`), which can be retrieved
   * with `file.toString()` (or `String(file)`).
   * In some cases, such as when using `rehypeReact` to create a React node,
   * the result is stored on `file.result`.
   *
   * @param file
   *   `VFile` or anything that can be given to `new VFile()`.
   * @param callback
   *   Callback called with an error or the resulting file.
   * @returns
   *   Nothing.
   */
  process(
    file: Compatible | undefined,
    callback: ProcessCallback<VFileWithOutput<CompileResult>>
  ): void

  /**
   * Process a file.
   *
   * This performs all phases of the processor:
   *
   * 1.  Parse a file into a unist node using the configured `Parser`
   * 2.  Run transforms on that node
   * 3.  Compile the resulting node using the `Compiler`
   *
   * The result from the compiler is stored on the file.
   * What the result is depends on which plugins you use.
   * The result is typically text (`string` or `Buffer`), which can be retrieved
   * with `file.toString()` (or `String(file)`).
   * In some cases, such as when using `rehypeReact` to create a React node,
   * the result is stored on `file.result`.
   *
   * @param file
   *   `VFile` or anything that can be given to `new VFile()`.
   * @returns
   *   Promise that resolves to the resulting `VFile`.
   */
  process(file: Compatible): Promise<VFileWithOutput<CompileResult>>

  /**
   * Process a file, synchronously.
   * Throws when asynchronous transforms are configured.
   *
   * This performs all phases of the processor:
   *
   * 1.  Parse a file into a unist node using the configured `Parser`
   * 2.  Run transforms on that node
   * 3.  Compile the resulting node using the `Compiler`
   *
   * The result from the compiler is stored on the file.
   * What the result is depends on which plugins you use.
   * The result is typically text (`string` or `Buffer`), which can be retrieved
   * with `file.toString()` (or `String(file)`).
   * In some cases, such as when using `rehypeReact` to create a React node,
   * the result is stored on `file.result`.
   *
   * @param file
   *   `VFile` or anything that can be given to `new VFile()`, optional.
   * @returns
   *   Resulting file.
   */
  processSync(
    file?: Compatible | undefined
  ): VFileWithOutput<CompileResult>

  /**
   * Get an in-memory key-value store accessible to all phases of the process.
   *
   * @returns
   *   Key-value store.
   */
  data(): Record<string, unknown>

  /**
   * Set an in-memory key-value store accessible to all phases of the process.
   *
   * @param data
   *   Key-value store.
   * @returns
   *   Current processor.
   */
  data(
    data: Record<string, unknown>
  ): Processor<ParseTree, CurrentTree, CompileTree, CompileResult>

  /**
   * Get an in-memory value by key.
   *
   * @param key
   *   Key to get.
   * @returns
   *   The value at `key`.
   */
  data(key: string): unknown

  /**
   * Set an in-memory value by key.
   *
   * @param key
   *   Key to set.
   * @param value
   *   Value to set.
   * @returns
   *   Current processor.
   */
  data(
    key: string,
    value: unknown
  ): Processor<ParseTree, CurrentTree, CompileTree, CompileResult>

  /**
   * Freeze a processor.
   * Frozen processors are meant to be extended and not to be configured or
   * processed directly.
   *
   * Once a processor is frozen it cannot be unfrozen.
   * New processors working just like it can be created by calling the
   * processor.
   *
   * It’s possible to freeze processors explicitly, by calling `.freeze()`, but
   * `.parse()`, `.run()`, `.stringify()`, and `.process()` call `.freeze()` to
   * freeze a processor too.
   *
   * @returns
   *   Frozen processor.
   */
  freeze(): FrozenProcessor<ParseTree, CurrentTree, CompileTree, CompileResult>
}

/**
 * A plugin is a function.
 * It configures the processor and in turn can receive options.
 * Plugins can configure processors by interacting with parsers and compilers
 * (at `this.Parser` or `this.Compiler`) or by specifying how the syntax tree
 * is handled (by returning a `Transformer`).
 *
 * @typeParam PluginParameters
 *   Plugin settings.
 * @typeParam Input
 *   Value that is accepted by the plugin.
 *
 *   *   If the plugin returns a transformer, then this should be the node
 *       type that the transformer expects.
 *   *   If the plugin sets a parser, then this should be `string`.
 *   *   If the plugin sets a compiler, then this should be the node type that
 *       the compiler expects.
 * @typeParam Output
 *   Value that the plugin yields.
 *
 *   *   If the plugin returns a transformer, then this should be the node
 *       type that the transformer yields, and defaults to `Input`.
 *   *   If the plugin sets a parser, then this should be the node type that
 *       the parser yields.
 *   *   If the plugin sets a compiler, then this should be the result that
 *       the compiler yields (`string`, `Buffer`, or something else).
 * @this
 *   The current processor.
 *   Plugins can configure the processor by interacting with `this.Parser` or
 *   `this.Compiler`, or by accessing the data associated with the whole process
 *   (`this.data`).
 * @param settings
 *   Configuration for plugin.
 *   Plugins typically receive one options object, but could receive other and
 *   more values.
 *   Users can also pass a boolean instead of settings: `true` (to turn
 *   a plugin on) or `false` (to turn a plugin off).
 *   When a plugin is turned off, it won’t be called.
 *
 *   When creating your own plugins, please accept only a single object!
 *   It allows plugins to be reconfigured and it helps users to know that every
 *   plugin accepts one options object.
 * @returns
 *   Plugins can return a `Transformer` to specify how the syntax tree is
 *   handled.
 */
type Plugin<
  PluginParameters extends any[] = any[],
  Input = Node$2,
  Output = Input
> = (
  this: Input extends Node$2
    ? Output extends Node$2
      ? // This is a transform, so define `Input` as the current tree.
        Processor<void, Input>
      : // Compiler.
        Processor<void, Input, Input, Output>
    : Output extends Node$2
    ? // Parser.
      Processor<Output, Output>
    : // No clue.
      Processor,
  ...settings: PluginParameters
) => // If both `Input` and `Output` are `Node`, expect an optional `Transformer`.
Input extends Node$2
  ? Output extends Node$2
    ? Transformer<Input, Output> | void
    : void
  : void

/**
 * Presets provide a sharable way to configure processors with multiple plugins
 * and/or settings.
 */
interface Preset {
  plugins?: PluggableList
  settings?: Record<string, unknown>
}

/**
 * A tuple of a plugin and its setting(s).
 * The first item is a plugin (function) to use and other items are options.
 * Plugins are deduped based on identity: passing a function in twice will
 * cause it to run only once.
 *
 * @typeParam PluginParameters
 *   Plugin settings.
 * @typeParam Input
 *   Value that is accepted by the plugin.
 *
 *   *   If the plugin returns a transformer, then this should be the node
 *       type that the transformer expects.
 *   *   If the plugin sets a parser, then this should be `string`.
 *   *   If the plugin sets a compiler, then this should be the node type that
 *       the compiler expects.
 * @typeParam Output
 *   Value that the plugin yields.
 *
 *   *   If the plugin returns a transformer, then this should be the node
 *       type that the transformer yields, and defaults to `Input`.
 *   *   If the plugin sets a parser, then this should be the node type that
 *       the parser yields.
 *   *   If the plugin sets a compiler, then this should be the result that
 *       the compiler yields (`string`, `Buffer`, or something else).
 */
type PluginTuple<
  PluginParameters extends any[] = any[],
  Input = Node$2,
  Output = Input
> = [Plugin<PluginParameters, Input, Output>, ...PluginParameters]

/**
 * A union of the different ways to add plugins and settings.
 *
 * @typeParam PluginParameters
 *   Plugin settings.
 */
type Pluggable<PluginParameters extends any[] = any[]> =
  | PluginTuple<PluginParameters, any, any>
  | Plugin<PluginParameters, any, any>
  | Preset

/**
 * A list of plugins and presets.
 */
type PluggableList = Pluggable[]

/**
 * Transformers modify the syntax tree or metadata of a file.
 * A transformer is a function that is called each time a file is passed
 * through the transform phase.
 * If an error occurs (either because it’s thrown, returned, rejected, or passed
 * to `next`), the process stops.
 *
 * @typeParam Input
 *   Node type that the transformer expects.
 * @typeParam Output
 *   Node type that the transformer yields.
 * @param node
 *   Tree to be transformed.
 * @param file
 *   File associated with node.
 * @param next
 *   Callback that you must call when done.
 *   Note: this is given if you accept three parameters in your transformer.
 *   If you accept up to two parameters, it’s not given, and you can return
 *   a promise.
 * @returns
 *   Any of the following:
 *
 *   * `void` — If nothing is returned, the next transformer keeps using same
 *     tree.
 *   * `Error` — Can be returned to stop the process.
 *   * `Node` — Can be returned and results in further transformations and
 *     `stringify`s to be performed on the new tree.
 *   * `Promise` — If a promise is returned, the function is asynchronous, and
 *      must be resolved (optionally with a `Node`) or rejected (optionally with
 *      an `Error`).
 *
 *   If you accept a `next` callback, nothing should be returned.
 */
type Transformer<
  Input extends Node$2 = Node$2,
  Output extends Node$2 = Input
> = (
  node: Input,
  file: VFile$1,
  next: TransformCallback<Output>
) => Promise<Output | undefined | void> | Output | Error | undefined | void

/**
 * Callback you must call when a transformer is done.
 *
 * @typeParam Tree
 *   Node that the plugin yields.
 * @param error
 *   Pass an error to stop the process.
 * @param node
 *   Pass a tree to continue transformations (and `stringify`) on the new tree.
 * @param file
 *   Pass a file to continue transformations (and `stringify`) on the new file.
 * @returns
 *   Nothing.
 */
type TransformCallback<Tree extends Node$2 = Node$2> = (
  error?: Error | null | undefined,
  node?: Tree | undefined,
  file?: VFile$1 | undefined
) => void

/**
 * Function handling the parsing of text to a syntax tree.
 * Used in the parse phase in the process and called with a `string` and
 * `VFile` representation of the document to parse.
 *
 * `Parser` can be a normal function, in which case it must return a `Node`:
 * the syntax tree representation of the given file.
 *
 * `Parser` can also be a constructor function (a function with keys in its
 * `prototype`), in which case it’s called with `new`.
 * Instances must have a parse method that is called without arguments and
 * must return a `Node`.
 *
 * @typeParam Tree
 *   The node that the parser yields (and `run` receives).
 */
type Parser<Tree extends Node$2 = Node$2> =
  | ParserClass<Tree>
  | ParserFunction<Tree>

/**
 * A class to parse files.
 *
 * @typeParam Tree
 *   The node that the parser yields.
 */
declare class ParserClass<Tree extends Node$2 = Node$2> {
  prototype: {
    /**
     * Parse a file.
     *
     * @returns
     *   Parsed tree.
     */
    parse(): Tree
  }

  /**
   * Constructor.
   *
   * @param document
   *   Document to parse.
   * @param file
   *   File associated with `document`.
   * @returns
   *   Instance.
   */
  constructor(document: string, file: VFile$1)
}

/**
 * Normal function to parse a file.
 *
 * @typeParam Tree
 *   The node that the parser yields.
 * @param document
 *   Document to parse.
 * @param file
 *   File associated with `document`.
 * @returns
 *   Node representing the given file.
 */
type ParserFunction<Tree extends Node$2 = Node$2> = (
  document: string,
  file: VFile$1
) => Tree

/**
 * Function handling the compilation of syntax tree to a text.
 * Used in the stringify phase in the process and called with a `Node` and
 * `VFile` representation of the document to stringify.
 *
 * `Compiler` can be a normal function, in which case it must return a
 * `string`: the text representation of the given syntax tree.
 *
 * `Compiler` can also be a constructor function (a function with keys in its
 * `prototype`), in which case it’s called with `new`.
 * Instances must have a `compile` method that is called without arguments
 * and must return a `string`.
 *
 * @typeParam Tree
 *   The node that the compiler receives.
 * @typeParam Result
 *   The thing that the compiler yields.
 */
type Compiler<Tree extends Node$2 = Node$2, Result = unknown> =
  | CompilerClass<Tree, Result>
  | CompilerFunction<Tree, Result>

/**
 * A class to compile trees.
 *
 * @typeParam Tree
 *   The node that the compiler receives.
 * @typeParam Result
 *   The thing that the compiler yields.
 */
declare class CompilerClass<Tree extends Node$2 = Node$2, Result = unknown> {
  prototype: {
    /**
     * Compile a tree.
     *
     * @returns
     *   New content: compiled text (`string` or `Buffer`, for `file.value`) or
     *   something else (for `file.result`).
     */
    compile(): Result
  }

  /**
   * Constructor.
   *
   * @param tree
   *   Tree to compile.
   * @param file
   *   File associated with `tree`.
   * @returns
   *   Instance.
   */
  constructor(tree: Tree, file: VFile$1)
}

/**
 * Normal function to compile a tree.
 *
 * @typeParam Tree
 *   The node that the compiler receives.
 * @typeParam Result
 *   The thing that the compiler yields.
 * @param tree
 *   Tree to compile.
 * @param file
 *   File associated with `tree`.
 * @returns
 *   New content: compiled text (`string` or `Buffer`, for `file.value`) or
 *   something else (for `file.result`).
 */
type CompilerFunction<Tree extends Node$2 = Node$2, Result = unknown> = (
  tree: Tree,
  file: VFile$1
) => Result

/**
 * Callback called when a done running.
 *
 * @typeParam Tree
 *   The tree that the callback receives.
 * @param error
 *   Error passed when unsuccessful.
 * @param node
 *   Tree to transform.
 * @param file
 *   File passed when successful.
 * @returns
 *   Nothing.
 */
type RunCallback<Tree extends Node$2 = Node$2> = (
  error?: Error | null | undefined,
  node?: Tree | undefined,
  file?: VFile$1 | undefined
) => void

/**
 * Callback called when a done processing.
 *
 * @typeParam File
 *   The file that the callback receives.
 * @param error
 *   Error passed when unsuccessful.
 * @param file
 *   File passed when successful.
 * @returns
 *   Nothing.
 */
type ProcessCallback<File extends VFile$1 = VFile$1> = (
  error?: Error | null | undefined,
  file?: File | undefined
) => void

/**
 * Configuration for internal plugin `recma-jsx-rewrite`.
 */
type RecmaJsxRewriteOptions$1 = {
    /**
     * Whether to use an import statement or `arguments[0]` to get the provider.
     */
    outputFormat?: 'function-body' | 'program' | null | undefined;
    /**
     * Place to import a provider from.
     */
    providerImportSource?: string | null | undefined;
    /**
     * Whether to add extra info to error messages in generated code.
     *
     * This also results in the development automatic JSX runtime
     * (`/jsx-dev-runtime`, `jsxDEV`) being used, which passes positional info to
     * nodes.
     * The default can be set to `true` in Node.js through environment variables:
     * set `NODE_ENV=development`.
     */
    development?: boolean | null | undefined;
};

interface StartOfSourceMap {
    file?: string;
    sourceRoot?: string;
}

interface RawSourceMap extends StartOfSourceMap {
    version: string;
    sources: string[];
    names: string[];
    sourcesContent?: string[];
    mappings: string;
}

interface Position {
    line: number;
    column: number;
}

interface LineRange extends Position {
    lastColumn: number;
}

interface FindPosition extends Position {
    // SourceMapConsumer.GREATEST_LOWER_BOUND or SourceMapConsumer.LEAST_UPPER_BOUND
    bias?: number;
}

interface SourceFindPosition extends FindPosition {
    source: string;
}

interface MappedPosition extends Position {
    source: string;
    name?: string;
}

interface MappingItem {
    source: string;
    generatedLine: number;
    generatedColumn: number;
    originalLine: number;
    originalColumn: number;
    name: string;
}

declare class SourceMapConsumer {
    static GENERATED_ORDER: number;
    static ORIGINAL_ORDER: number;

    static GREATEST_LOWER_BOUND: number;
    static LEAST_UPPER_BOUND: number;

    constructor(rawSourceMap: RawSourceMap);
    computeColumnSpans(): void;
    originalPositionFor(generatedPosition: FindPosition): MappedPosition;
    generatedPositionFor(originalPosition: SourceFindPosition): LineRange;
    allGeneratedPositionsFor(originalPosition: MappedPosition): Position[];
    hasContentsOfAllSources(): boolean;
    sourceContentFor(source: string, returnNullOnMissing?: boolean): string;
    eachMapping(callback: (mapping: MappingItem) => void, context?: any, order?: number): void;
}

interface Mapping {
    generated: Position;
    original: Position;
    source: string;
    name?: string;
}

declare class SourceMapGenerator$1 {
    constructor(startOfSourceMap?: StartOfSourceMap);
    static fromSourceMap(sourceMapConsumer: SourceMapConsumer): SourceMapGenerator$1;
    addMapping(mapping: Mapping): void;
    setSourceContent(sourceFile: string, sourceContent: string): void;
    applySourceMap(sourceMapConsumer: SourceMapConsumer, sourceFile?: string, sourceMapPath?: string): void;
    toString(): string;
}

type SourceMapGenerator = typeof SourceMapGenerator$1;
/**
 * Configuration for internal plugin `recma-stringify`.
 */
type RecmaStringifyOptions$1 = {
    /**
     * Generate a source map by passing a `SourceMapGenerator` from `source-map`
     * in.
     */
    SourceMapGenerator?: SourceMapGenerator | null | undefined;
};

/**
 * Configuration for internal plugin `recma-document`.
 */
type RecmaDocumentOptions$1 = {
    /**
     * Whether to use either `import` and `export` statements to get the runtime
     * (and optionally provider) and export the content, or get values from
     * `arguments` and return things.
     */
    outputFormat?: 'function-body' | 'program' | null | undefined;
    /**
     * Whether to keep `import` (and `export … from`) statements or compile them
     * to dynamic `import()` instead.
     */
    useDynamicImport?: boolean | null | undefined;
    /**
     * Resolve `import`s (and `export … from`, and `import.meta.url`) relative to
     * this URL.
     */
    baseUrl?: string | null | undefined;
    /**
     * Pragma for JSX (used in classic runtime).
     */
    pragma?: string | null | undefined;
    /**
     * Pragma for JSX fragments (used in classic runtime).
     */
    pragmaFrag?: string | null | undefined;
    /**
     * Where to import the identifier of `pragma` from (used in classic runtime).
     */
    pragmaImportSource?: string | null | undefined;
    /**
     * Place to import automatic JSX runtimes from (used in automatic runtime).
     */
    jsxImportSource?: string | null | undefined;
    /**
     * JSX runtime to use.
     */
    jsxRuntime?: 'automatic' | 'classic' | null | undefined;
};

// Type definitions for non-npm package Hast 2.3


/**
 * This map registers all node types that may be used as top-level content in the document.
 *
 * These types are accepted inside `root` nodes.
 *
 * This interface can be augmented to register custom node types.
 *
 * @example
 * declare module 'hast' {
 *   interface RootContentMap {
 *     // Allow using raw nodes defined by `rehype-raw`.
 *     raw: Raw;
 *   }
 * }
 */
interface RootContentMap {
    comment: Comment;
    doctype: DocType;
    element: Element;
    text: Text$1;
}

/**
 * This map registers all node types that may be used as content in an element.
 *
 * These types are accepted inside `element` nodes.
 *
 * This interface can be augmented to register custom node types.
 *
 * @example
 * declare module 'hast' {
 *   interface RootContentMap {
 *     custom: Custom;
 *   }
 * }
 */
interface ElementContentMap {
    comment: Comment;
    element: Element;
    text: Text$1;
}

type Content$1 = RootContent | ElementContent;

type RootContent = RootContentMap[keyof RootContentMap];

type ElementContent = ElementContentMap[keyof ElementContentMap];

/**
 * Node in hast containing other nodes.
 */
interface Parent$1 extends Parent$2 {
    /**
     * List representing the children of a node.
     */
    children: Content$1[];
}

/**
 * Nodes in hast containing a value.
 */
interface Literal$1 extends Literal$2 {
    value: string;
}

/**
 * Root represents a document.
 * Can be used as the rood of a tree, or as a value of the
 * content field on a 'template' Element, never as a child.
 */
interface Root$1 extends Parent$1 {
    /**
     * Represents this variant of a Node.
     */
    type: 'root';

    /**
     * List representing the children of a node.
     */
    children: RootContent[];
}

/**
 * Element represents an HTML Element.
 */
interface Element extends Parent$1 {
    /**
     * Represents this variant of a Node.
     */
    type: 'element';

    /**
     * Represents the element’s local name.
     */
    tagName: string;

    /**
     * Represents information associated with the element.
     */
    properties?: Properties | undefined;

    /**
     * If the tagName field is 'template', a content field can be present.
     */
    content?: Root$1 | undefined;

    /**
     * List representing the children of a node.
     */
    children: ElementContent[];
}

/**
 * Represents information associated with an element.
 */
interface Properties {
    [PropertyName: string]: boolean | number | string | null | undefined | Array<string | number>;
}

/**
 * Represents an HTML DocumentType.
 */
interface DocType extends Node$2 {
    /**
     * Represents this variant of a Node.
     */
    type: 'doctype';

    name: string;
}

/**
 * Represents an HTML Comment.
 */
interface Comment extends Literal$1 {
    /**
     * Represents this variant of a Literal.
     */
    type: 'comment';
}

/**
 * Represents an HTML Text.
 */
interface Text$1 extends Literal$1 {
    /**
     * Represents this variant of a Literal.
     */
    type: 'text';
}

/**
 * Specify casing to use for attribute names.
 *
 * HTML casing is for example `class`, `stroke-linecap`, `xml:lang`.
 * React casing is for example `className`, `strokeLinecap`, `xmlLang`.
 */
type ElementAttributeNameCase = 'html' | 'react';
/**
 * Casing to use for property names in `style` objects.
 *
 * CSS casing is for example `background-color` and `-webkit-line-clamp`.
 * DOM casing is for example `backgroundColor` and `WebkitLineClamp`.
 */
type StylePropertyNameCase = 'css' | 'dom';
/**
 * Configuration for internal plugin `rehype-recma`.
 */
type Options$3 = {
    /**
     * Specify casing to use for attribute names.
     *
     * This casing is used for hast elements, not for embedded MDX JSX nodes
     * (components that someone authored manually).
     */
    elementAttributeNameCase?: ElementAttributeNameCase | null | undefined;
    /**
     * Specify casing to use for property names in `style` objects.
     *
     * This casing is used for hast elements, not for embedded MDX JSX nodes
     * (components that someone authored manually).
     */
    stylePropertyNameCase?: StylePropertyNameCase | null | undefined;
};

// Type definitions for Mdast 3.0


type AlignType = 'left' | 'right' | 'center' | null;

type ReferenceType = 'shortcut' | 'collapsed' | 'full';

/**
 * This map registers all node types that may be used where markdown block content is accepted.
 *
 * These types are accepted inside block quotes, list items, footnotes, and roots.
 *
 * This interface can be augmented to register custom node types.
 *
 * @example
 * declare module 'mdast' {
 *   interface BlockContentMap {
 *     // Allow using math nodes defined by `remark-math`.
 *     math: Math;
 *   }
 * }
 */
interface BlockContentMap {
    paragraph: Paragraph;
    heading: Heading;
    thematicBreak: ThematicBreak;
    blockquote: Blockquote;
    list: List;
    table: Table;
    html: HTML;
    code: Code;
}

/**
 * This map registers all frontmatter node types.
 *
 * This interface can be augmented to register custom node types.
 *
 * @example
 * declare module 'mdast' {
 *   interface FrontmatterContentMap {
 *     // Allow using toml nodes defined by `remark-frontmatter`.
 *     toml: TOML;
 *   }
 * }
 */
interface FrontmatterContentMap {
    yaml: YAML;
}

/**
 * This map registers all node definition types.
 *
 * This interface can be augmented to register custom node types.
 *
 * @example
 * declare module 'mdast' {
 *   interface DefinitionContentMap {
 *     custom: Custom;
 *   }
 * }
 */
interface DefinitionContentMap {
    definition: Definition;
    footnoteDefinition: FootnoteDefinition;
}

/**
 * This map registers all node types that are acceptable in a static phrasing context.
 *
 * This interface can be augmented to register custom node types in a phrasing context, including links and link
 * references.
 *
 * @example
 * declare module 'mdast' {
 *   interface StaticPhrasingContentMap {
 *     mdxJsxTextElement: MDXJSXTextElement;
 *   }
 * }
 */
interface StaticPhrasingContentMap {
    text: Text;
    emphasis: Emphasis;
    strong: Strong;
    delete: Delete;
    html: HTML;
    inlineCode: InlineCode;
    break: Break;
    image: Image;
    imageReference: ImageReference;
    footnote: Footnote;
    footnoteReference: FootnoteReference;
}

/**
 * This map registers all node types that are acceptable in a (interactive) phrasing context (so not in links).
 *
 * This interface can be augmented to register custom node types in a phrasing context, excluding links and link
 * references.
 *
 * @example
 * declare module 'mdast' {
 *   interface PhrasingContentMap {
 *     custom: Custom;
 *   }
 * }
 */
interface PhrasingContentMap extends StaticPhrasingContentMap {
    link: Link;
    linkReference: LinkReference;
}

/**
 * This map registers all node types that are acceptable inside lists.
 *
 * This interface can be augmented to register custom node types that are acceptable inside lists.
 *
 * @example
 * declare module 'mdast' {
 *   interface ListContentMap {
 *     custom: Custom;
 *   }
 * }
 */
interface ListContentMap {
    listItem: ListItem;
}

/**
 * This map registers all node types that are acceptable inside tables (not table cells).
 *
 * This interface can be augmented to register custom node types that are acceptable inside tables.
 *
 * @example
 * declare module 'mdast' {
 *   interface TableContentMap {
 *     custom: Custom;
 *   }
 * }
 */
interface TableContentMap {
    tableRow: TableRow;
}

/**
 * This map registers all node types that are acceptable inside tables rows (not table cells).
 *
 * This interface can be augmented to register custom node types that are acceptable inside table rows.
 *
 * @example
 * declare module 'mdast' {
 *   interface RowContentMap {
 *     custom: Custom;
 *   }
 * }
 */
interface RowContentMap {
    tableCell: TableCell;
}

type Content = TopLevelContent | ListContent | TableContent | RowContent | PhrasingContent;

type TopLevelContent = BlockContent | FrontmatterContent | DefinitionContent;

type BlockContent = BlockContentMap[keyof BlockContentMap];

type FrontmatterContent = FrontmatterContentMap[keyof FrontmatterContentMap];

type DefinitionContent = DefinitionContentMap[keyof DefinitionContentMap];

type ListContent = ListContentMap[keyof ListContentMap];

type TableContent = TableContentMap[keyof TableContentMap];

type RowContent = RowContentMap[keyof RowContentMap];

type PhrasingContent = PhrasingContentMap[keyof PhrasingContentMap];

type StaticPhrasingContent = StaticPhrasingContentMap[keyof StaticPhrasingContentMap];

interface Parent extends Parent$2 {
    children: Content[];
}

interface Literal extends Literal$2 {
    value: string;
}

interface Root extends Parent {
    type: 'root';
}

interface Paragraph extends Parent {
    type: 'paragraph';
    children: PhrasingContent[];
}

interface Heading extends Parent {
    type: 'heading';
    depth: 1 | 2 | 3 | 4 | 5 | 6;
    children: PhrasingContent[];
}

interface ThematicBreak extends Node$2 {
    type: 'thematicBreak';
}

interface Blockquote extends Parent {
    type: 'blockquote';
    children: Array<BlockContent | DefinitionContent>;
}

interface List extends Parent {
    type: 'list';
    ordered?: boolean | null | undefined;
    start?: number | null | undefined;
    spread?: boolean | null | undefined;
    children: ListContent[];
}

interface ListItem extends Parent {
    type: 'listItem';
    checked?: boolean | null | undefined;
    spread?: boolean | null | undefined;
    children: Array<BlockContent | DefinitionContent>;
}

interface Table extends Parent {
    type: 'table';
    align?: AlignType[] | null | undefined;
    children: TableContent[];
}

interface TableRow extends Parent {
    type: 'tableRow';
    children: RowContent[];
}

interface TableCell extends Parent {
    type: 'tableCell';
    children: PhrasingContent[];
}

interface HTML extends Literal {
    type: 'html';
}

interface Code extends Literal {
    type: 'code';
    lang?: string | null | undefined;
    meta?: string | null | undefined;
}

interface YAML extends Literal {
    type: 'yaml';
}

interface Definition extends Node$2, Association, Resource {
    type: 'definition';
}

interface FootnoteDefinition extends Parent, Association {
    type: 'footnoteDefinition';
    children: Array<BlockContent | DefinitionContent>;
}

interface Text extends Literal {
    type: 'text';
}

interface Emphasis extends Parent {
    type: 'emphasis';
    children: PhrasingContent[];
}

interface Strong extends Parent {
    type: 'strong';
    children: PhrasingContent[];
}

interface Delete extends Parent {
    type: 'delete';
    children: PhrasingContent[];
}

interface InlineCode extends Literal {
    type: 'inlineCode';
}

interface Break extends Node$2 {
    type: 'break';
}

interface Link extends Parent, Resource {
    type: 'link';
    children: StaticPhrasingContent[];
}

interface Image extends Node$2, Resource, Alternative {
    type: 'image';
}

interface LinkReference extends Parent, Reference {
    type: 'linkReference';
    children: StaticPhrasingContent[];
}

interface ImageReference extends Node$2, Reference, Alternative {
    type: 'imageReference';
}

interface Footnote extends Parent {
    type: 'footnote';
    children: PhrasingContent[];
}

interface FootnoteReference extends Node$2, Association {
    type: 'footnoteReference';
}

// Mixin
interface Resource {
    url: string;
    title?: string | null | undefined;
}

interface Association {
    identifier: string;
    label?: string | null | undefined;
}

interface Reference extends Association {
    referenceType: ReferenceType;
}

interface Alternative {
    alt?: string | null | undefined;
}

type HastContent = Content$1
type HastElement = Element
type HastElementContent = ElementContent
type HastProperties = Properties
type HastRoot = Root$1
type MdastContent = Content
type MdastDefinition = Definition
type MdastFootnoteDefinition = FootnoteDefinition
type MdastParent = Parent
type MdastRoot = Root
type HastNodes = HastRoot | HastContent
type MdastNodes = MdastRoot | MdastContent
type MdastParents = Extract<MdastNodes, MdastParent>
/**
 * hast fields.
 */
type EmbeddedHastFields = {
  /**
   * Generate a specific element with this tag name instead.
   */
  hName?: string | null | undefined
  /**
   * Generate an element with these properties instead.
   */
  hProperties?: HastProperties | null | undefined
  /**
   * Generate an element with this content instead.
   */
  hChildren?: Array<HastElementContent> | null | undefined
}
/**
 * mdast data with embedded hast fields.
 */
type MdastData = Record<string, unknown> & EmbeddedHastFields
/**
 * mdast node with embedded hast data.
 */
type MdastNodeWithData = MdastNodes & {
  data?: MdastData | null | undefined
}
/**
 * Point-like value.
 */
type PointLike = {
  /**
   * Line.
   */
  line?: number | null | undefined
  /**
   * Column.
   */
  column?: number | null | undefined
  /**
   * Offset.
   */
  offset?: number | null | undefined
}
/**
 * Position-like value.
 */
type PositionLike = {
  /**
   * Point-like value.
   */
  start?: PointLike | null | undefined
  /**
   * Point-like value.
   */
  end?: PointLike | null | undefined
}
/**
 * Handle a node.
 */
type Handler = (
  state: State,
  node: any,
  parent: MdastParents | null | undefined
) => HastElementContent | Array<HastElementContent> | null | undefined
/**
 * Signature of `state` for when props are passed.
 */
type HFunctionProps = (
  node: MdastNodes | PositionLike | null | undefined,
  tagName: string,
  props: HastProperties,
  children?: Array<HastElementContent> | null | undefined
) => HastElement
/**
 * Signature of `state` for when no props are passed.
 */
type HFunctionNoProps = (
  node: MdastNodes | PositionLike | null | undefined,
  tagName: string,
  children?: Array<HastElementContent> | null | undefined
) => HastElement
/**
 * Info on `state`.
 */
type HFields = {
  /**
   * Whether HTML is allowed.
   */
  dangerous: boolean
  /**
   * Prefix to use to prevent DOM clobbering.
   */
  clobberPrefix: string
  /**
   * Label to use to introduce the footnote section.
   */
  footnoteLabel: string
  /**
   * HTML used for the footnote label.
   */
  footnoteLabelTagName: string
  /**
   * Properties on the HTML tag used for the footnote label.
   */
  footnoteLabelProperties: HastProperties
  /**
   * Label to use from backreferences back to their footnote call.
   */
  footnoteBackLabel: string
  /**
   * Definition cache.
   */
  definition: (identifier: string) => MdastDefinition | null
  /**
   * Footnote definitions by their identifier.
   */
  footnoteById: Record<string, MdastFootnoteDefinition>
  /**
   * Identifiers of order when footnote calls first appear in tree order.
   */
  footnoteOrder: Array<string>
  /**
   * Counts for how often the same footnote was called.
   */
  footnoteCounts: Record<string, number>
  /**
   * Applied handlers.
   */
  handlers: Handlers
  /**
   * Handler for any none not in `passThrough` or otherwise handled.
   */
  unknownHandler: Handler
  /**
   * Copy a node’s positional info.
   */
  patch: (from: MdastNodes, node: HastNodes) => void
  /**
   * Honor the `data` of `from`, and generate an element instead of `node`.
   */
  applyData: <Type extends HastNodes>(
    from: MdastNodes,
    to: Type
  ) => Element | Type
  /**
   * Transform an mdast node to hast.
   */
  one: (
    node: MdastNodes,
    parent: MdastParents | null | undefined
  ) => HastElementContent | Array<HastElementContent> | null | undefined
  /**
   * Transform the children of an mdast parent to hast.
   */
  all: (node: MdastNodes) => Array<HastElementContent>
  /**
   * Wrap `nodes` with line endings between each node, adds initial/final line endings when `loose`.
   */
  wrap: <Type_1 extends Content$1>(
    nodes: Type_1[],
    loose?: boolean | null | undefined
  ) => (Type_1 | Text$1)[]
  /**
   * Like `state` but lower-level and usable on non-elements.
   * Deprecated: use `patch` and `applyData`.
   */
  augment: (
    left: MdastNodeWithData | PositionLike | null | undefined,
    right: HastElementContent
  ) => HastElementContent
  /**
   * List of node types to pass through untouched (except for their children).
   */
  passThrough: Array<string>
}
/**
 * Configuration (optional).
 */
type Options$2 = {
  /**
   * Whether to persist raw HTML in markdown in the hast tree.
   */
  allowDangerousHtml?: boolean | null | undefined
  /**
   * Prefix to use before the `id` attribute on footnotes to prevent it from
   * *clobbering*.
   */
  clobberPrefix?: string | null | undefined
  /**
   * Label to use from backreferences back to their footnote call (affects
   * screen readers).
   */
  footnoteBackLabel?: string | null | undefined
  /**
   * Label to use for the footnotes section (affects screen readers).
   */
  footnoteLabel?: string | null | undefined
  /**
   * Properties to use on the footnote label (note that `id: 'footnote-label'`
   * is always added as footnote calls use it with `aria-describedby` to
   * provide an accessible label).
   */
  footnoteLabelProperties?: HastProperties | null | undefined
  /**
   * Tag name to use for the footnote label.
   */
  footnoteLabelTagName?: string | null | undefined
  /**
   * Extra handlers for nodes.
   */
  handlers?: Handlers | null | undefined
  /**
   * List of custom mdast node types to pass through (keep) in hast (note that
   * the node itself is passed, but eventual children are transformed).
   */
  passThrough?: Array<string> | null | undefined
  /**
   * Handler for all unknown nodes.
   */
  unknownHandler?: Handler | null | undefined
}
/**
 * Handle nodes.
 */
type Handlers = Record<string, Handler>
/**
 * Info passed around.
 */
type State = HFunctionProps & HFunctionNoProps & HFields

// Expose node type.
/**
 * Raw string of HTML embedded into HTML AST.
 */
// eslint-disable-next-line @typescript-eslint/consistent-type-definitions
interface Raw$1 extends Literal$1 {
  /**
   * Node type.
   */
  type: 'raw'
}

// Register nodes in content.
declare module 'hast' {
  // eslint-disable-next-line @typescript-eslint/consistent-type-definitions
  interface RootContentMap {
    /**
     * Raw string of HTML embedded into HTML AST.
     */
    raw: Raw$1
  }

  // eslint-disable-next-line @typescript-eslint/consistent-type-definitions
  interface ElementContentMap {
    /**
     * Raw string of HTML embedded into HTML AST.
     */
    raw: Raw$1
  }
}

// Expose node type.
/**
 * Raw string of HTML embedded into HTML AST.
 */
// eslint-disable-next-line @typescript-eslint/consistent-type-definitions
interface Raw extends Literal$1 {
  /**
   * Node type.
   */
  type: 'raw'
}

// Register nodes in content.
declare module 'hast' {
  // eslint-disable-next-line @typescript-eslint/consistent-type-definitions
  interface RootContentMap {
    /**
     * Raw string of HTML embedded into HTML AST.
     */
    raw: Raw
  }

  // eslint-disable-next-line @typescript-eslint/consistent-type-definitions
  interface ElementContentMap {
    /**
     * Raw string of HTML embedded into HTML AST.
     */
    raw: Raw
  }
}

type Options$1 = Options$2

type Options = Options$1

type RemarkRehypeOptions = Options;
type RehypeRecmaOptions = Options$3;
type RecmaDocumentOptions = RecmaDocumentOptions$1;
type RecmaStringifyOptions = RecmaStringifyOptions$1;
type RecmaJsxRewriteOptions = RecmaJsxRewriteOptions$1;
/**
 * Base configuration.
 */
type BaseProcessorOptions$1 = {
    /**
     * Whether to keep JSX.
     */
    jsx?: boolean | null | undefined;
    /**
     * Format of the files to be processed.
     */
    format?: 'mdx' | 'md' | null | undefined;
    /**
     * Whether to compile to a whole program or a function body..
     */
    outputFormat?: "function-body" | "program" | undefined;
    /**
     * Extensions (with `.`) for markdown.
     */
    mdExtensions?: Array<string> | null | undefined;
    /**
     * Extensions (with `.`) for MDX.
     */
    mdxExtensions?: Array<string> | null | undefined;
    /**
     * List of recma (esast, JavaScript) plugins.
     */
    recmaPlugins?: PluggableList | null | undefined;
    /**
     * List of remark (mdast, markdown) plugins.
     */
    remarkPlugins?: PluggableList | null | undefined;
    /**
     * List of rehype (hast, HTML) plugins.
     */
    rehypePlugins?: PluggableList | null | undefined;
    /**
     * Options to pass through to `remark-rehype`.
     */
    remarkRehypeOptions?: RemarkRehypeOptions | null | undefined;
};
/**
 * Configuration for internal plugins.
 */
type PluginOptions$1 = Omit<RehypeRecmaOptions & RecmaDocumentOptions & RecmaStringifyOptions & RecmaJsxRewriteOptions, 'outputFormat'>;

/**
 * Compile MDX to JS.
 *
 * @param {VFileCompatible} vfileCompatible
 *   MDX document to parse (`string`, `Buffer`, `vfile`, anything that can be
 *   given to `vfile`).
 * @param {CompileOptions | null | undefined} [compileOptions]
 *   Compile configuration.
 * @return {Promise<VFile>}
 *   File.
 */
declare function compile$1(vfileCompatible: VFileCompatible, compileOptions?: CompileOptions$1 | null | undefined): Promise<VFile>;
type VFile = VFile$1;
type VFileCompatible = Compatible;
type PluginOptions = PluginOptions$1;
type BaseProcessorOptions = BaseProcessorOptions$1;
/**
 * Core configuration.
 */
type CoreProcessorOptions = Omit<BaseProcessorOptions, 'format'>;
/**
 * Extra configuration.
 */
type ExtraOptions = {
    /**
     * Format of `file`.
     */
    format?: 'detect' | 'mdx' | 'md' | null | undefined;
};
/**
 * Configuration.
 */
type CompileOptions$1 = CoreProcessorOptions & PluginOptions & ExtraOptions;

interface JSXOptions {
    pragma?: string;
    pragmaFrag?: string;
    throwIfNamespace?: false;
    runtime?: 'classic' | 'automatic';
    importSource?: string;
}
type MdxCompileOptions = Parameters<typeof compile$1>[1];
interface CompileOptions {
    skipCsf?: boolean;
    mdxCompileOptions?: MdxCompileOptions;
    jsxOptions?: JSXOptions;
}

declare const SEPARATOR = "/* ========= */";

declare const genBabel: (store: any, root: any) => any;
declare const plugin: (store: any) => (root: any) => any;
declare const postprocess: (code: string, extractedExports: string) => string;
declare const compile: (input: string, { skipCsf, mdxCompileOptions, jsxOptions }?: CompileOptions) => Promise<string>;
declare const compileSync: (input: string, { skipCsf, mdxCompileOptions, jsxOptions }?: CompileOptions) => string;

export { CompileOptions, JSXOptions, MdxCompileOptions, SEPARATOR, compile, compileSync, genBabel, plugin, postprocess, wrapperJs };
