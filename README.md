<hr> 

# pullantis

## Install pulumi
install `pulumi` using package manager 
- linux: `apt-get install pulumi` 
- Mac OS: `brew install pulumi` 

Now we need to create directory and 

## Start ngrok
install `ngrok` using package manager 
- linux: `apt-get install ngrok` 
- Mac OS: `brew install ngrok` 
```
ngrok http 4141
```
**save your <mark>ngrok's forwarded address</mark>**

<br> 
<hr> 
<br> 


## Setup GitHub WebHook
Go to your [GitHub](https://www.github.com) repo and move in `settings > webhooks`  
Click `Add Webhook`   
 
Payload URL:  `https://MYADRESS.ngrok.io/events` where `MYADRESS` is your Ngrok's adress forwarding

Content Type: `application/json`  

Secret: Any Secret text

Event Triggers:  `Let me select individual events.`  

- `Commit comments` 
- `Issue comments`
- `Pull request reviews`
- `Pushes` 
  
Click `Add Webhook` and that's it!


 ## Retrieve Pulumi Access Token

 Visit [Pulumi](https://app.pulumi.com/) and go in `user` > `settings` > `Access Tokens`  
 Click `NEW ACCESS TOKEN` and input `pullantis` in description! 
 Now we have <mark>pulumi_token</mark>



<hr> 
<br> 
<br> 
