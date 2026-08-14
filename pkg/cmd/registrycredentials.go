package cmd

import (
	"github.com/kernel/hypeman-go"
	"github.com/urfave/cli/v3"
)

// registryCredentialFlags are the Docker-style registry credentials that the
// server borrows for a single image pull or push request. They are shared by
// every command that talks to a remote registry on the caller's behalf.
func registryCredentialFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "username",
			Usage: "Registry username",
		},
		&cli.StringFlag{
			Name:  "password",
			Usage: "Registry password or access token",
		},
		&cli.StringFlag{
			Name:  "registry-token",
			Usage: "Bearer token for an Authorization header",
		},
	}
}

func registryCredentialsFromCommand(cmd *cli.Command) (hypeman.PushCredentialsParam, bool) {
	return buildRegistryCredentials(
		cmd.String("username"),
		cmd.String("password"),
		cmd.String("registry-token"),
	)
}

// buildRegistryCredentials reports false when nothing was supplied, so callers
// can leave the field unset and let the server use its own registry
// credentials.
func buildRegistryCredentials(username, password, registryToken string) (hypeman.PushCredentialsParam, bool) {
	credentials := hypeman.PushCredentialsParam{}
	supplied := false

	if username != "" {
		credentials.Username = hypeman.Opt(username)
		supplied = true
	}
	if password != "" {
		credentials.Password = hypeman.Opt(password)
		supplied = true
	}
	if registryToken != "" {
		credentials.RegistryToken = hypeman.Opt(registryToken)
		supplied = true
	}

	return credentials, supplied
}
