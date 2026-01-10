import React from "react";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "../button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../select";
import { FieldRendererProps } from "./types";

interface Predicate {
  type: string;
  value: string;
}

export const AnyPredicateListFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const predicates: Predicate[] = Array.isArray(value) ? value : [];
  const operators = field.typeOptions?.anyPredicateList?.operators ?? [];

  const addPredicate = () => {
    const newPredicate: Predicate = {
      type: operators[0]?.value ?? "",
      value: "",
    };
    onChange([...predicates, newPredicate]);
  };

  const removePredicate = (index: number) => {
    const newPredicates = predicates.filter((_, i) => i !== index);
    onChange(newPredicates.length > 0 ? newPredicates : undefined);
  };

  const updatePredicate = (index: number, field: keyof Predicate, newValue: string) => {
    const newPredicates = [...predicates];
    newPredicates[index] = { ...newPredicates[index], [field]: newValue };
    onChange(newPredicates);
  };

  return (
    <div className="space-y-3">
      {predicates.map((predicate, index) => (
        <div key={index} className="flex gap-2 items-start">
          <div className="flex-1 grid grid-cols-2 gap-2">
            <Select value={predicate.type} onValueChange={(val) => updatePredicate(index, "type", val)}>
              <SelectTrigger className={`w-full ${hasError ? "border-red-500 border-2" : ""}`}>
                <SelectValue placeholder="Select operator" />
              </SelectTrigger>
              <SelectContent>
                {operators.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value ?? ""}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Input
              type="text"
              value={predicate.value ?? ""}
              onChange={(e) => updatePredicate(index, "value", e.target.value)}
              placeholder="Value"
              className={hasError ? "border-red-500 border-2" : ""}
            />
          </div>
          <Button variant="ghost" size="icon" onClick={() => removePredicate(index)} className="mt-1">
            <Trash2 className="h-4 w-4 text-red-500" />
          </Button>
        </div>
      ))}
      <Button variant="outline" onClick={addPredicate} className="w-full mt-3">
        <Plus className="h-4 w-4 mr-2" />
        Add Condition
      </Button>
    </div>
  );
};
