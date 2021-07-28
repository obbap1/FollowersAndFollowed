package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/joho/godotenv"
)

const (
	ComparisonChecker = "following"
	MyUserName        = "BirthdayTracker"
)

type FollowersAndFollowed struct {
	LeftArray  []string
	RightArray []string
	AllArray   []string
}

var (
	c              *twitter.Client
	h              *http.Client
	twitterBaseUrl = "https://api.twitter.com/2/users/"
)

func setupArray(allArray, arr *[]string) *[]string {
	for k := range *arr {
		(*arr)[k] = strings.TrimSpace(strings.Replace((*arr)[k], "@", "", -1))
		contains := false
		for _, v := range *allArray {
			if v == (*arr)[k] {
				contains = true
				break
			}
		}

		if !contains && (*arr)[k] != MyUserName {
			*allArray = append(*allArray, strings.TrimSpace((*arr)[k]))
		}

	}

	return allArray
}

func AttachAt(s string) string {
	return "@" + s
}

func FormatSuccessMessage(follower, user string) string {
	return "\n" + "Yes. " + AttachAt(follower) + " is following " + AttachAt(user)
}

func FormatFailureMessage(follower, user string) string {
	return "\n" + "No. " + AttachAt(follower) + " is NOT following " + AttachAt(user)
}

// Sets up the twitter client
func SetupTwitterClient() (*twitter.Client, *http.Client) {
	if c != nil && h != nil {
		return c, h
	}
	API_KEY, API_SECRET_KEY, ACCESS_TOKEN, ACCESS_TOKEN_SECRET := os.Getenv("API_KEY"), os.Getenv("API_SECRET_KEY"), os.Getenv("ACCESS_TOKEN"), os.Getenv("ACCESS_TOKEN_SECRET")
	config := oauth1.NewConfig(API_KEY, API_SECRET_KEY)
	token := oauth1.NewToken(ACCESS_TOKEN, ACCESS_TOKEN_SECRET)
	httpClient := config.Client(oauth1.NoContext, token)

	// Twitter client
	client := twitter.NewClient(httpClient)

	c = client
	h = httpClient

	return client, httpClient
}

// Finds followers and followed from the sentence
func FindFollowersAndFollowed(sentence string) (*FollowersAndFollowed, error) {
	re := regexp.MustCompile(ComparisonChecker)
	if followingArray := re.FindAll([]byte(sentence), -1); len(followingArray) != 1 {
		return nil, fmt.Errorf("the sentence isnt properly constructed, as there are %d followings instead of 1", len(followingArray))
	}

	followingLowerBoundIndex, followingUpperBoundIndex := re.FindStringIndex(sentence)[0], re.FindStringIndex(sentence)[1]

	leftContent, rightContent := sentence[0:followingLowerBoundIndex], sentence[followingUpperBoundIndex:]

	if len(leftContent) < 1 || len(rightContent) < 1 {
		return nil, fmt.Errorf("the sentence isnt properly constructed, Followers to check and People that are followed should be greater than one")
	}

	split := func(r rune) bool {
		return r == ',' || r == ' '
	}

	leftArray, rightArray := strings.FieldsFunc(leftContent, split), strings.FieldsFunc(rightContent, split)

	cleanLeftArray, cleanRightArray := make([]string, 0, len(leftArray)), make([]string, 0, len(rightArray))

	for _, c := range leftArray {
		if strings.TrimSpace(c) != AttachAt(MyUserName) {
			cleanLeftArray = append(cleanLeftArray, c)
		}
	}

	for _, c := range rightArray {
		if strings.TrimSpace(c) != AttachAt(MyUserName) {
			cleanRightArray = append(cleanRightArray, c)
		}
	}

	allArray := make([]string, 0, len(cleanLeftArray)+len(cleanRightArray))

	allArray = *setupArray(&allArray, &cleanLeftArray)
	allArray = *setupArray(&allArray, &cleanRightArray)

	fmt.Println(allArray)

	if len(allArray) == 0 {
		return nil, fmt.Errorf("no users to seach for")
	}

	return &FollowersAndFollowed{
		LeftArray:  cleanLeftArray,
		RightArray: cleanRightArray,
		AllArray:   allArray,
	}, nil

}

