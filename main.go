package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

var (
	tcp  = flag.String("tcp", "127.0.0.1:8081", "serve as FCGI via TCP, example: 0.0.0.0:8000")
	unix = flag.String("unix", "", "serve as FCGI via UNIX socket, example: /tmp/myprogram.sock")
	from string
	to   string
)

func addBasicAuth(URL string, req *http.Request) (string, error) {
	req.ParseForm()
	user, pass, authok := req.BasicAuth()

	if authok {
		parsedURL, err := url.Parse(URL)

		if err != nil {
			return "", err
		}

		parsedURL.User = url.UserPassword(user, pass)

		return parsedURL.String(), nil
	}

	return URL, nil
}

func rewriteURL(req *http.Request) (string, error) {
	newURL := to + strings.TrimPrefix(strings.TrimPrefix(req.URL.Path, "/"), from) + "?" + req.URL.RawQuery

	newURL, _ = addBasicAuth(newURL, req)

	return newURL, nil
}

func serve(w http.ResponseWriter, req *http.Request) {
	newURL, _ := rewriteURL(req)

	log.Println(req.URL.String() + " -> " + newURL)
	resp, err := http.Get(newURL)
	if err != nil {
		log.Println(newURL, err)
		w.WriteHeader(http.StatusInternalServerError)

		return
	}

	log.Println(newURL, resp.Status)

	repositoryPath := fmt.Sprintf("cache/%x", sha256.Sum256([]byte(req.URL.Path)))
	repositoryURL := strings.Split(newURL, "/info/refs")[0]
	log.Printf("%s -> %x\n", req.URL.Path, repositoryPath)

	var cmd *exec.Cmd
	os.MkdirAll(repositoryPath, 0775)
	cmd = exec.Command("git", "init")
	cmd.Dir = repositoryPath
	err = cmd.Run()
	cmd = exec.Command("git", "remote", "add", "origin", repositoryURL)
	cmd.Dir = repositoryPath
	err = cmd.Run()
	cmd = exec.Command("git", "fetch", "--all")
	cmd.Dir = repositoryPath
	err = cmd.Run()

	headers := w.Header()
	for key, value := range resp.Header {
		headers.Add(key, value[0])
	}
	headers.Add("Content-Length", "0")
	w.WriteHeader(resp.StatusCode)

	io.WriteString(w, "")
}

func main() {
	log.Println("Starting server...")

	flag.Parse()

	from = flag.Arg(0)
	to = flag.Arg(1)

	var (
		listener net.Listener
		err      error
	)
	if *unix != "" {
		listener, err = net.Listen("unix", *unix)
	} else {
		listener, err = net.Listen("tcp", *tcp)
	}
	if err != nil {
		log.Panicln(err)
	}
	defer listener.Close()

	log.Println("Listening on", listener.Addr())

	mux := http.NewServeMux()
	mux.HandleFunc("/", serve)

	if err := fcgi.Serve(listener, mux); err != nil {
		log.Panicln(err)
	}
}
