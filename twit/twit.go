package twit

import (
	"bufio"
	"fmt"
	"os"

	"github.com/ChimeraCoder/anaconda"
)

var api *anaconda.TwitterApi

func init() {
	f, err := os.Open("CREDENTIALS")
	if err != nil {
		panic(err)
	}
	s := bufio.NewScanner(f)
	s.Scan()
	anaconda.SetConsumerKey(s.Text())
	s.Scan()
	anaconda.SetConsumerSecret(s.Text())
	s.Scan()
	tkn := s.Text()
	s.Scan()
	tknSkrt := s.Text()
	api = anaconda.NewTwitterApi(tkn, tknSkrt)
}

// Tweeter is a lambda type
type Tweeter func(s string) error

// MakeTweeter creates lambdas for tweeting
func MakeTweeter(s string) Tweeter {
	return func(s2 string) error {
		_, err := api.PostTweet(fmt.Sprintf("%s %s\n", s2, s), nil)
		return err
	}
}
