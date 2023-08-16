package cli

import (
	"github.com/felipemarinho97/dev-spaces/handlers"
	"github.com/urfave/cli/v2"
)

func destroyCommand(c *cli.Context) error {
	h := c.Context.Value("handler").(*handlers.Handler)

	name := c.String("name")

	return h.Destroy(c.Context, handlers.DestroyOptions{
		Name: name,
	})
}
