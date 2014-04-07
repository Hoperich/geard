package cmd

const WhenDaemon = "daemon"
const WhenLocal = "local"

type FuncInit func() error

func Initialize(when ...string) error {
	for i := range initializers {
		init := &initializers[i]
		if !init.run && (len(init.when) == 0 || any(init.when, when)) {
			if err := init.Func(); err != nil {
				return err
			}
			init.run = true
		}
	}
	return nil
}

// Helper function for invoking local initializers.
func LocalInitializers(funcs ...FuncInit) FuncInit {
	funcs = append(funcs, func() error { return Initialize(WhenLocal) })
	return all(funcs...)
}

func any(a, b []string) bool {
	for _, first := range a {
		for _, second := range b {
			if first == second {
				return true
			}
		}
	}
	return false
}

func all(funcs ...FuncInit) FuncInit {
	return func() error {
		for i := range funcs {
			if err := funcs[i](); err != nil {
				return err
			}
		}
		return nil
	}
}
