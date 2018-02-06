package tokens_test

import (
	"testing"

	stubs "github.com/pcarranza/meeseeks-box/testingstubs"
	"github.com/pcarranza/meeseeks-box/tokens"
)

func Test_TokenLifecycle(t *testing.T) {
	stubs.WithTmpDB(func(_ string) {
		id, err := tokens.Create(tokens.NewTokenRequest{
			UserID:      "myuser",
			ChannelLink: "mychannel",
			Text:        "echo hello",
		})
		stubs.Must(t, "could not create token", err)
		if id == "" {
			t.Fatal("create token returned an empty token id(?)")
		}

		tk, err := tokens.Get(id)
		stubs.Must(t, "could not get token back", err)

		stubs.AssertEquals(t, id, tk.TokenID)
		stubs.AssertEquals(t, "myuser", tk.UserID)
		stubs.AssertEquals(t, "mychannel", tk.ChannelLink)
		stubs.AssertEquals(t, "echo hello", tk.Text)
	})
}

func Test_TokenListing(t *testing.T) {
	stubs.WithTmpDB(func(_ string) {
		id, err := tokens.Create(tokens.NewTokenRequest{
			Text:        "echo something",
			UserID:      "myuser",
			ChannelLink: "mychannel",
		})
		stubs.Must(t, "could not create token", err)

		t1, err := tokens.Get(id)
		stubs.Must(t, "could not get token back", err)

		id, err = tokens.Create(tokens.NewTokenRequest{
			Text:        "echo something else",
			UserID:      "someone_else",
			ChannelLink: "my_other_channel",
		})
		stubs.Must(t, "could not create token", err)

		t2, err := tokens.Get(id)
		stubs.Must(t, "could not get token back", err)

		tt := []struct {
			Name     string
			Filter   tokens.Filter
			Expected []tokens.Token
		}{
			{
				Name:     "empty list",
				Expected: []tokens.Token{},
				Filter: tokens.Filter{
					Limit: 0,
				},
			},
			{
				Name:     "filter by username works",
				Expected: []tokens.Token{t2},
				Filter: tokens.Filter{
					Limit: 5,
					Match: func(tk tokens.Token) bool {
						return tk.UserID == t2.UserID
					},
				},
			},
			{
				Name:     "filter by channel works",
				Expected: []tokens.Token{t1},
				Filter: tokens.Filter{
					Limit: 5,
					Match: func(tk tokens.Token) bool {
						return tk.ChannelLink == t1.ChannelLink
					},
				},
			},
		}
		for _, tc := range tt {
			t.Run(tc.Name, func(t *testing.T) {
				token, err := tokens.Find(tc.Filter)
				stubs.Must(t, "failed to list tokens", err)
				stubs.AssertEquals(t, tc.Expected, token)
			})
		}
	})
}
