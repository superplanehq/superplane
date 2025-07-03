"use strict";
/**
 * @fileoverview This rule identifies storybook addons that are invalid because they are either not installed or contain a typo in their name.
 * @author Andre "andrelas1" Santos
 */
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
const fs_1 = require("fs");
const ts_dedent_1 = __importDefault(require("ts-dedent"));
const path_1 = require("path");
const create_storybook_rule_1 = require("../utils/create-storybook-rule");
const constants_1 = require("../utils/constants");
const ast_1 = require("../utils/ast");
const utils_1 = require("../utils");
module.exports = (0, create_storybook_rule_1.createStorybookRule)({
    name: 'no-uninstalled-addons',
    defaultOptions: [
        {
            packageJsonLocation: '',
            ignore: [],
        },
    ],
    meta: {
        type: 'problem',
        docs: {
            description: 'This rule identifies storybook addons that are invalid because they are either not installed or contain a typo in their name.',
            categories: [constants_1.CategoryId.RECOMMENDED],
            recommended: 'error', // or 'error'
        },
        messages: {
            addonIsNotInstalled: `The {{ addonName }} is not installed in {{packageJsonPath}}. Did you forget to install it or is your package.json in a different location?`,
        },
        schema: [
            {
                type: 'object',
                properties: {
                    packageJsonLocation: {
                        type: 'string',
                    },
                    ignore: {
                        type: 'array',
                        items: {
                            type: 'string',
                        },
                    },
                },
            },
        ],
    },
    create(context) {
        // variables should be defined here
        const { packageJsonLocation, ignore } = context.options.reduce((acc, val) => {
            return {
                packageJsonLocation: val['packageJsonLocation'] || acc.packageJsonLocation,
                ignore: val['ignore'] || acc.ignore,
            };
        }, { packageJsonLocation: '', ignore: [] });
        //----------------------------------------------------------------------
        // Helpers
        //----------------------------------------------------------------------
        // this will not only exclude the nullables but it will also exclude the type undefined from them, so that TS does not complain
        function excludeNullable(item) {
            return !!item;
        }
        const mergeDepsWithDevDeps = (packageJson) => {
            const deps = Object.keys(packageJson.dependencies || {});
            const devDeps = Object.keys(packageJson.devDependencies || {});
            return [...deps, ...devDeps];
        };
        const isAddonInstalled = (addon, installedAddons) => {
            // cleanup /register or /preset + file extension from registered addon
            const addonName = addon
                .replace(/\.[mc]?js$/, '')
                .replace(/\/register$/, '')
                .replace(/\/preset$/, '');
            return installedAddons.includes(addonName);
        };
        const filterLocalAddons = (addon) => {
            const isLocalAddon = (addon) => addon.startsWith('.') ||
                addon.startsWith('/') ||
                // for local Windows files e.g. (C: F: D:)
                /\w:.*/.test(addon) ||
                addon.startsWith('\\');
            return !isLocalAddon(addon);
        };
        const areThereAddonsNotInstalled = (addons, installedSbAddons) => {
            const result = addons
                // remove local addons (e.g. ./my-addon/register.js)
                .filter(filterLocalAddons)
                .filter((addon) => !isAddonInstalled(addon, installedSbAddons) && !ignore.includes(addon))
                .map((addon) => ({ name: addon }));
            return result.length ? result : false;
        };
        const getPackageJson = (path) => {
            const packageJson = {
                devDependencies: {},
                dependencies: {},
            };
            try {
                const file = (0, fs_1.readFileSync)(path, 'utf8');
                const parsedFile = JSON.parse(file);
                packageJson.dependencies = parsedFile.dependencies || {};
                packageJson.devDependencies = parsedFile.devDependencies || {};
            }
            catch (e) {
                throw new Error((0, ts_dedent_1.default) `The provided path in your eslintrc.json - ${path} is not a valid path to a package.json file or your package.json file is not in the same folder as ESLint is running from.

          Read more at: https://github.com/storybookjs/eslint-plugin-storybook/blob/main/docs/rules/no-uninstalled-addons.md
          `);
            }
            return packageJson;
        };
        const extractAllAddonsFromTheStorybookConfig = (addonsExpression) => {
            if (addonsExpression === null || addonsExpression === void 0 ? void 0 : addonsExpression.elements) {
                // extract all nodes taht are a string inside the addons array
                const nodesWithAddons = addonsExpression.elements
                    .map((elem) => ((0, ast_1.isLiteral)(elem) ? { value: elem.value, node: elem } : undefined))
                    .filter(excludeNullable);
                const listOfAddonsInString = nodesWithAddons.map((elem) => elem.value);
                // extract all nodes that are an object inside the addons array
                const nodesWithAddonsInObj = addonsExpression.elements
                    .map((elem) => ((0, ast_1.isObjectExpression)(elem) ? elem : { properties: [] }))
                    .map((elem) => {
                    const property = elem.properties.find((prop) => (0, ast_1.isProperty)(prop) && (0, ast_1.isIdentifier)(prop.key) && prop.key.name === 'name');
                    return (0, ast_1.isLiteral)(property === null || property === void 0 ? void 0 : property.value)
                        ? { value: property.value.value, node: property.value }
                        : undefined;
                })
                    .filter(excludeNullable);
                const listOfAddonsInObj = nodesWithAddonsInObj.map((elem) => elem.value);
                const listOfAddons = [...listOfAddonsInString, ...listOfAddonsInObj];
                const listOfAddonElements = [...nodesWithAddons, ...nodesWithAddonsInObj];
                return { listOfAddons, listOfAddonElements };
            }
            return { listOfAddons: [], listOfAddonElements: [] };
        };
        function reportUninstalledAddons(addonsProp) {
            const packageJsonPath = (0, path_1.resolve)(packageJsonLocation || `./package.json`);
            let packageJsonObject;
            try {
                packageJsonObject = getPackageJson(packageJsonPath);
            }
            catch (e) {
                // if we cannot find the package.json, we cannot check if the addons are installed
                throw new Error(e);
            }
            const depsAndDevDeps = mergeDepsWithDevDeps(packageJsonObject);
            const { listOfAddons, listOfAddonElements } = extractAllAddonsFromTheStorybookConfig(addonsProp);
            const result = areThereAddonsNotInstalled(listOfAddons, depsAndDevDeps);
            if (result) {
                const elemsWithErrors = listOfAddonElements.filter((elem) => !!result.find((addon) => addon.name === elem.value));
                const rootDir = process.cwd().split(path_1.sep).pop();
                const packageJsonPath = `${rootDir}${path_1.sep}${(0, path_1.relative)(process.cwd(), packageJsonLocation)}`;
                elemsWithErrors.forEach((elem) => {
                    context.report({
                        node: elem.node,
                        messageId: 'addonIsNotInstalled',
                        data: {
                            addonName: elem.value,
                            packageJsonPath,
                        },
                    });
                });
            }
        }
        function findAddonsPropAndReport(node) {
            const addonsProp = node.properties.find((prop) => (0, ast_1.isProperty)(prop) && (0, ast_1.isIdentifier)(prop.key) && prop.key.name === 'addons');
            if ((addonsProp === null || addonsProp === void 0 ? void 0 : addonsProp.value) && (0, ast_1.isArrayExpression)(addonsProp.value)) {
                reportUninstalledAddons(addonsProp.value);
            }
        }
        //----------------------------------------------------------------------
        // Public
        //----------------------------------------------------------------------
        return {
            AssignmentExpression: function (node) {
                if ((0, ast_1.isObjectExpression)(node.right)) {
                    findAddonsPropAndReport(node.right);
                }
            },
            ExportDefaultDeclaration: function (node) {
                const meta = (0, utils_1.getMetaObjectExpression)(node, context);
                if (!meta)
                    return null;
                findAddonsPropAndReport(meta);
            },
            ExportNamedDeclaration: function (node) {
                const addonsProp = (0, ast_1.isVariableDeclaration)(node.declaration) &&
                    node.declaration.declarations.find((decl) => (0, ast_1.isVariableDeclarator)(decl) && (0, ast_1.isIdentifier)(decl.id) && decl.id.name === 'addons');
                if (addonsProp && (0, ast_1.isArrayExpression)(addonsProp.init)) {
                    reportUninstalledAddons(addonsProp.init);
                }
            },
        };
    },
});
