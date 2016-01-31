package dns

type Config struct {
	LogLevel string `default:"debug"`
	Bind     string `default:":53"`
	Domain   string `default:"pgp.re."`
	Hostname string `default:"dns.pgp.st"`
	Email    string `default:"hello@pgp.st"`
	Redis    string `default:"redis://127.0.0.1:6379"`
}
