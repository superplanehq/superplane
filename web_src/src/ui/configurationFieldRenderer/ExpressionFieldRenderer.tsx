import React, { useState, useEffect } from "react";
import { Plus, Trash2, Code, FormInput } from "lucide-react";
import { Button } from "../button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../select";
import { Textarea } from "@/components/ui/textarea";
import { FieldRendererProps } from "./types";
import { Tabs, TabsList, TabsTrigger } from "../tabs";

interface Condition {
  variable: string;
  operator: string;
  value: string;
}

interface ExpressionStructure {
  conditions: Condition[];
  logicalOperators: string[]; // "and" or "or" between conditions
}

const OPERATORS = [
  { value: "==", label: "equals" },
  { value: "!=", label: "not equals" },
  { value: ">", label: "greater than" },
  { value: "<", label: "less than" },
  { value: ">=", label: "greater than or equal" },
  { value: "<=", label: "less than or equal" },
  { value: "contains", label: "contains" },
  { value: "startswith", label: "starts with" },
  { value: "endswith", label: "ends with" },
  { value: "matches", label: "matches" },
  { value: "in", label: "in" },
];

const LOGICAL_OPERATORS = [
  { value: "and", label: "AND" },
  { value: "or", label: "OR" },
];

/**
 * Parses an expression string into a structured format
 */
function parseExpressionToStructure(expression: string): ExpressionStructure {
  if (!expression || expression.trim() === "") {
    return { conditions: [], logicalOperators: [] };
  }

  const conditions: Condition[] = [];
  const logicalOperators: string[] = [];

  // Split by logical operators while preserving them
  const logicalOpRegex = /\s+(&&|\|\||and|or)\s+/gi;
  const parts = expression.split(logicalOpRegex);

  for (let i = 0; i < parts.length; i += 2) {
    const comparisonPart = parts[i]?.trim();
    if (!comparisonPart) continue;

    // Find the comparison operator
    const comparisonOpRegex = /\s*(>=|<=|==|!=|>|<|contains|startswith|endswith|matches|in)\s*/i;
    const match = comparisonPart.match(comparisonOpRegex);

    if (match && match.index !== undefined) {
      const operator = match[1];
      const matchStart = match.index;
      const fullMatch = match[0];
      const operatorInMatch = fullMatch.indexOf(operator);
      const operatorIndex = matchStart + operatorInMatch;
      const variable = comparisonPart.substring(0, operatorIndex).trim();
      const value = comparisonPart.substring(operatorIndex + operator.length).trim();

      conditions.push({
        variable,
        operator,
        value,
      });
    } else {
      // If we can't parse it, create a condition with the whole part as variable
      conditions.push({
        variable: comparisonPart,
        operator: "==",
        value: "",
      });
    }

    // Get the logical operator after this condition (if any)
    if (i + 1 < parts.length) {
      const logicalOp = parts[i + 1]?.trim().toLowerCase();
      if (logicalOp === "&&" || logicalOp === "and") {
        logicalOperators.push("and");
      } else if (logicalOp === "||" || logicalOp === "or") {
        logicalOperators.push("or");
      }
    }
  }

  return { conditions, logicalOperators };
}

/**
 * Converts a structured expression back to a string
 */
function structureToExpression(structure: ExpressionStructure): string {
  if (structure.conditions.length === 0) {
    return "";
  }

  const parts: string[] = [];

  for (let i = 0; i < structure.conditions.length; i++) {
    const condition = structure.conditions[i];
    const variable = condition.variable.trim();
    const operator = condition.operator.trim();
    const value = condition.value.trim();

    // Format the value - if it's not already quoted and contains spaces or special chars, quote it
    let formattedValue = value;
    if (
      value &&
      !value.startsWith('"') &&
      !value.startsWith("'") &&
      !value.match(/^-?\d+$/) &&
      value !== "true" &&
      value !== "false" &&
      value !== "null" &&
      value !== "undefined"
    ) {
      // Check if it needs quoting
      if (value.includes(" ") || value.includes("$") || value.includes(".")) {
        formattedValue = `"${value.replace(/"/g, '\\"')}"`;
      }
    }

    parts.push(`${variable} ${operator} ${formattedValue}`);

    // Add logical operator if there's one for this position
    if (i < structure.logicalOperators.length) {
      parts.push(structure.logicalOperators[i]);
    }
  }

  return parts.join(" ");
}

