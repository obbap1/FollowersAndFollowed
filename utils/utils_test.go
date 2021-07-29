package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	FirstTestCase       = "_Obbap, BirthdayTracker following woonomic, ajmedwards"
	DummyTestCase       = "@obbap following following following @noliaaa"
	DummySecondTestCase = "@obbap, @kezi following"
	DummyThirdTestCase  = "following @noliaaa @omomo @mentus"
	SecondTestCase      = "@obbap following @noliaaa, @daveed_kz, @udori"
	ThirdTestCase       = "@obbap, @daveed_kz, @udori following @noliaaa"
)

func TestAttachAt(t *testing.T) {
	assert.True(t, AttachAt("pbaba") == "@pbaba", "Attach at is meant to add the @ to the start of the word")
}

func TestFormatSuccessMessage(t *testing.T) {
	assert.True(t, strings.Contains(FormatSuccessMessage("pbaba", "emeka"), "is following"), "Should contain is following")
}

func TestFormatFailureMessage(t *testing.T) {
	assert.True(t, strings.Contains(FormatFailureMessage("pbaba", "emeka"), "is NOT following"), "Should contain is NOT following")
}

func TestFindFollowersAndFollowed(t *testing.T) {
	ans, err := FindFollowersAndFollowed(FirstTestCase)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, len(ans.LeftArray) == 2, "Left array should have 2 items in it")
	assert.True(t, len(ans.RightArray) == 2, "Right array should have 2 items in it")
	assert.True(t, len(ans.AllArray) == 4, "All array should have 4 items in it")

	_, err = FindFollowersAndFollowed(DummyTestCase)
	if err == nil {
		t.Fatalf("Should fail with multiple followings")
	}

	_, err = FindFollowersAndFollowed(DummySecondTestCase)
	if err == nil {
		t.Fatalf("Should fail without text after the following")
	}

	_, err = FindFollowersAndFollowed(DummyThirdTestCase)
	if err == nil {
		t.Fatalf("Should fail without text before the following")
	}

	second, err := FindFollowersAndFollowed(SecondTestCase)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, len(second.LeftArray) == 1, "Left array should have 1 items in it")
	assert.True(t, len(second.RightArray) == 3, "Right array should have 3 items in it")
	assert.True(t, len(second.AllArray) == 4, "All array should have 4 items in it")

	third, err := FindFollowersAndFollowed(ThirdTestCase)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, len(third.LeftArray) == 3, "Left array should have 3 items in it")
	assert.True(t, len(third.RightArray) == 1, "Right array should have 1 items in it")
	assert.True(t, len(third.AllArray) == 4, "All array should have 4 items in it")

}
