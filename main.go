package main

// import libraries
import (

	// njson "github.com/m7shapan/njson"
	"encoding/json"
	flag "flag"
	f "fmt"
	"log"
	"net/http"
	"net/http/httputil"
)

type webhook struct {
	Action      string
	PullRequest struct {
		ref string
	}
}

func getAction(w http.ResponseWriter) {

}

// webhook handler
// referrenced to https://groob.io/tutorial/go-github-webhook/
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	f.Println("HookHandler: Data received")
	f.Printf("headers: %v\n", r.Header)
	f.Printf("git event: %v\n", r.Header.Get("X-Github-Event"))
	webhookData := make(map[string]interface{})
	// var webhookData webhook

	err := json.NewDecoder(r.Body).Decode(&webhookData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		f.Println(err)
	}
	f.Println(string(requestDump))

	// f.Fprintf(w, "webhook: %+v", webhookData)
	// switch action := webhookData.Action; action {
	// case "opened":
	// 	f.Print(webhookData.PullRequest.ref)
	// default:
	// 	f.Print("closed")
	// }

	// for v, k := range webhookData {
	// 	if v == "pull_request" {
	// 		// f.Print(k)
	// 		// f.Printf("%T", k)
	// 	}
	// 	// f.Println("\n\n", v, k)
	// }
}

func argPassed(arg string) bool {
	var received bool = false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == arg {
			received = true
		}
	})
	return received
}

func main() {

	f.Println("Visit https://github.com/levankhelo/pullantis/blob/master/README.md for setup information")

	// Arguments
	var targetRepo = flag.String("repo", "", "GitHub repository name")
	var userGit = flag.String("git-user", "", "GitHub Token")
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

	if !argPassed(*tokenGit) {
		f.Print("COOL")
	}

	if *allowServer {
		log.Println("server started")
		http.HandleFunc(*webhookGit, handleWebhook)
		log.Fatal(http.ListenAndServe(":"+(*localPort), nil))
	}

}
