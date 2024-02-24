package logging

type Option func()

func WithoutEnvironment() Option {
	return func() {
		withEnvironment = false
	}
}

func WithoutVersion() Option {
	return func() {
		withVersion = false
	}
}
