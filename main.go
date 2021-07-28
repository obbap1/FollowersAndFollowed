package main

import (
	"fmt"
	utils "follow-info/utils"
)

func main() {
	err := utils.FetchMentions()
	if err != nil {
		fmt.Printf("Error is: %s", err)
	}
}
