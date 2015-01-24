// Package github integrates everything necessary to test commits, comment on
// pull requests and close them if the build failed.
package github

import (
	"io/ioutil"
	"leeroy/config"
	"leeroy/logging"
	"log"
	"net/http"
)

// Parse a GitHub request body and add it to the build queue.
func Parse(jobs chan logging.Job, req *http.Request, blog *logging.Buildlog,
	c *config.Config) {
	e := req.Header["X-Github-Event"][0]

	switch e {
	case "push":
		handlePush(req, jobs)
	case "pull_request":
		handlePR(req, blog, c)
	default:
		log.Println("event not supported", e)
	}
}

// Parse the body of a request.
func parseBody(req *http.Request) []byte {
	b, err := ioutil.ReadAll(req.Body)

	if err != nil {
		log.Println(err)
	}

	return b
}
