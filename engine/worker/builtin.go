package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/plugin"
)

var mapBuiltinActions = map[string]BuiltInActionFunc{}

func init() {
	mapBuiltinActions[sdk.ArtifactUpload] = runArtifactUpload
	mapBuiltinActions[sdk.ArtifactDownload] = runArtifactDownload
	mapBuiltinActions[sdk.ScriptAction] = runScriptAction
	mapBuiltinActions[sdk.JUnitAction] = runParseJunitTestResultAction
	mapBuiltinActions[sdk.GitCloneAction] = runGitClone
}

// BuiltInAction defines builtin action signature
type BuiltInAction func(context.Context, *sdk.Action, int64, []sdk.Parameter, LoggerFunc) sdk.Result

// BuiltInActionFunc returns the BuiltInAction given a worker
type BuiltInActionFunc func(*currentWorker) BuiltInAction

// LoggerFunc is the type for the logging function through BuiltInActions
type LoggerFunc func(format string)

func getLogger(w *currentWorker, buildID int64, stepOrder int) LoggerFunc {
	return func(s string) {
		if !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		w.sendLog(buildID, s, stepOrder, false)
	}
}

func (w *currentWorker) runBuiltin(ctx context.Context, a *sdk.Action, buildID int64, params []sdk.Parameter, stepOrder int) sdk.Result {
	defer w.drainLogsAndCloseLogger(ctx)

	//Define a loggin function
	sendLog := getLogger(w, buildID, stepOrder)

	f, ok := mapBuiltinActions[a.Name]
	if !ok {
		res := sdk.Result{
			Status: sdk.StatusFail.String(),
			Reason: fmt.Sprintf("Unknown builtin step: %s\n", a.Name),
		}
		return res
	}

	return f(w)(ctx, a, buildID, params, sendLog)
}

func (w *currentWorker) runPlugin(ctx context.Context, a *sdk.Action, buildID int64, params []sdk.Parameter, stepOrder int, sendLog LoggerFunc) sdk.Result {
	chanRes := make(chan sdk.Result)

	go func(buildID int64, params []sdk.Parameter) {
		res := sdk.Result{Status: sdk.StatusFail.String()}

		//For the moment we consider that plugin name = action name = plugin binary file name
		pluginName := a.Name
		//The binary file has been downloaded during requirement check in /tmp
		pluginBinary := path.Join(os.TempDir(), a.Name)

		var tlsskipverify bool
		if os.Getenv("CDS_SKIP_VERIFY") != "" {
			tlsskipverify = true
		}

		//Create the rpc server
		pluginClient := plugin.NewClient(ctx, pluginName, pluginBinary, w.id, w.apiEndpoint, tlsskipverify)
		defer pluginClient.Kill()

		//Get the plugin interface
		_plugin, err := pluginClient.Instance()
		if err != nil {
			result := sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: fmt.Sprintf("Unable to init plugin %s: %s\n", pluginName, err),
			}
			sendLog(result.Reason)
			chanRes <- result
		}

		//Manage all parameters
		pluginArgs := plugin.Arguments{
			Data: map[string]string{},
		}
		for _, p := range a.Parameters {
			pluginArgs.Data[p.Name] = p.Value
		}
		for _, p := range params {
			pluginArgs.Data[p.Name] = p.Value
		}
		for _, v := range w.currentJob.buildVariables {
			pluginArgs.Data["cds.build."+v.Name] = v.Value
		}

		//Call the Run function on the plugin interface
		id := w.currentJob.pbJob.PipelineBuildID
		if w.currentJob.wJob != nil {
			id = w.currentJob.wJob.WorkflowNodeRunID
		}

		pluginAction := plugin.Job{
			IDPipelineBuild:    id,
			IDPipelineJobBuild: buildID,
			OrderStep:          stepOrder,
			Args:               pluginArgs,
		}

		pluginResult := _plugin.Run(pluginAction)

		if pluginResult == plugin.Success {
			res.Status = sdk.StatusSuccess.String()
		}

		chanRes <- res
	}(buildID, params)

	for {
		select {
		case <-ctx.Done():
			log.Error("CDS Worker execution canceled: %v", ctx.Err())
			w.sendLog(buildID, "CDS Worker execution canceled\n", stepOrder, false)
			return sdk.Result{
				Status: sdk.StatusFail.String(),
				Reason: "CDS Worker execution canceled",
			}
		case res := <-chanRes:
			return res
		}
	}
}
