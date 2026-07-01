package contexts

type ExpressionContext struct {
	configurationBuilder *NodeConfigurationBuilder
}

func NewExpressionContext(configurationBuilder *NodeConfigurationBuilder) *ExpressionContext {
	return &ExpressionContext{configurationBuilder: configurationBuilder}
}

func (c *ExpressionContext) Run(expression string) (any, error) {
	return c.configurationBuilder.ResolveExpression(expression)
}

func (c *ExpressionContext) RunWithExtraVariables(expression string, variables map[string]any) (any, error) {
	return c.configurationBuilder.ResolveExpressionWithExtraVariables(expression, variables)
}

func (c *ExpressionContext) BuildExecutionMessageChain() (map[string]any, error) {
	return c.configurationBuilder.BuildExecutionMessageChain()
}
