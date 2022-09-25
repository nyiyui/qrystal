package creds

type ProvidedServer struct {
	CertPath string `yaml:"cert-path"`
	KeyPath  string `yaml:"key-path"`
}

type ProvidedClient struct {
	CertPath string `yaml:"cert-path"`
}

type ACME struct {
	Domain       string `yaml:"domain"`
	SSLEmail     string `yaml:"email"`
	DirectoryURL string `yaml:"directory-url"`
}

// ServerConfig specifies where to get certificates and keys from for a server.
// Each field is used in order if it is non-nil.
type ServerConfig struct {
	Provided *ProvidedServer `yaml:"provided"`
	ACME     *ACME           `yaml:"acme"`
}

// ClientConfig specifies where to get certificates from for a client.
// Each field is used in order if it is non-nil.
type ClientConfig struct {
	Provided *ProvidedClient `yaml:"provided"`
	RootCA   *struct{}       `yaml:"root-ca"`
}
