package cmder

import (
	"github.com/urfave/cli"
)

var (
	appInstance *cli.App
)

func SetApp(app *cli.App) {
	appInstance = app
}

func App() *cli.App {
	return appInstance
}
