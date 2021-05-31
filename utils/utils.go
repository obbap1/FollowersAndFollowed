package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/joho/godotenv"
)

const (
	ComparisonChecker   = "following"
	FirstTestCase       = "_Obbap, BirthdayTracker following woonomic, ajmedwards"
	DummyTestCase       = "@obbap following following following @noliaaa"
	DummySecondTestCase = "@obbap, @kezi following"
	DummyThirdTestCase  = "following @noliaaa @omomo @mentus"
	SecondTestCase      = "@obbap following @noliaaa, @daveed_kz, @udori"
	ThirdTestCase       = "@obbap, @daveed_kz, @udori following @noliaaa"
)

type FollowersAndFollowed struct {
	LeftArray  []string
	RightArray []string
	AllArray   []string
}

func SetupTwitterClient() (*twitter.Client, *http.Client) {
	API_KEY, API_SECRET_KEY, ACCESS_TOKEN, ACCESS_TOKEN_SECRET := os.Getenv("API_KEY"), os.Getenv("API_SECRET_KEY"), os.Getenv("ACCESS_TOKEN"), os.Getenv("ACCESS_TOKEN_SECRET")
	config := oauth1.NewConfig(API_KEY, API_SECRET_KEY)
	token := oauth1.NewToken(ACCESS_TOKEN, ACCESS_TOKEN_SECRET)
	httpClient := config.Client(oauth1.NoContext, token)

	// Twitter client
	client := twitter.NewClient(httpClient)

	return client, httpClient
}

func FindFollowersAndFollowed(sentence string) (*FollowersAndFollowed, error) {
	re := regexp.MustCompile(ComparisonChecker)
	if followingArray := re.FindAll([]byte(sentence), -1); len(followingArray) != 1 {
		return nil, fmt.Errorf("The sentence isnt properly constructed, as there are %d followings instead of 1", len(followingArray))
	}

	followingLowerBoundIndex, followingUpperBoundIndex := re.FindStringIndex(sentence)[0], re.FindStringIndex(sentence)[1]

	leftContent, rightContent := sentence[0:followingLowerBoundIndex], sentence[followingUpperBoundIndex:]

	if len(leftContent) < 1 || len(rightContent) < 1 {
		return nil, fmt.Errorf("The sentence isnt properly constructed, Followers to check and People that are followed should be greater than one")
	}

	leftArray, rightArray := strings.Split(leftContent, ","), strings.Split(rightContent, ",")

	return &FollowersAndFollowed{
		LeftArray:  leftArray,
		RightArray: rightArray,
		AllArray:   append(leftArray, rightArray...),
	}, nil

}

func main() {

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	client, httpClient := SetupTwitterClient()
	values, err := FindFollowersAndFollowed(FirstTestCase)
	if err != nil {
		panic(err)
	}

	users, _, err := client.Users.Lookup(&twitter.UserLookupParams{ScreenName: values.AllArray})
	if err != nil {
		panic(err)
	}

	if len(users) != len(values.AllArray) {
		fmt.Printf("One or more of the usernames are invalid")
		return
	}

	var data map[string]interface{}

	MY_ID := os.Getenv("MY_ID")

	resp, err := httpClient.Get("https://api.twitter.com/2/users/" + MY_ID + "/mentions")

	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(body, &data)

	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v", data)

	for k, v := range data {
		fmt.Println("k", k, "v", v)
	}

	// followers, _, _ := client.Followers.List(&twitter.FollowerListParams{})

	// client.Re

	// fmt.Printf("Result %+v", followers)
	// FindFollowersAndFollowed(SecondTestCase)
	// FindFollowersAndFollowed(ThirdTestCase)
	// FindFollowersAndFollowed(DummyTestCase)
	// err := FindFollowersAndFollowed(DummySecondTestCase)
	// err = FindFollowersAndFollowed(DummyThirdTestCase)

	// fmt.Println(err)
}
