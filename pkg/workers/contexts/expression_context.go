package contexts

import (
	"gorm.io/gorm"
)

type ExpressionContext struct {
	tx                   *gorm.DB
	configurationBuilder *NodeConfigurationBuilder
}

func NewExpressionContext(tx *gorm.DB, configurationBuilder *NodeConfigurationBuilder) *ExpressionContext {
	return &ExpressionContext{tx: tx, configurationBuilder: configurationBuilder}
}

func (c *ExpressionContext) Run(expression string) (any, error) {
	return c.configurationBuilder.ResolveExpression(expression)
}
