package main

import (
	"flag"
	"leeroy/build"
	"leeroy/config"
	"leeroy/integrations"
	"leeroy/logging"
	"leeroy/web"
	"log"
	"net/http"
)

var cfgFlag = flag.String("config", "leeroy.json", "JSON formatted config")
var createUser = flag.Bool("createUser", false, "create a new user")
var updateUser = flag.Bool("updateUser", false, "update user information")
var deleteUser = flag.Bool("deleteUser", false, "delete a user")
var listUsers = flag.Bool("listUsers", false, "list all users")

func main() {
	flag.Parse()

	c := config.FromFile(*cfgFlag)

	err := c.Validate()

	if err != nil {
		log.Fatal("Configuration error: ", err)
	}

	if *createUser == true {
		c.CreateUserCMD()
		return
	}

	if *updateUser == true {
		c.UpdateUserCMD()
		return
	}

	if *deleteUser == true {
		c.DeleteUserCMD()
		return
	}

	if *listUsers == true {
		c.ListUserCMD()
		return
	}

	jobs := make(chan logging.Job, 100)
	b := logging.New(c.BuildLogPath)

	go build.Build(jobs, c, b)

	log.Println("leeroy up an running!")

	http.HandleFunc("/callback/", web.Auth(func(w http.ResponseWriter, r *http.Request) {
		integrations.Callback(w, r, jobs, c, b)
	}, c.Secret))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		web.Status(w, r, c, b)
	})
	http.HandleFunc("/status/repo/", func(w http.ResponseWriter, r *http.Request) {
		web.Repo(w, r, c, b)
	})
	http.HandleFunc("/status/branch/", func(w http.ResponseWriter, r *http.Request) {
		web.Branch(w, r, c, b)
	})
	http.HandleFunc("/status/commit/", func(w http.ResponseWriter, r *http.Request) {
		web.Commit(w, r, c, b)
	})
	http.HandleFunc("/status/badge/", func(w http.ResponseWriter, r *http.Request) {
		web.Badge(w, r, c, b)
	})

	if c.Scheme() == "https" {
		log.Println("HTTPS:", c.URL)
		log.Fatal(http.ListenAndServeTLS(c.Host(), c.Cert, c.Key, nil))
	} else {
		log.Println("HTTP:", c.URL)
		log.Fatal(http.ListenAndServe(c.Host(), nil))
	}
}
