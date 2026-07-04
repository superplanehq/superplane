package authorization

import "fmt"

type HTTPRoute struct {
	Method  string
	Pattern string
}

func (route HTTPRoute) String() string {
	return fmt.Sprintf("%s %s", route.Method, route.Pattern)
}
