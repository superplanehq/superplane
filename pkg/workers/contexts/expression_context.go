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

func (c *ExpressionContext) RunWithScope(expression string, scope map[string]any) (any, error) {
	return c.configurationBuilder.ResolveExpressionWithScope(expression, scope)
}
