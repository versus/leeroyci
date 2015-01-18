// Package github integrates everything necessary to test commits, comment on
// pull requests and close them if the build failed.
package github

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"leeroy/config"
	"leeroy/logging"
	"log"
	"net/http"
	"time"
)

// TODO: parse full body, not just the fields needed

// PRCallback handles pull requests coming from GitHubs webhook.
type PRCallback struct {
	Number int
	Action string
	PR     PRPullRequest `json:"pull_request"`
}

// PRPullRequest stores the most basic information about a pull request.
type PRPullRequest struct {
	URL         string `json:"url"`
	State       string
	CommentsURL string `json:"comments_url"`
	Head        PRCommit
}

// PRCommit points to the latest commit and repository of a pull request.
type PRCommit struct {
	Commit     string `json:"sha"`
	Repository PRRepo `json:"repo"`
}

// PRRepo stores the repository URL of a pull request.
type PRRepo struct {
	HTMLURL string `json:"html_url"`
}

// RepoURL returns the base URL for repository (HTML, not API)
func (p *PRCallback) RepoURL() string {
	return p.PR.Head.Repository.HTMLURL
}

// Handle GitHub pull requests.
func handlePR(req *http.Request, blog *logging.Buildlog, c *config.Config) {
	b := parseBody(req)

	var pc PRCallback
	err := json.Unmarshal(b, &pc)

	if err != nil {
		log.Println(string(b))
		panic("Could not unmarshal request")
	}

	if pc.Action != "closed" {
		log.Println("handling pull request", pc.Number)
		go updatePR(pc, blog, c)
	}
}

// Updates the status of a pull request once the build is done. Sleeps 10
// seconds between the checks.
func updatePR(pc PRCallback, blog *logging.Buildlog, c *config.Config) {
	counter := 0 // used as pseudo rate limiting so GitHub likes us

	for {
		for _, j := range blog.Jobs {
			if j.Commit == pc.PR.Head.Commit {
				r, err := c.ConfigForRepo(j.URL)

				if err != nil {
					log.Println(err)
					return
				}

				if r.CommentPR {
					PostPR(c, j, pc)
				}

				if r.ClosePR {
					ClosePR(r.AccessKey, j, pc)
				}

				return
			}
		}

		// Check if the PR is still revelevant or if a new commit was pushed
		// or closed. Terminate the goroutine if this is the case.
		if counter >= 30 {
			if prIsCurrent(pc, c) == false {
				return
			}
			counter = 0
		} else {
			counter++
		}

		time.Sleep(10 * time.Second)
	}
}

// Returns if PRCallback is for the latest commit.
func prIsCurrent(pc PRCallback, c *config.Config) bool {
	cl := &http.Client{}
	r, err := http.NewRequest("GET", pc.PR.URL, nil)

	if err != nil {
		log.Println(err)
		return true
	}

	rp, err := c.ConfigForRepo(pc.RepoURL())

	if err != nil {
		log.Println(err)
		return true
	}

	t := base64.URLEncoding.EncodeToString([]byte(rp.AccessKey))

	r.Header.Add("content-type", "application/json")
	r.Header.Add("Authorization", "Basic "+t)

	resp, err := cl.Do(r)

	if err != nil {
		log.Println(err)
		return true
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var pr PRPullRequest
	err = json.Unmarshal(body, &pr)

	if err != nil {
		log.Println(err)
		return true
	}

	if pr.Head.Commit != pc.PR.Head.Commit {
		return false
	}

	if pr.State != "open" {
		return false
	}

	return true
}
