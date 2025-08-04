package client

type Config struct {
    ServerURL              string
    LocalPort              int
    Subdomain              string
    AuthToken              string
    Verbose                bool
    UseSubdomainLocalhost  bool
}