export const ExpressionFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const expressionString = (value as string) || "";
  const [mode, setMode] = useState<"form" | "text">("form");
  const [structure, setStructure] = useState<ExpressionStructure>(() => parseExpressionToStructure(expressionString));
  const [textValue, setTextValue] = useState(expressionString);

  // Update structure when expression string changes externally
  useEffect(() => {
    if (mode === "form" && expressionString !== structureToExpression(structure)) {
      const parsed = parseExpressionToStructure(expressionString);
      setStructure(parsed);
    }
  }, [expressionString, mode]);

  // Sync text value when switching to text mode
  useEffect(() => {
    if (mode === "text") {
      setTextValue(expressionString || structureToExpression(structure));
    }
  }, [mode]);

  const handleStructureChange = (newStructure: ExpressionStructure) => {
    setStructure(newStructure);
    const newExpression = structureToExpression(newStructure);
    onChange(newExpression || undefined);
  };

  const handleTextChange = (newText: string) => {
    setTextValue(newText);
    onChange(newText || undefined);
  };

  const addCondition = () => {
    const newCondition: Condition = {
      variable: "$.",
      operator: "==",
      value: "",
    };
    const newStructure: ExpressionStructure = {
      conditions: [...structure.conditions, newCondition],
      logicalOperators: [...structure.logicalOperators, "and"],
    };
    handleStructureChange(newStructure);
  };

  const removeCondition = (index: number) => {
    const newConditions = structure.conditions.filter((_, i) => i !== index);
    const newLogicalOperators = structure.logicalOperators.filter((_, i) => i !== index - 1);
    const newStructure: ExpressionStructure = {
      conditions: newConditions,
      logicalOperators: newLogicalOperators,
    };
    handleStructureChange(newStructure);
  };

  const updateCondition = (index: number, field: keyof Condition, newValue: string) => {
    const newConditions = [...structure.conditions];
    newConditions[index] = { ...newConditions[index], [field]: newValue };
    const newStructure: ExpressionStructure = {
      conditions: newConditions,
      logicalOperators: structure.logicalOperators,
    };
    handleStructureChange(newStructure);
  };

  const updateLogicalOperator = (index: number, newOperator: string) => {
    const newLogicalOperators = [...structure.logicalOperators];
    newLogicalOperators[index] = newOperator;
    const newStructure: ExpressionStructure = {
      conditions: structure.conditions,
      logicalOperators: newLogicalOperators,
    };
    handleStructureChange(newStructure);
  };

  const handleModeChange = (newMode: "form" | "text") => {
    if (newMode === "text") {
      // Convert structure to expression when switching to text mode
      const expression = structureToExpression(structure);
      setTextValue(expression);
      onChange(expression || undefined);
    } else {
      // Parse expression when switching to form mode
      const parsed = parseExpressionToStructure(textValue);
      setStructure(parsed);
      onChange(textValue || undefined);
    }
    setMode(newMode);
  };

  return (
    <div className="space-y-3">
      {/* Mode Toggle */}
      <div className="flex items-center justify-between">
        <label className="text-sm font-medium text-gray-700 dark:text-gray-300">{field.label || "Expression"}</label>
        <Tabs value={mode} onValueChange={(v) => handleModeChange(v as "form" | "text")} className="w-auto">
          <TabsList className="h-8">
            <TabsTrigger value="form" className="text-xs px-3">
              <FormInput className="h-3 w-3 mr-1" />
              Form
            </TabsTrigger>
            <TabsTrigger value="text" className="text-xs px-3">
              <Code className="h-3 w-3 mr-1" />
              Text
            </TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {/* Form Mode */}
      {mode === "form" && (
        <div className="space-y-3">
          {structure.conditions.length === 0 ? (
            <div className="text-sm text-gray-500 dark:text-gray-400 text-center py-4 border border-dashed border-gray-300 dark:border-gray-600 rounded">
              No conditions yet. Click "Add Condition" to get started.
            </div>
          ) : (
            structure.conditions.map((condition, index) => (
              <div key={index} className="space-y-2">
                <div className="flex gap-2 items-start">
                  <div className="flex-1 grid grid-cols-3 gap-2">
                    <Input
                      type="text"
                      value={condition.variable}
                      onChange={(e) => updateCondition(index, "variable", e.target.value)}
                      placeholder="$.field"
                      className={hasError ? "border-red-500 border-2" : ""}
                    />
                    <Select value={condition.operator} onValueChange={(val) => updateCondition(index, "operator", val)}>
                      <SelectTrigger className={hasError ? "border-red-500 border-2" : ""}>
                        <SelectValue placeholder="Operator" />
                      </SelectTrigger>
                      <SelectContent>
                        {OPERATORS.map((op) => (
                          <SelectItem key={op.value} value={op.value}>
                            {op.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Input
                      type="text"
                      value={condition.value}
                      onChange={(e) => updateCondition(index, "value", e.target.value)}
                      placeholder="value"
                      className={hasError ? "border-red-500 border-2" : ""}
                    />
                  </div>
                  <Button variant="ghost" size="icon" onClick={() => removeCondition(index)} className="mt-1">
                    <Trash2 className="h-4 w-4 text-red-500" />
                  </Button>
                </div>
                {/* Logical Operator between conditions */}
                {index < structure.conditions.length - 1 && (
                  <div className="flex items-center justify-center">
                    <Select
                      value={structure.logicalOperators[index] || "and"}
                      onValueChange={(val) => updateLogicalOperator(index, val)}
                    >
                      <SelectTrigger className="w-24">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {LOGICAL_OPERATORS.map((op) => (
                          <SelectItem key={op.value} value={op.value}>
                            {op.label}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                )}
              </div>
            ))
          )}
          <Button variant="outline" onClick={addCondition} className="w-full">
            <Plus className="h-4 w-4 mr-2" />
            Add Condition
          </Button>
        </div>
      )}

      {/* Text Mode */}
      {mode === "text" && (
        <Textarea
          value={textValue}
          onChange={(e) => handleTextChange(e.target.value)}
          placeholder={field.placeholder || 'e.g., $.status == "active" && $.count > 10'}
          className={`min-h-[100px] font-mono text-sm ${hasError ? "border-red-500 border-2" : ""}`}
        />
      )}

      {field.description && <p className="text-xs text-gray-500 dark:text-gray-400">{field.description}</p>}
    </div>
  );
};
