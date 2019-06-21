package oauth2ns

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/fatih/color"
	rndm "github.com/nmrshll/rndm-go"
	"github.com/palantir/stacktrace"
	"github.com/skratchdot/open-golang/open"
	"golang.org/x/oauth2"
)

type AuthorizedClient struct {
	*http.Client
	Token *oauth2.Token
}

const (
	// IP is the ip of this machine that will be called back in the browser. It may not be a hostname.
	// If IP is not 127.0.0.1 DEVICE_NAME must be set. It can be any short string.
	IP          = "127.0.0.1"
	DEVICE_NAME = ""
	// PORT is the port that the temporary oauth server will listen on
	PORT = 14565
	// seconds to wait before giving up on auth and exiting
	authTimeout                = 120
	oauthStateStringContextKey = 987
)

type AuthenticateUserOption func(*AuthenticateUserFuncConfig) error
type AuthenticateUserFuncConfig struct {
	AuthCallHTTPParams url.Values
}

func WithAuthCallHTTPParams(values url.Values) AuthenticateUserOption {
	return func(conf *AuthenticateUserFuncConfig) error {
		conf.AuthCallHTTPParams = values
		return nil
	}
}

// AuthenticateUser starts the login process
func AuthenticateUser(oauthConfig *oauth2.Config, options ...AuthenticateUserOption) (*AuthorizedClient, error) {
	// validate params
	if oauthConfig == nil {
		return nil, stacktrace.NewError("oauthConfig can't be nil")
	}
	// read options
	var optionsConfig AuthenticateUserFuncConfig
	for _, processConfigFunc := range options {
		processConfigFunc(&optionsConfig)
	}

	// add transport for self-signed certificate to context
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	sslcli := &http.Client{Transport: tr}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, sslcli)

	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	oauthConfig.RedirectURL = fmt.Sprintf("http://%s:%s/oauth/callback", IP, strconv.Itoa(PORT))

	// Some random string, random for each request
	oauthStateString := rndm.String(8)
	ctx = context.WithValue(ctx, oauthStateStringContextKey, oauthStateString)
	urlString := oauthConfig.AuthCodeURL(oauthStateString, oauth2.AccessTypeOffline)

	if optionsConfig.AuthCallHTTPParams != nil {
		parsedURL, err := url.Parse(urlString)
		if err != nil {
			return nil, stacktrace.Propagate(err, "failed parsing url string")
		}
		params := parsedURL.Query()
		for key, value := range optionsConfig.AuthCallHTTPParams {
			params[key] = value
		}
		parsedURL.RawQuery = params.Encode()
		urlString = parsedURL.String()
	}

	if IP != "127.0.0.1" {
		urlString = fmt.Sprintf("%s&device_id=%s&device_name=%s", urlString, DEVICE_NAME, DEVICE_NAME)
	}

	clientChan, stopHTTPServerChan, cancelAuthentication := startHTTPServer(ctx, oauthConfig)
	log.Println(color.CyanString("You will now be taken to your browser for authentication or open the url below in a browser."))
	log.Println(color.CyanString(urlString))
	log.Println(color.CyanString("If you are opening the url manually on a different machine you will need to curl the result url on this machine manually."))
	time.Sleep(1000 * time.Millisecond)
	err := open.Run(urlString)
	if err != nil {
		log.Println(color.RedString("Failed to open browser, you MUST do the manual process."))
	}
	time.Sleep(600 * time.Millisecond)

	// shutdown the server after timeout
	go func() {
		log.Printf("Authentication will be cancelled in %s seconds", strconv.Itoa(authTimeout))
		time.Sleep(authTimeout * time.Second)
		stopHTTPServerChan <- struct{}{}
	}()

	select {
	// wait for client on clientChan
	case client := <-clientChan:
		// After the callbackHandler returns a client, it's time to shutdown the server gracefully
		stopHTTPServerChan <- struct{}{}
		return client, nil

		// if authentication process is cancelled first return an error
	case <-cancelAuthentication:
		return nil, fmt.Errorf("authentication timed out and was cancelled")
	}
}

func startHTTPServer(ctx context.Context, conf *oauth2.Config) (clientChan chan *AuthorizedClient, stopHTTPServerChan chan struct{}, cancelAuthentication chan struct{}) {
	// init returns
	clientChan = make(chan *AuthorizedClient)
	stopHTTPServerChan = make(chan struct{})
	cancelAuthentication = make(chan struct{})

	http.HandleFunc("/oauth/callback", callbackHandler(ctx, conf, clientChan))
	srv := &http.Server{Addr: ":" + strconv.Itoa(PORT)}

	// handle server shutdown signal
	go func() {
		// wait for signal on stopHTTPServerChan
		<-stopHTTPServerChan
		log.Println("Shutting down server...")

		// give it 5 sec to shutdown gracefully, else quit program
		d := time.Now().Add(5 * time.Second)
		ctx, cancel := context.WithDeadline(context.Background(), d)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf(color.RedString("Auth server could not shutdown gracefully: %v"), err)
		}

		// after server is shutdown, quit program
		cancelAuthentication <- struct{}{}
	}()

	// handle callback request
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
		fmt.Println("Server gracefully stopped")
	}()

	return clientChan, stopHTTPServerChan, cancelAuthentication
}

func callbackHandler(ctx context.Context, oauthConfig *oauth2.Config, clientChan chan *AuthorizedClient) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestStateString := ctx.Value(oauthStateStringContextKey).(string)
		responseStateString := r.FormValue("state")
		if responseStateString != requestStateString {
			fmt.Printf("invalid oauth state, expected '%s', got '%s'\n", requestStateString, responseStateString)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}

		code := r.FormValue("code")
		token, err := oauthConfig.Exchange(ctx, code)
		if err != nil {
			fmt.Printf("oauthoauthConfig.Exchange() failed with '%s'\n", err)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		}
		// The HTTP Client returned by oauthConfig.Client will refresh the token as necessary
		client := &AuthorizedClient{
			oauthConfig.Client(ctx, token),
			token,
		}
		// show success page
		successPage := `
		<div style="height:100px; width:100%!; display:flex; flex-direction: column; justify-content: center; align-items:center; background-color:#2ecc71; color:white; font-size:22"><div>Success!</div></div>
		<p style="margin-top:20px; font-size:18; text-align:center">You are authenticated, you can now return to the program. This will auto-close</p>
		<script>window.onload=function(){setTimeout(this.close, 4000)}</script>
		`
		fmt.Fprintf(w, successPage)
		// quitSignalChan <- quitSignal
		clientChan <- client
	}
}
