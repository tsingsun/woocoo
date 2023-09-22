package handler

// MiddlewareOptions middleware options for middleware to implement advanced features.if middleware config initial from
// Configuration, you don't need to use this.
//
// you need to register the middleware again if you use MiddlewareOption to change the middleware behavior.
// example:
//
//	middleware := handler.NewJWT(handler.WithMiddlewareConfig(func() any { return &jwt.Config{...} }))
//	web.RegisterMiddleware(middleware)
type MiddlewareOptions struct {
	// set the function that gets the default middleware config, so you can change the default config
	ConfigFunc MiddlewareInitConfigFunc
}

type MiddlewareOption func(o *MiddlewareOptions)

// MiddlewareInitConfigFunc is the function that gets the middleware config and mutate.
//
// if you want the middleware config passes to the middleware, you can use this function, for example,
// in Middleware.ApplyFunc, the config is set up by the configuration, you cannot change it.
// in this case,you can use this function to change the config before ApplyFunc.
//
// parameter config must be a pointer type.
type MiddlewareInitConfigFunc func(config any)

// WithMiddlewareConfig use to change the default middleware config
func WithMiddlewareConfig(configFunc MiddlewareInitConfigFunc) MiddlewareOption {
	return func(o *MiddlewareOptions) {
		o.ConfigFunc = configFunc
	}
}

// NewMiddlewareOption apply the options to MiddlewareOptions
func NewMiddlewareOption(options ...MiddlewareOption) MiddlewareOptions {
	o := MiddlewareOptions{}
	o.applyOptions(options...)
	return o
}

func (o *MiddlewareOptions) applyOptions(options ...MiddlewareOption) {
	for _, option := range options {
		option(o)
	}
}
