package server

type Options struct {
	Address string
	TLS     *TLS
}

type TLS struct {
	CertFile string
	KeyFile  string
}

type Option func(o *Options)

func WithAddress(addr string) Option {
	return func(o *Options) {
		o.Address = addr
	}
}

func WithTLS(certFile, keyFile string) Option {
	return func(o *Options) {
		o.TLS = &TLS{
			CertFile: certFile,
			KeyFile:  keyFile,
		}
	}
}
