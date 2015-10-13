package server

type appOption func(*appConfig) error

type AppOption appOption

type appConfig struct {
	debug func(msg interface{})
}

func Debug(fn func(msg interface{})) AppOption {
	return func(conf *appConfig) error {
		conf.debug = fn
		return nil
	}
}
