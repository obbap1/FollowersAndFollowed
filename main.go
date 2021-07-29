package main

import (
	"fmt"
	utils "follow-info/utils"

	cron "github.com/robfig/cron/v3"

	cache "github.com/hashicorp/golang-lru"
)

var lru *cache.Cache

func setupCache() (*cache.Cache, error) {
	if lru != nil {
		return lru, nil
	}
	lruCache, err := cache.New(2000)
	if err != nil {
		return nil, fmt.Errorf("\n an error occured setting up cache: %s", err)
	}

	lru = lruCache

	return lruCache, nil
}

func main() {
	c := cron.New()
	lruCache, err := setupCache()
	if err != nil {
		fmt.Printf("\n Error setting up cache. Error is: %s", err)
		return
	}
	c.AddFunc("@every 2m", func() {
		fmt.Println("\n Fetching Mentions...")
		err := utils.FetchMentions(lruCache)
		if err != nil {
			fmt.Printf("\n Error fetching mentions. Error is: %s", err)
		}
	})

	// purge cache every 4hrs
	c.AddFunc("@every 4h", func() {
		fmt.Println("\n Purging cache...")
		lruCache.Purge()
	})

	fmt.Println("Starting, waiting for cron jobs...")

	c.Start()

	select {}
}
