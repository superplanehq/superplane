"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getAllNamedExports = exports.isValidStoryExport = exports.getDescriptor = exports.getMetaObjectExpression = exports.docsUrl = void 0;
/* eslint-disable no-fallthrough */
const csf_1 = require("@storybook/csf");
const utils_1 = require("@typescript-eslint/utils");
const ast_1 = require("./ast");
const docsUrl = (ruleName) => `https://github.com/storybookjs/eslint-plugin-storybook/blob/main/docs/rules/${ruleName}.md`;
exports.docsUrl = docsUrl;
const getMetaObjectExpression = (node, context) => {
    let meta = node.declaration;
    if ((0, ast_1.isIdentifier)(meta)) {
        const variable = utils_1.ASTUtils.findVariable(context.getScope(), meta.name);
        const decl = variable && variable.defs.find((def) => (0, ast_1.isVariableDeclarator)(def.node));
        if (decl && (0, ast_1.isVariableDeclarator)(decl.node)) {
            meta = decl.node.init;
        }
    }
    if ((0, ast_1.isTSAsExpression)(meta) || (0, ast_1.isTSSatisfiesExpression)(meta)) {
        meta = meta.expression;
    }
    return (0, ast_1.isObjectExpression)(meta) ? meta : null;
};
exports.getMetaObjectExpression = getMetaObjectExpression;
const getDescriptor = (metaDeclaration, propertyName) => {
    const property = metaDeclaration &&
        metaDeclaration.properties.find((p) => 'key' in p && 'name' in p.key && p.key.name === propertyName);
    if (!property || (0, ast_1.isSpreadElement)(property)) {
        return undefined;
    }
    const { type } = property.value;
    switch (type) {
        case 'ArrayExpression':
            return property.value.elements.map((t) => {
                if (!['StringLiteral', 'Literal'].includes(t.type)) {
                    throw new Error(`Unexpected descriptor element: ${t.type}`);
                }
                // @ts-expect-error TODO: t should be only StringLiteral or Literal, and the type is not resolving correctly
                return t.value;
            });
        case 'Literal':
        // @ts-expect-error TODO: Investigation needed. Type systems says, that "RegExpLiteral" does not exist
        case 'RegExpLiteral':
            // @ts-expect-error TODO: investigation needed
            return property.value.value;
        default:
            throw new Error(`Unexpected descriptor: ${type}`);
    }
};
exports.getDescriptor = getDescriptor;
const isValidStoryExport = (node, nonStoryExportsConfig) => (0, csf_1.isExportStory)(node.name, nonStoryExportsConfig) && node.name !== '__namedExportsOrder';
exports.isValidStoryExport = isValidStoryExport;
const getAllNamedExports = (node) => {
    // e.g. `export { MyStory }`
    if (!node.declaration && node.specifiers) {
        return node.specifiers.reduce((acc, specifier) => {
            if ((0, ast_1.isIdentifier)(specifier.exported)) {
                acc.push(specifier.exported);
            }
            return acc;
        }, []);
    }
    const decl = node.declaration;
    if ((0, ast_1.isVariableDeclaration)(decl)) {
        const declaration = decl.declarations[0];
        if (declaration) {
            const { id } = declaration;
            // e.g. `export const MyStory`
            if ((0, ast_1.isIdentifier)(id)) {
                return [id];
            }
        }
    }
    if ((0, ast_1.isFunctionDeclaration)(decl)) {
        // e.g. `export function MyStory() {}`
        if ((0, ast_1.isIdentifier)(decl.id)) {
            return [decl.id];
        }
    }
    return [];
};
exports.getAllNamedExports = getAllNamedExports;
