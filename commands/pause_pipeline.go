package commands

import (
	"fmt"

	"github.com/concourse/fly/internal/displayhelpers"
	"github.com/concourse/fly/rc"
)

type PausePipelineCommand struct {
	Pipeline string `short:"p"  long:"pipeline" required:"true" description:"Pipeline to pause"`
}

func (command *PausePipelineCommand) Execute(args []string) error {
	pipelineName := command.Pipeline

	client, err := rc.TargetClient(Fly.Target)
	if err != nil {
		return err
	}

	found, err := client.PausePipeline(pipelineName)
	if err != nil {
		return err
	}

	if found {
		fmt.Printf("paused '%s'\n", pipelineName)
	} else {
		displayhelpers.Failf("pipeline '%s' not found\n", pipelineName)
	}

	return nil
}
