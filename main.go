package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	oauth2v2google "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

var googleConfig *oauth2.Config

func generateOauthState(w http.ResponseWriter) string {
	var expiration = time.Now().Add(365 * 24 * time.Hour)
	b := make([]byte, 30)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	cookie := http.Cookie{Name: "oauthstate", Value: state, Expires: expiration}
	http.SetCookie(w, &cookie)
	return state
}
func init() {
	//Replace with your client id .json file
	b, err := ioutil.ReadFile("client_secret_601894538001-pfl2vfrksh0ut7v98jcni124nfoir209.apps.googleusercontent.com.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}
	//Specify the scopes
	config, err := google.ConfigFromJSON(b, oauth2v2google.OpenIDScope, oauth2v2google.UserinfoProfileScope, oauth2v2google.UserinfoEmailScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	googleConfig = config
}

func main() {
	e := echo.New()
	//serve demo html
	e.GET("/", func(c echo.Context) error {
		return c.File("demo.html")
	})
	e.GET("/signin", func(c echo.Context) error {

		//generate random oauth2 state then put the newly generated into response cookie for CSRF protection
		oauthState := generateOauthState(c.Response().Writer)
		authURL := googleConfig.AuthCodeURL(oauthState, oauth2.AccessTypeOffline)
		return c.Redirect(http.StatusTemporaryRedirect, authURL)
	})

	e.GET("/signin/callback", func(c echo.Context) error {
		code := c.QueryParam("code")
		state := c.QueryParam("state")
		oauthState, _ := c.Request().Cookie("oauthstate")
		//Check if authorization code is present and the given state param equal to oauthStateCookie
		if code == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Code is not present")
		}
		if state != oauthState.Value {
			return echo.NewHTTPError(http.StatusBadRequest, "Missmatch state")
		}
		//Get token from google
		token, err := googleConfig.Exchange(context.TODO(), code)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Unable to retrieve token from google ")
		}
		//You can save the token for further use

		//To be able to access google api (eg. oauth2 api, email api, people api, etc) we need to create the http client
		//Create http client with token
		httpClient := googleConfig.Client(context.Background(), token)

		//Instantiate the desired service (eg oauth2 api service for accessing user profile and user email)
		oauthService, err := oauth2v2google.NewService(context.Background(), option.WithHTTPClient(httpClient))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Unable to instantiate service")
		}
		// Access the desired resource with the newly created service
		userProfile, err := oauthService.Userinfo.V2.Me.Get().Do()
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Unable to retrieve resource from service")
		}
		return c.JSON(200, userProfile)
	})
	log.Fatal(e.Start(":6969"))
}
