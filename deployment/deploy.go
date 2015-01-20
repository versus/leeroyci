// Package deployment deploys code that passed all tests.
package deployment

import (
	"leeroy/config"
	"leeroy/logging"
	"leeroy/notification"
	"log"
	"os/exec"
)

// Deploy a job if all tests are passed.
func Deploy(j *logging.Job, c *config.Config) {
	if j.Success() != true {
		log.Println("Not deploying", j.Branch, "build did not succeed.")
		return
	}

	r, err := c.ConfigForRepo(j.URL)

	if err != nil {
		log.Println("Repo", j.URL, "does not exist.")
		return
	}

	d, err := r.DeployTarget(j.Branch)

	if err != nil {
		log.Println("Deployment target for", j.Branch, "does not exist")
		return
	}

	notification.Notify(c, j, notification.KindDeployStart)

	o, err := call(d.Execute, r.URL, j.Branch)

	t := logging.Task{
		Command: d.Execute,
		Output:  o,
	}

	if err != nil {
		t.Return = err.Error()
	}

	j.Deployed = &t

	if err != nil {
		log.Println(err.Error())
	}

	notification.Notify(c, j, notification.KindDeployDone)
}

// Call a deployment script and return the output.
func call(app string, repo string, branch string) (string, error) {
	cmd := exec.Command(app, repo, branch)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