func FetchMentions() error {
	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	client, httpClient := SetupTwitterClient()

	data := make(map[string]interface{})

	MY_ID := os.Getenv("MY_ID")

	resp, err := httpClient.Get(twitterBaseUrl + MY_ID + "/mentions")

	if err != nil {
		return fmt.Errorf("an error occured while fetching the user's details: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return fmt.Errorf("an error occured while fetching the request body: %s", err)
	}

	err = json.Unmarshal(body, &data)

	if err != nil {
		return fmt.Errorf("an error occured while unmarshalling the request body: %s", err)
	}

	content := data["data"].([]interface{})

	fmt.Printf("%+v", content)

	mentionsChan := make(chan string, len(content))

	for _, v := range content {
		vi := v.(map[string]interface{})

		text := vi["text"].(string)

		id, err := strconv.ParseInt(vi["id"].(string), 10, 64)

		if err != nil {
			return fmt.Errorf("error while parsing int64: %s", err)
		}

		go func(s string, id int64) {
			tweet, err := FetchResults(s)
			if err != nil {
				mentionsChan <- fmt.Sprintf("Error is: %s", err)
				return
			}
			// TODO: ratelimit here as well

			// send tweet
			tweetSent, r, err := client.Statuses.Update(tweet, &twitter.StatusUpdateParams{InReplyToStatusID: id})

			fmt.Printf("%+v", r)

			if err != nil {
				mentionsChan <- fmt.Sprintf("Error is: %s", err)
				return
			}
			// send signal
			mentionsChan <- fmt.Sprintf("\n done with processing: %+v, %s", tweetSent, tweet)

		}(text, id)

	}

	for range content {
		fmt.Println(<-mentionsChan)
	}

	return nil

}

func FetchResults(sentence string) (string, error) {
	values, err := FindFollowersAndFollowed(sentence)
	if err != nil {
		return "", fmt.Errorf("an error occured while formatting the sentence: %s", err)
	}

	client, _ := SetupTwitterClient()

	users, _, err := client.Users.Lookup(&twitter.UserLookupParams{ScreenName: values.AllArray})
	if err != nil {
		return "", fmt.Errorf("an error occured while searching for the users on twitter: %s", err)
	}

	if len(users) != len(values.AllArray) {
		return "", fmt.Errorf("%d out of %d of the usernames are invalid", len(values.AllArray)-len(users), len(values.AllArray))
	}

	response := ""

	for _, user := range values.LeftArray {
		// TODO: check if followers / following count is lesser and use the lesser array in size
		limit, _, err := client.RateLimits.Status(&twitter.RateLimitParams{Resources: []string{"friends"}})

		rateLimit := limit.Resources.Friends["/friends/list"]

		fmt.Printf("%d of %d requests are left", rateLimit.Remaining, rateLimit.Limit)

		if rateLimit.Remaining == 0 {
			duration := int64(rateLimit.Reset) - time.Now().Unix()
			fmt.Printf("Sleeping for %d", duration)
			time.Sleep(time.Duration(duration) * time.Second)
		}

		if err != nil {
			return "", fmt.Errorf("an error occured while fetching the rate limits: %s", err)
		}

		resp, _, err := client.Friends.IDs(&twitter.FriendIDParams{ScreenName: user})
		if err != nil {
			return "", fmt.Errorf("an error occured while fetching the user's details: %s", err)
		}
		// TODO: implement caching with redis / in memory, so we dont loop through users all the time

		for _, follower := range values.RightArray {
			var chosenUser twitter.User
			for _, u := range users {
				if u.ScreenName == follower {
					chosenUser = u
					break
				}
			}

			found, cursor, resolved := false, resp.NextCursor, false

			for _, id := range resp.IDs {
				if id == chosenUser.ID {
					response += FormatSuccessMessage(follower, user)
					found = true
					resolved = true
					break
				}
			}

			if !found && resp.NextCursor == 0 {
				response += FormatFailureMessage(follower, user)
				resolved = true
			}

			for !found && cursor != 0 {
				resp, _, err := client.Friends.IDs(&twitter.FriendIDParams{ScreenName: user, Cursor: cursor})
				if err != nil {
					return "", fmt.Errorf("an error occured while fetching the user's details: %s", err)
				}

				cursor = resp.NextCursor

				for _, id := range resp.IDs {
					if id == chosenUser.ID {
						response += FormatSuccessMessage(follower, user)
						found = true
						resolved = true
						break
					}
				}

			}

			if !found && !resolved {
				response += FormatFailureMessage(follower, user)
			}

		}

	}

	return response, nil
}
