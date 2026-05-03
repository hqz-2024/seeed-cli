package commands

import (
	"fmt"
    "github.com/urfave/cli/v3"
)

func CustomHelp(){
	    // EXAMPLE: Append to an existing template
    cli.RootCommandHelpTemplate = fmt.Sprintf(`


 ███████╗███████╗███████╗███████╗██████╗       ██████╗██╗     ██╗
 ██╔════╝██╔════╝██╔════╝██╔════╝██╔══██╗     ██╔════╝██║     ██║
 ███████╗█████╗  █████╗  █████╗  ██║  ██║     ██║     ██║     ██║
 ╚════██║██╔══╝  ██╔══╝  ██╔══╝  ██║  ██║     ██║     ██║     ██║
 ███████║███████╗███████╗███████╗██████╔╝     ╚██████╗███████╗██║
 ╚══════╝╚══════╝╚══════╝╚══════╝╚═════╝       ╚═════╝╚══════╝╚═╝

	%s`, cli.RootCommandHelpTemplate)
}