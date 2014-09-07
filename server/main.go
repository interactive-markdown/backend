package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/fsouza/go-dockerclient"
)

const endpoint = "unix://var/run/docker.sock"

var (
	langs = map[string]string{
		"python": "python2",
		"sample": "sample",
	}

	c *docker.Client
)

func imgNameFromLang(lang string) string {
	return fmt.Sprintf("interactivemarkdown/%s", lang)
}

type Session struct {
	Lang string `json:"language"`
	Code string `json:"code"`
}

func main() {
	var err error
	c, err = docker.NewClient(endpoint)
	if err != nil {
		panic(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	http.HandleFunc("/sessions", newSession)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

func newSession(w http.ResponseWriter, r *http.Request) {
	var session Session

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
		panic(err)
	}

	lang := langs[session.Lang]
	img := imgNameFromLang(lang)

	codeTmpFilename := fmt.Sprintf("/tmp/mkdn/%x", md5.Sum([]byte(session.Code)))

	file, err := os.Create(codeTmpFilename)
	if err != nil {
		panic(err)
	}

	file.Write([]byte(session.Code))

	container, err := c.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: img,
			Cmd:   []string{codeTmpFilename},
		},
	})
	if err != nil {
		panic(err)
	}

	err = c.StartContainer(container.ID, &docker.HostConfig{
		Binds: []string{"/tmp/mkdn:/tmp/mkdn"},
	})
	if err != nil {
		panic(err)
	}

	err = c.Logs(docker.LogsOptions{
		Container:    container.ID,
		OutputStream: w,
		ErrorStream:  w,
		Stdout:       true,
		Stderr:       true,
		Follow:       true,
	})
	if err != nil {
		panic(err)
	}
}
