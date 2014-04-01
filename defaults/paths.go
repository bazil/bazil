package defaults

import (
	"github.com/Wessie/appdirs"
)

var app = appdirs.App{
	Name: "bazil",
}

func DataDir() string {
	return app.UserData()
}
