<hr> 

# pullantis


## Start ngrok
install `ngrok` from package manager 
- linux: `apt-get install ngrok` 
- Mac: `brew install ngrok` 
```
ngrok http 4141
```
**save your ngrok's forwarded address**

<br> 
<hr> 
<br> 


## Setup GitHub WebHook
Go to your [GitHub](https://www.github.com) repo and move in `settings > webhooks`  
Click `Add Webhook`   
 
Payload URL:  `https://MYADRESS.ngrok.io/events` where `MYADRESS` is your Ngrok's adress forwarding

Content Type: `application/json`  

Event Triggers:  `Let me select individual events.`  

- `Commit comments` 
- `Issue comments`
- `Pull request reviews`
- `Pushes` 
  
 
 ## Retrieve Pulumi Access Token

 go 


<hr> 
<br> 
<br> 
