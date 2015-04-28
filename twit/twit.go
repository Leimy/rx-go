package twit

import (
	"github.com/ChimeraCoder/anaconda"
	"bufio"
	"os"
	"fmt"
)

var api *anaconda.TwitterApi

func init () {
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

// Tweeter lambda type
type Tweeter func (s string) error

// Create lambdas for tweeting
func MakeTweeter (s string) Tweeter {
	return func(s2 string) error {
		_, err := api.PostTweet(fmt.Sprintf("%s %s\n", s2, s), nil)
		return err
	}
}
