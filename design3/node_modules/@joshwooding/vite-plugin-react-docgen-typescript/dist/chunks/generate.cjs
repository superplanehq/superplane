'use strict';

const ts = require('typescript');
const MagicString = require('magic-string');

function _interopNamespaceDefault(e) {
	const n = Object.create(null);
	if (e) {
		for (const k in e) {
			n[k] = e[k];
		}
	}
	n.default = e;
	return n;
}

const ts__namespace = /*#__PURE__*/_interopNamespaceDefault(ts);

function createLiteral(value) {
  switch (typeof value) {
    case "string":
      return ts__namespace.factory.createStringLiteral(value);
    case "number":
      return ts__namespace.factory.createNumericLiteral(value);
    case "boolean":
      return value ? ts__namespace.factory.createTrue() : ts__namespace.factory.createFalse();
  }
}
function insertTsIgnoreBeforeStatement(statement) {
  ts__namespace.setSyntheticLeadingComments(statement, [
    {
      text: " @ts-ignore",
      kind: ts__namespace.SyntaxKind.SingleLineCommentTrivia,
      pos: -1,
      end: -1
    }
  ]);
  return statement;
}
function setDisplayName(d) {
  return insertTsIgnoreBeforeStatement(
    ts__namespace.factory.createExpressionStatement(
      ts__namespace.factory.createBinaryExpression(
        ts__namespace.factory.createPropertyAccessExpression(
          ts__namespace.factory.createIdentifier(d.displayName),
          ts__namespace.factory.createIdentifier("displayName")
        ),
        ts__namespace.SyntaxKind.EqualsToken,
        ts__namespace.factory.createStringLiteral(d.displayName)
      )
    )
  );
}
function createPropDefinition(propName, prop, options) {
  const setDefaultValue = (defaultValue) => ts__namespace.factory.createPropertyAssignment(
    ts__namespace.factory.createStringLiteral("defaultValue"),
    defaultValue?.value !== void 0 && (typeof defaultValue.value === "string" || typeof defaultValue.value === "number" || typeof defaultValue.value === "boolean") ? ts__namespace.factory.createObjectLiteralExpression([
      ts__namespace.factory.createPropertyAssignment(
        ts__namespace.factory.createIdentifier("value"),
        createLiteral(defaultValue.value)
      )
    ]) : ts__namespace.factory.createNull()
  );
  const setStringLiteralField = (fieldName, fieldValue) => ts__namespace.factory.createPropertyAssignment(
    ts__namespace.factory.createStringLiteral(fieldName),
    ts__namespace.factory.createStringLiteral(fieldValue)
  );
  const setDescription = (description) => setStringLiteralField("description", description);
  const setName = (name) => setStringLiteralField("name", name);
  const setRequired = (required) => ts__namespace.factory.createPropertyAssignment(
    ts__namespace.factory.createStringLiteral("required"),
    required ? ts__namespace.factory.createTrue() : ts__namespace.factory.createFalse()
  );
  const setValue = (typeValue) => Array.isArray(typeValue) && typeValue.every((value) => typeof value.value === "string") ? ts__namespace.factory.createPropertyAssignment(
    ts__namespace.factory.createStringLiteral("value"),
    ts__namespace.factory.createArrayLiteralExpression(
      typeValue.map(
        (value) => ts__namespace.factory.createObjectLiteralExpression([
          setStringLiteralField("value", value.value)
        ])
      )
    )
  ) : void 0;
  const setType = (typeName, typeValue) => {
    const objectFields = [setStringLiteralField("name", typeName)];
    const valueField = setValue(typeValue);
    if (valueField) {
      objectFields.push(valueField);
    }
    return ts__namespace.factory.createPropertyAssignment(
      ts__namespace.factory.createStringLiteral(options.typePropName),
      ts__namespace.factory.createObjectLiteralExpression(objectFields)
    );
  };
  return ts__namespace.factory.createPropertyAssignment(
    ts__namespace.factory.createStringLiteral(propName),
    ts__namespace.factory.createObjectLiteralExpression([
      setDefaultValue(prop.defaultValue),
      setDescription(prop.description),
      setName(prop.name),
      setRequired(prop.required),
      setType(prop.type.name, prop.type.value)
    ])
  );
}
function setComponentDocGen(d, options) {
  return insertTsIgnoreBeforeStatement(
    ts__namespace.factory.createExpressionStatement(
      ts__namespace.factory.createBinaryExpression(
        ts__namespace.factory.createPropertyAccessExpression(
          ts__namespace.factory.createIdentifier(d.displayName),
          ts__namespace.factory.createIdentifier("__docgenInfo")
        ),
        ts__namespace.SyntaxKind.EqualsToken,
        ts__namespace.factory.createObjectLiteralExpression([
          ts__namespace.factory.createPropertyAssignment(
            ts__namespace.factory.createStringLiteral("description"),
            ts__namespace.factory.createStringLiteral(d.description)
          ),
          ts__namespace.factory.createPropertyAssignment(
            ts__namespace.factory.createStringLiteral("displayName"),
            ts__namespace.factory.createStringLiteral(d.displayName)
          ),
          ts__namespace.factory.createPropertyAssignment(
            ts__namespace.factory.createStringLiteral("props"),
            ts__namespace.factory.createObjectLiteralExpression(
              Object.entries(d.props).map(
                ([propName, prop]) => createPropDefinition(propName, prop, options)
              )
            )
          )
        ])
      )
    )
  );
}
function generateDocgenCodeBlock(options) {
  const sourceFile = ts__namespace.createSourceFile(
    options.filename,
    options.source,
    ts__namespace.ScriptTarget.ESNext
  );
  const wrapInTryStatement = (statements) => ts__namespace.factory.createTryStatement(
    ts__namespace.factory.createBlock(statements, true),
    ts__namespace.factory.createCatchClause(
      ts__namespace.factory.createVariableDeclaration(
        ts__namespace.factory.createIdentifier("__react_docgen_typescript_loader_error")
      ),
      ts__namespace.factory.createBlock([])
    ),
    void 0
  );
  const codeBlocks = options.componentDocs.map(
    (d) => wrapInTryStatement(
      [
        options.setDisplayName ? setDisplayName(d) : null,
        setComponentDocGen(d, options)
      ].filter((s2) => s2 !== null)
    )
  );
  const printer = ts__namespace.createPrinter({ newLine: ts__namespace.NewLineKind.LineFeed });
  const printNode = (sourceNode) => printer.printNode(ts__namespace.EmitHint.Unspecified, sourceNode, sourceFile);
  const s = new MagicString(options.source);
  codeBlocks.forEach((node) => {
    s.append(printNode(node));
  });
  return {
    code: s.toString(),
    map: s.generateMap()
  };
}

exports.generateDocgenCodeBlock = generateDocgenCodeBlock;
