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
