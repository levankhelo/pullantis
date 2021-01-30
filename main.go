package main

// import libraries
import (
	"encoding/json"
	flag "flag"
	f "fmt"
	"log"
	"net/http"
)

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	f.Println("HELLO")
	webhookData := make(map[string]interface{})

	err := json.NewDecoder(r.Body).Decode(&webhookData)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	f.Println("got webhook payload: ")
	for k, v := range webhookData {
		f.Printf("%s : %v\n", k, v)
	}
}

func main() {

	f.Println("Visit https://github.com/levankhelo/pullantis/blob/master/README.md for setup information")

	// Arguments
	var userGit = flag.String("git-user", "", "GitHub username")
	var tokenGit = flag.String("git-token", "", "GitHub Token")
	var userPulumi = flag.String("pulumi-user", "", "Pulumi username")
	var tokenPulumi = flag.String("pulumi-token", "", "Pulumi Token")
	var webhookGit = flag.String("webhook", "/events", "GitHub webhook tag")
	var localPort = flag.String("port", "4141", "local port for listener")

	// Parse Arguments
	flag.Parse()

	f.Println("----GIT----\n", "user: ", *userGit, "\nToken: ", *tokenGit)
	f.Println("---PULUMI--\n", "user: ", *userPulumi, "\nToken: ", *tokenPulumi)
	f.Println("--WEBHOOK--\n", "hook: ", *webhookGit, "\nport: ", *localPort)
	f.Println("\n---\n---\n")
	f.Printf("%#v\n", *localPort)

	http.HandleFunc(*webhookGit, handleWebhook)
	log.Fatal(http.ListenAndServe(*localPort, nil))

}
