package main

// import libraries
import (
	"context"
	"encoding/json"
	"flag"
	f "fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	github "github.com/google/go-github/v33/github"
	"golang.org/x/oauth2"
)

//// STRUCTURES

// Queue primitive data structure
type Queue struct {
	list   []interface{}
	length int
}

func (q Queue) push(el interface{}) (qu Queue) {
	q.list = append(q.list, el)
	q.length++
	q.disp()
	return q
}
func (q Queue) pop() (qu Queue) {

	if q.length > 1 {
		q.list = q.list[1:]
	} else {
		q.list = make([]interface{}, 0)
	}
	q.length--
	q.disp()
	return q
}
func (q Queue) disp() {
	f.Printf("Queue: %v\n", q)
}

//// FUNCTIONS

// SetupCloseHandler - handle clean up after exit
func SetupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// new thread for cleanup
	go func() {
		<-c
		f.Println("\r- Ctrl+C pressed in Terminal")
		// remove temp directory
		os.RemoveAll("./tmp/")
		f.Println("Exited")
		os.Exit(0)
	}()
}

// goCloneGitRepo - Clone GitHub Repo in tmp/ directory
func goCloneGitRepo(tokenGit string, repo *github.Repository) {
	f.Printf("\n\n%v\n\n", *repo.CloneURL)
	cloneurl := *repo.CloneURL
	gitCommand := cloneurl[:8] + tokenGit + "@" + cloneurl[8:]
	cmd := exec.Command("git", "clone", gitCommand)
	cmd.Run()
	f.Println("Git: Clonned ", cloneurl)
}

// goGitLogin - log in to git and return client
func goGitLogin(token string) (user *github.Client, contx context.Context) {
	/*
		Args:
			token: git token
		Returns:
			user: github client
			contx: context
	*/
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	// authenticate new client using context and token
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc), ctx
}

// goGitCheckout - checkout clonned repo to target branch
func goGitCheckout(branch string, project string) {

	// move to git directory
	os.Chdir("tmp/" + project)
	f.Println("Moved in ", "tmp/"+project, "directory")

	// execute command to checkout on branch
	cmd := exec.Command("git", "checkout", branch)
	cmd.Run()
	f.Println("Git: check out on branch", branch)

	// move back to work directory
	os.Chdir("../../")
	f.Println("Moved back to work directory")
}

// ?? goGetGitRepo - get github repo we need as variable
func goGetGitRepo(ctx context.Context, gitClient *github.Client, targetRepo string) (rep github.Repository) {
	// get repository we are looking for
	repos, _, err := gitClient.Repositories.List(ctx, "", nil)
	if err != nil {
		f.Println("Could not authenticate git")
		return
	}
	var repo github.Repository
	for i := 0; i < len(repos); i++ {
		f.Println(*repos[i].Name)
		if *repos[i].Name == targetRepo {
			repo = *repos[i]
		}
	}
	return repo
}

func runPulumiPlan(project string) {
	os.Chdir("tmp/" + project)
}

var queue Queue

// pull request structure
type GitPL struct {
	project string
	branch  string // branch name -
	action  string // action done: comment pull request close
	ID      int    // ID of pull request -> pull_request.number
}

func commentOnReview() {

}

func handleQueue(branch string, project string) {
	if queue.length == 0 {
		goGitCheckout(branch, project)
		runPulumiPlan(project)
		return
	}
	f.Println("NOPE")
	queue = queue.push(branch)
}

// referrence - https://groob.io/tutorial/go-github-webhook/
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	// f.Printf("\n\nreceived %v\n\n", m)
	f.Println("HookHandler: Data received")
	// f.Printf("headers: %v\n", r.Header)
	f.Printf("git event: %v\n", r.Header.Get("X-Github-Event"))
	webhookData := make(map[string]interface{})

	err := json.NewDecoder(r.Body).Decode(&webhookData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var gitProjectName string = webhookData["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["repo"].(map[string]interface{})["name"].(string)
	var gitBranchName string = webhookData["pull_request"].(map[string]interface{})["head"].(map[string]interface{})["ref"].(string)

	handleQueue(gitBranchName, gitProjectName)

	f.Println(gitBranchName)
}

func main() {

	os.RemoveAll("./tmp/")

	f.Println("Visit https://github.com/levankhelo/pullantis/blob/master/README.md for setup information")

	// Arguments
	var targetRepo = flag.String("repo", "pullantis", "GitHub repository name")
	var userGit = flag.String("git-user", "levankhelo", "GitHub Token")
	var tokenGit = flag.String("git-token", "", "GitHub Token")
	var tokenPulumi = flag.String("pulumi-token", "", "Pulumi Token")
	var webhookGit = flag.String("webhook", "/events", "GitHub webhook tag")
	var localPort = flag.String("port", "4141", "local port for listener")
	var noServer = flag.Bool("no-server", false, "allow server listener")

	// Parse Arguments
	flag.Parse()

	// display arg values
	f.Println("----GIT----\n|\t", "\n|\tUser:", *userGit, "\n|\tToken:", *tokenGit, "\n|\tRepo:", *targetRepo)
	f.Println("---PULUMI--\n|\t", "\n|\tToken:", *tokenPulumi)
	f.Println("--WEBHOOK--\n|\t", "Hook:", *webhookGit, "\n|\tPort:", *localPort)

	// init git user using Access Token
	gitClient, ctx := goGitLogin(*tokenGit)

	// get target reository
	repo := goGetGitRepo(ctx, gitClient, *targetRepo)
	f.Printf("\n|%v\n\n\n", repo)
	SetupCloseHandler()

	os.Mkdir("tmp/", 0700)
	os.Chdir("tmp/")
	goCloneGitRepo(*tokenGit, &repo)
	os.Chdir("../")

	if *noServer != true {
		log.Println("server started")
		http.HandleFunc(*webhookGit, handleWebhook)
		log.Fatal(http.ListenAndServe(":"+(*localPort), nil))
	} else {
		log.Println("building without server")
		os.RemoveAll("./tmp/")
	}

}
