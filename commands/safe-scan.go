package commands

import (
    "fmt"  
    "context"

    "github.com/urfave/cli/v3" 
)


func HandleSafeScan(context.Context, *cli.Command) error {
	fmt.Println("HandleSafeScan")
	return nil
}
