package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	cache "github.com/hashicorp/golang-lru"
)

const (
	ComparisonChecker = "following"
	MyUserName        = "AmeboTracker"
	FileHolder        = "holder.data"
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

func AttachDummyAt(s string) string {
	return "@/" + s
}

func FormatSuccessMessage(follower, user string) string {
	return "\n" + "Yes. " + AttachDummyAt(user) + " is following " + AttachDummyAt(follower)
}

func FormatFailureMessage(follower, user string) string {
	return "\n" + "No. " + AttachDummyAt(user) + " is NOT following " + AttachDummyAt(follower)
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

	if len(allArray) == 0 {
		return nil, fmt.Errorf("no users to seach for")
	}

	return &FollowersAndFollowed{
		LeftArray:  cleanLeftArray,
		RightArray: cleanRightArray,
		AllArray:   allArray,
	}, nil

}

func FetchMentions(lruCache *cache.Cache) error {

	client, httpClient := SetupTwitterClient()

	data := make(map[string]interface{})

	MY_ID := os.Getenv("MY_ID")

	fmt.Println("My id is...", MY_ID)

	url := twitterBaseUrl + MY_ID + "/mentions"

	fileData, err := ioutil.ReadFile(FileHolder)

	if err != nil {
		return fmt.Errorf("an error occured while reading file %s: %s", FileHolder, err)
	}

	if len(fileData) != 0 {
		fmt.Println("file data...", string(fileData))
		url += "?since_id=" + strings.TrimSpace(string(fileData))
	}

	resp, err := httpClient.Get(url)

	if err != nil {
		return fmt.Errorf("an error occured while fetching the user's details: %s", err)
	}

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return fmt.Errorf("an error occured while fetching the request body: %s", err)
	}

	if body == nil {
		return fmt.Errorf("no mentions are available")
	}

	err = json.Unmarshal(body, &data)

	if err != nil {
		return fmt.Errorf("an error occured while unmarshalling the request body: %s", err)
	}

	resultMap := data["meta"].(map[string]interface{})

	if resultMap["result_count"].(float64) == 0 {
		fmt.Println("There are no new mentions!")
		return nil
	}

	fmt.Printf("Number of mentions to process: %f \n", resultMap["result_count"].(float64))

	content := data["data"].([]interface{})

	mentionsChan := make(chan string, len(content))

	for k, v := range content {
		vi := v.(map[string]interface{})

		text := vi["text"].(string)

		id, err := strconv.ParseInt(vi["id"].(string), 10, 64)

		if err != nil {
			return fmt.Errorf("error while parsing int64: %s", err)
		}

		if k == len(content)-1 {
			ioutil.WriteFile(FileHolder, []byte(strconv.FormatInt(id, 10)), 0777)
		}

		go func(s string, id int64) {
			tweet, err := FetchResults(s, lruCache)
			if err != nil {
				mentionsChan <- fmt.Sprintf("Error is: %s", err)
				return
			}
			// TODO: ratelimit here as well

			if err != nil {
				mentionsChan <- fmt.Sprintf("Error is: %s", err)
				return
			}

			if os.Getenv("STUB_TWEET") == "true" {
				fmt.Println("\n Fake sending tweet...", tweet)
				mentionsChan <- fmt.Sprintf("\n done with processing: %s", tweet)
			} else {
				// send tweet
				tweetSent, _, err := client.Statuses.Update(tweet, &twitter.StatusUpdateParams{InReplyToStatusID: id})
				if err != nil {
					mentionsChan <- fmt.Sprintf("Error is: %s", err)
					return
				}
				// send signal
				mentionsChan <- fmt.Sprintf("\n done with processing: %s", tweetSent.Text)

			}

		}(text, id)

	}

	for range content {
		fmt.Println(<-mentionsChan)
	}

	return nil

}

func FetchResults(sentence string, lruCache *cache.Cache) (string, error) {

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

		limit, _, err := client.RateLimits.Status(&twitter.RateLimitParams{Resources: []string{"friends"}})

		rateLimit := limit.Resources.Friends["/friends/list"]

		fmt.Printf("%d of %d requests are left", rateLimit.Remaining, rateLimit.Limit)

		if rateLimit.Remaining == 0 {
			duration := int64(rateLimit.Reset) - time.Now().Unix()
			fmt.Printf("Sleeping for %d seconds.........******************..............*********", duration)
			time.Sleep(time.Duration(duration) * time.Second)
		}

		if err != nil {
			return "", fmt.Errorf("an error occured while fetching the rate limits: %s", err)
		}

		resp, _, err := client.Friends.IDs(&twitter.FriendIDParams{ScreenName: user})
		if err != nil {
			return "", fmt.Errorf("an error occured while fetching the user's details: %s", err)
		}

		for _, follower := range values.RightArray {
			// check if value is in cache
			if lruCache.Contains(user + ":" + follower) {
				fmt.Printf("Getting value for %s and %s from cache!", user, follower)
				if val, ok := lruCache.Get(user + ":" + follower); ok {
					if val.(bool) {
						response += FormatSuccessMessage(follower, user)
					} else {
						response += FormatFailureMessage(follower, user)
					}
					continue
				}
			}
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

			//add to cache
			lruCache.Add(user+":"+follower, found)
		}

	}

	return response, nil
}
