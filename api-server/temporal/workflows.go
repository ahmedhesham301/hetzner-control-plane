package temporal

import (
	"encoding/json"
	"time"

	"go.temporal.io/sdk/workflow"
)

type CreateServiceWorkflowParams struct {
	RawService    json.RawMessage
	Env           string
	TemplatesPath string
	NetworkID     int64
}

func CreateServiceWorkflow(ctx workflow.Context, params CreateServiceWorkflowParams) error {
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 15,
	}

	ctx = workflow.WithActivityOptions(ctx, activityOptions)
	logger := workflow.GetLogger(ctx)

	// Check if image exists
	var imageID *int64
	err := workflow.ExecuteActivity(ctx, checkImageExist, params.RawService).Get(ctx, &imageID)
	if err != nil {
		logger.Error("error checking if image exists", "err", err)
		return err
	}
	// If not build it
	if imageID == nil {
		imageParams := buildImageParams{
			ServiceJSON:   params.RawService,
			Env:           params.Env,
			TemplatesPath: params.TemplatesPath,
			NetworkID:     params.NetworkID,
		}
		err := workflow.ExecuteActivity(ctx, buildImage, imageParams).Get(ctx, &imageID)
		if err != nil {
			logger.Error("error building image", "err", err)
			return err
		}
	}
	// create a firewall that allows traffic if env is dev
	var allowAllFirewallID *int64
	if params.Env == "dev" {
		err = workflow.ExecuteActivity(ctx, GetOrCreateAllowAllFirewall).Get(ctx, &allowAllFirewallID)
		if err != nil {
			logger.Error("error building image", "err", err)
			return err
		}
	}

	// Deploy it
	deployParams := deployServiceParams{
		ServiceJSON:        params.RawService,
		ImageID:            *imageID,
		Env:                params.Env,
		AllowAllFirewallID: allowAllFirewallID,
		NetworkID:          params.NetworkID,
	}
	err = workflow.ExecuteActivity(ctx, deployService, deployParams).Get(ctx, nil)
	if err != nil {
		logger.Error("error building image", "err", err)
		return err
	}
	return nil
}
