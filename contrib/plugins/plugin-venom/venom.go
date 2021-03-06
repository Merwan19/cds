package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/ovh/cds/sdk/plugin"
	"github.com/runabove/venom"
	"github.com/runabove/venom/context/default"
	"github.com/runabove/venom/context/webctx"
	"github.com/runabove/venom/executors/exec"
	"github.com/runabove/venom/executors/http"
	"github.com/runabove/venom/executors/imap"
	"github.com/runabove/venom/executors/readfile"
	"github.com/runabove/venom/executors/smtp"
	"github.com/runabove/venom/executors/ssh"
	"github.com/runabove/venom/executors/web"
)

// VenomPlugin implements plugin interface
type VenomPlugin struct {
	plugin.Common
}

// Name returns the plugin name
func (s VenomPlugin) Name() string {
	return "plugin-venom"
}

// Description returns the plugin description
func (s VenomPlugin) Description() string {
	return `This plugin helps you to run venom. Venom: https://github.com/runabove/venom.

Add an extra step of type junit on your job to view tests results on CDS UI.`
}

// Author returns the plugin author's name
func (s VenomPlugin) Author() string {
	return "Yvonnick Esnault <yvonnick.esnault@corp.ovh.com>"
}

// Parameters return parameters description
func (s VenomPlugin) Parameters() plugin.Parameters {
	params := plugin.NewParameters()
	params.Add("path", plugin.StringParameter, "Path containers yml venom files. Format: adirectory/, ./*aTest.yml, ./foo/b*/**/z*.yml", ".")
	params.Add("exclude", plugin.TextParameter, "Exclude some files, one file per line", "")
	params.Add("parallel", plugin.StringParameter, "Launch Test Suites in parallel. Enter here number of routines", "2")
	params.Add("output", plugin.StringParameter, "Directory where output xunit result file", ".")
	params.Add("details", plugin.StringParameter, "Output Details Level: low, medium, high", "low")
	params.Add("loglevel", plugin.StringParameter, "Log Level: debug, info, warn or error", "error")
	params.Add("vars", plugin.StringParameter, "Empty: all {{.cds...}} vars will be rewrited. Otherwise, you can limit rewrite to some variables. Example, enter cds.app.yourvar,cds.build.foo,myvar=foo to rewrite {{.cds.app.yourvar}}, {{.cds.build.foo}} and {{.foo}}. Default: Empty", "")
	return params
}

type venomWriter struct {
	plugin.IJob
}

func (w venomWriter) Write(buf []byte) (int, error) {
	err := plugin.SendLog(w, "VENOM %s", buf)
	return 0, err
}

// Run execute the action
func (s VenomPlugin) Run(a plugin.IJob) plugin.Result {
	// Parse parameters
	path := a.Arguments().Get("path")
	exclude := a.Arguments().Get("exclude")
	parallel := a.Arguments().Get("parallel")
	output := a.Arguments().Get("output")
	details := a.Arguments().Get("details")
	loglevel := a.Arguments().Get("loglevel")
	vars := a.Arguments().Get("vars")

	if path == "" {
		path = "."
	}
	p, err := strconv.Atoi(parallel)
	if err != nil {
		plugin.SendLog(a, "VENOM - parallel arg must be an integer\n")
		return plugin.Fail
	}

	venom.RegisterExecutor(exec.Name, exec.New())
	venom.RegisterExecutor(http.Name, http.New())
	venom.RegisterExecutor(imap.Name, imap.New())
	venom.RegisterExecutor(readfile.Name, readfile.New())
	venom.RegisterExecutor(smtp.Name, smtp.New())
	venom.RegisterExecutor(ssh.Name, ssh.New())
	venom.RegisterExecutor(web.Name, web.New())

	venom.RegisterTestCaseContext(defaultctx.Name, defaultctx.New())
	venom.RegisterTestCaseContext(webctx.Name, webctx.New())

	venom.PrintFunc = func(format string, aa ...interface{}) (n int, err error) {
		plugin.SendLog(a, format, aa)
		return 0, nil
	}

	start := time.Now()
	w := venomWriter{a}
	data := make(map[string]string)
	if vars == "" {
		// no vars -> all .cds... variables can by used in yml
		data = a.Arguments().Data
	} else {
		// if vars is not empty
		// vars could be:
		// cds.foo.bar,cds.foo2.bar2
		// cds.foo.bar,cds.foo2.bar2,anotherVars=foo,anotherVars2=bar
		for _, v := range strings.Split(vars, ",") {
			t := strings.Split(v, "=")
			if len(t) > 1 {
				// if value of current var is setted, we take it
				data[t[0]] = t[1]
				plugin.SendLog(a, "VENOM - var %s has value %s\n", t[0], t[1])
			} else if len(t) == 1 && strings.HasPrefix(v, "cds.") {
				plugin.SendLog(a, "VENOM - try fo find var %s in cds variables\n", v)
				// if var starts with .cds, we try to take value from current CDS variables
				for k := range a.Arguments().Data {
					if k == v {
						plugin.SendLog(a, "VENOM - var %s is found with value %s\n", v, a.Arguments().Data[k])
						data[k] = a.Arguments().Data[k]
						break
					}
				}
			}
		}
	}
	tests, err := venom.Process([]string{path}, data, []string{exclude}, p, loglevel, details, w)
	if err != nil {
		plugin.SendLog(a, "VENOM - Fail on venom: %s\n", err)
		return plugin.Fail
	}

	elapsed := time.Since(start)
	plugin.SendLog(a, "VENOM - Output test results under: %s\n", output)
	if err := venom.OutputResult("xml", false, true, output, *tests, elapsed, "low"); err != nil {
		plugin.SendLog(a, "VENOM - Error while uploading test results: %s\n", err)
		return plugin.Fail
	}

	return plugin.Success
}

func main() {
	plugin.Main(&VenomPlugin{})
}
