package main

// import libraries
import (

	// njson "github.com/m7shapan/njson"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	f "fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/storage/memory"
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

func SetupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		os.RemoveAll("./tmp/")
		f.Println("Exited")
		os.Exit(0)
	}()
}

func main() {

	f.Println("Visit https://github.com/levankhelo/pullantis/blob/master/README.md for setup information")

	// Arguments
	var targetRepo = flag.String("repo", "pullantis", "GitHub repository name")
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

	gitClient := github.NewClient(tc)

	// list all repositories for the authenticated user
	repos, _, err := gitClient.Repositories.List(ctx, "", nil)
	if err != nil {
		f.Println("Could not authenticate git")
		return
	}
	var repo github.Repository
	for i := 0; i < len(repos); i++ {
		f.Println(*repos[i].Name)
		if *repos[i].Name == *targetRepo {
			repo = *repos[i]
		}
	}
	// f.Printf("%v\n", repo)

	SetupCloseHandler()

	os.Mkdir("tmp/", 0700)

	os.Chdir("tmp/")

	newRepo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: *repo.HTMLURL,
	})

	f.Printf("NEW REPO: %v\n", *newRepo)

	// newRepo.Pull(&git.PullOptions{
	// 	RemoteName: "origin",
	// })

	os.Chdir("../")

	if *allowServer {
		log.Println("server started")
		http.HandleFunc(*webhookGit, handleWebhook)
		log.Fatal(http.ListenAndServe(":"+(*localPort), nil))
	}

}
