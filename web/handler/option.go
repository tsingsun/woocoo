package handler

// middlewareOptions middleware options for middleware to implement advanced features.if middleware config initial from
// Configuration,you don't need to use this.
//
// you need register the middleware again if you use MiddlewareOption to change the middleware behavior.
// example:
//
//	middleware := handler.JWT(handler.WithMiddlewareConfig(func() interface{} { return &jwt.Config{...} }))
//	web.RegisterMiddleware(middleware)
type middlewareOptions struct {
	// set the function that get the default middleware config,so you can change the default config
	configFunc MiddlewareDefaultConfigFunc
}

type MiddlewareOption func(o *middlewareOptions)

// MiddlewareDefaultConfigFunc is the function that get the default middleware config.
//
// if you want the middleware config pass to the middleware,you can use this function. for example:
// in Middleware.ApplyFunc, the config is set up by the configuration,you can not change it.in this case,you can use this function.
// function return value is the pointer of the middleware config.
type MiddlewareDefaultConfigFunc func() interface{}

// WithMiddlewareConfig use to change the default middleware config
func WithMiddlewareConfig(configFunc MiddlewareDefaultConfigFunc) MiddlewareOption {
	return func(o *middlewareOptions) {
		o.configFunc = configFunc
	}
}

func (o *middlewareOptions) applyOptions(options ...MiddlewareOption) {
	for _, option := range options {
		option(o)
	}
}
