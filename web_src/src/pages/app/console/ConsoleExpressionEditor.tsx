import { forwardRef } from "react";

import { ExpressionEditor, type ExpressionEditorProps } from "@/components/ExpressionEditor";

import { widgetCelAdapter } from "./widget/celAdapter";

type ConsoleExpressionEditorProps = Omit<ExpressionEditorProps, "dialect">;

export const ConsoleExpressionEditor = forwardRef<HTMLTextAreaElement, ConsoleExpressionEditorProps>(
  function ConsoleExpressionEditorRender({ expressionAdapter = widgetCelAdapter, ...props }, ref) {
    return <ExpressionEditor {...props} ref={ref} dialect="cel" expressionAdapter={expressionAdapter} />;
  },
);
