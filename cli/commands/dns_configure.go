package commands

import (
	"fmt"
	"net/http"

	"github.com/felipemarinho97/dev-spaces/cli/config"
	"github.com/urfave/cli/v2"
)

func DNSConfigureCommand(c *cli.Context) error {
	config := c.Context.Value("config").(*config.Config)

	token := c.String("token")
	if token != "" {
		config.DNS.Token = token
		config.Save()
	}

	endpoint := c.String("endpoint")
	if endpoint != "" {
		config.DNS.Endpoint = endpoint
		config.Save()
	}

	domain := c.String("domain")
	if domain != "" {
		config.DNS.Domain = domain
		config.Save()
	}

	// if the 3 flags are not set, load the oauth2 flow from dns.devspaces.online
	if token == "" && endpoint == "" && domain == "" {
		fmt.Println("Please visit https://dns.devspaces.online and complete the authentication process to configure your DNS settings.")

		// wait for the refresh_token on port 3000
		fmt.Println("Waiting for the authentication to complete...")

		// create a webserver to listen for the refresh_token
		server := &http.Server{Addr: ":3000"}

		// create a channel to wait for the refresh_token
		done := make(chan bool)

		// create a handler to handle the request
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// get the refresh_token from the query string
			refreshToken := r.URL.Query().Get("refresh_token")

			// set the endpoint and domain
			config.DNS.Endpoint = "https://dns.devspaces.online/update"
			config.DNS.Domain = "devspaces.online"

			// set and save the refresh_token in the config file
			config.DNS.Token = refreshToken
			config.Save()

			// send a response to the user
			w.Write([]byte("You can close this window now."))
			done <- true
		})

		// start the server
		go server.ListenAndServe()

		// wait for the refresh_token
		<-done

	}

	fmt.Println("DNS settings configured successfully 🎉")

	return nil
}
