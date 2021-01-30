package main

// import libraries
import (

	// njson "github.com/m7shapan/njson"
	"context"
	"encoding/json"
	flag "flag"
	f "fmt"
	"log"
	"net/http"

	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

// webhook handler
// referrenced to https://groob.io/tutorial/go-github-webhook/
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	f.Println("HookHandler: Data received")
	// f.Printf("headers: %v\n", r.Header)
	// f.Printf("git event: %v\n", r.Header.Get("X-Github-Event"))
	webhookData := make(map[string]interface{})
	// var webhookData webhook

	err := json.NewDecoder(r.Body).Decode(&webhookData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var gitBranchName string = webhookData["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["ref"].(string)

	f.Println(gitBranchName)

}

func main() {

	f.Println("Visit https://github.com/levankhelo/pullantis/blob/master/README.md for setup information")

	// Arguments
	var targetRepo = flag.String("repo", "", "GitHub repository name")
	var userGit = flag.String("git-user", "levankhelo", "GitHub Token")
	var tokenGit = flag.String("git-token", "", "GitHub Token")
	var tokenPulumi = flag.String("pulumi-token", "", "Pulumi Token")
	var webhookGit = flag.String("webhook", "/events", "GitHub webhook tag")
	var localPort = flag.String("port", "4141", "local port for listener")
	var allowServer = flag.Bool("server", true, "allow server listener")

	// Parse Arguments
	flag.Parse()

	f.Println("----GIT----\n|\t", "\n|\tUser:", *userGit, "\n|\tToken:", *tokenGit, "\n|\tRepo:", *targetRepo)
	f.Println("---PULUMI--\n|\t", "\n|\tToken:", *tokenPulumi)
	f.Println("--WEBHOOK--\n|\t", "Hook:", *webhookGit, "\n|\tPort:", *localPort)

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *tokenGit},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// list all repositories for the authenticated user
	repos, _, err := client.Repositories.List(ctx, "", nil)
	if err != nil {
		f.Println("Could not authenticate git")
		return
	}
	f.Printf("%v", repos)

	if *allowServer {
		log.Println("server started")
		http.HandleFunc(*webhookGit, handleWebhook)
		log.Fatal(http.ListenAndServe(":"+(*localPort), nil))
	}

}
