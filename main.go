package main

// import libraries
import (

	// njson "github.com/m7shapan/njson"
	"encoding/json"
	flag "flag"
	f "fmt"
	"log"
	"net/http"
)

type webhook struct {
	Action      string
	PullRequest struct {
		ref string
	}
}

// webhook handler
// referrenced to https://groob.io/tutorial/go-github-webhook/
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	f.Println("HookHandler: Data received")
	// fmt.Printf("headers: %v\n", r.Header)
	// webhookData := make(map[string]interface{})
	var webhookData webhook

	err := json.NewDecoder(r.Body).Decode(&webhookData)
	if err != nil {
		f.Print("Error")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f.Fprintf(w, "webhook: %+v", webhookData)
	switch action := webhookData.Action; action {
	case "opened":
		f.Print(webhookData.PullRequest.ref)
	default:
		f.Print("closed")
	}

	// for k := range webhookData {
	// f.Println("\n\n", k)
	// f.Printf("\n----------------------------\n\n\n%s : %v\n\n\n----------------------------\n\n\n", k, v)
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
