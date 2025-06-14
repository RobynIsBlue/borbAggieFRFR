package config

import (
	"borbAggregatorFRFR/internal/database"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

type State struct {
	Db   *database.Queries
	Conf *Config
}

type Command struct {
	Name      string
	Arguments []string
}

type Commands struct {
	CommandFunc map[string]func(*State, Command) error
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

const configFileName = ".gatorconfig.json"

// const dbUrl = `postgres://postgres:postgres@localhost:5432/gator`

func Read() Config {
	path := getHomePath()
	file, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
		fmt.Println("bad")
		return Config{}
	}
	var configBby Config
	json.Unmarshal(file, &configBby)
	return configBby
}

func SetUser(user string, config Config) {
	path := getHomePath()
	config.CurrentUserName = user
	jsonData, err := json.Marshal(config)
	if err != nil {
		fmt.Println("bad")
		fmt.Println(err)
		return
	}
	os.WriteFile(path, jsonData, 0644)
}

func getHomePath() string {
	homePath, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("bad")
		fmt.Println(err)
		return ""
	}
	return homePath + `/` + configFileName
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Arguments) == 0 {
		return errors.New("need username")
	}
	_, err := s.Db.GetUser(context.Background(), cmd.Arguments[0])
	if err != nil {
		return err
	}
	SetUser(cmd.Arguments[0], *s.Conf)
	fmt.Println("User has been set")
	return nil
}

func HandlerRegister(s *State, cmd Command) error {
	if len(cmd.Arguments) == 0 {
		return errors.New("need name for user")
	}

	user, err := s.Db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      cmd.Arguments[0],
	})
	if err != nil {
		return err
	}
	s.Conf.CurrentUserName = cmd.Arguments[0]
	HandlerLogin(s, cmd)
	fmt.Printf("User was created. %v", user)
	return nil
}

func HandlerReset(s *State, cmd Command) error {
	err := s.Db.DeleteTableRows(context.Background())
	if err != nil {
		fmt.Println("table rows not deleted")
		os.Exit(1)
		return err
	}
	fmt.Println("table rows deleted")
	os.Exit(0)
	return nil
}

func HandlerGetUsers(s *State, cmd Command) error {
	usersDatabase, err := s.Db.GetUsers(context.Background())
	var users []string
	for _, user := range usersDatabase {
		users = append(users, user.Name)
	}
	if err != nil {
		return err
	}
	user, err := s.Db.GetUser(context.Background(), s.Conf.CurrentUserName)
	if err != nil {
		return err
	}
	for _, u := range users {
		if u == user.Name {
			fmt.Printf("%s (current)\n", u)
		} else {
			fmt.Println(u)
		}
	}
	return nil
}

func HandlerBrowse(s *State, cmd Command, user database.User) error {
	if len(cmd.Arguments) == 0 {
		cmd.Arguments = append(cmd.Arguments, "2")
	}
	limit, err := strconv.Atoi(cmd.Arguments[0])
	if err != nil {
		return errors.New("invalid limit variable")
	}
	posts, err := s.Db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		return errors.New("could not get posts")
	}
	for _, post := range posts {
		spew.Dump(post)
	}
	return nil
}

func HandlerScrapeFeeds(s *State, cmd Command) error {
	timeFromUser := cmd.Arguments[0]
	fmt.Printf("Collecting feeds every %s", timeFromUser)
	parsedTime, err := time.ParseDuration(timeFromUser)
	if err != nil {
		return errors.New("could not parse time specified")
	}
	ticker := time.NewTicker(parsedTime)
	for ; ; <-ticker.C {
		feedToFetch, err := s.Db.GetNextFeedToFetch(context.Background())
		if err != nil {
			return errors.New("could not find next feed to fetch")
		}
		s.Db.MarkFeedFetched(context.Background(), feedToFetch.ID)
		feed, err := fetchFeed(context.Background(), feedToFetch.Url)
		if err != nil {
			return errors.New("could not fetch feed")
		}
		for _, item := range feed.Channel.Item {
			fmt.Println(item.Title)
			if item.PubDate == "" {
				item.PubDate = "N/A"
			}
			_, err := s.Db.CreatePost(context.Background(), database.CreatePostParams{
				ID:          uuid.New(),
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
				Title:       item.Title,
				Url:         item.Link,
				Description: item.Description,
				PublishedAt: item.PubDate,
				FeedID:      feedToFetch.ID,
			})
			if err != nil {
				if err, ok := err.(*pq.Error); !ok {
					fmt.Println(err)
					fmt.Printf("%T\n", err)
				}
			}
		}
	}
}

func HandlerAddFeed(s *State, cmd Command, user database.User) error {
	if len(cmd.Arguments) < 2 {
		return errors.New("must have at least two arguments")
	}
	name := cmd.Arguments[0]
	url := cmd.Arguments[1]
	feed, err := s.Db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})
	if err != nil {
		return err
	}
	err = HandlerFollow(s, Command{
		Arguments: []string{url},
	}, user)
	if err != nil {
		return err
	}
	fmt.Println(feed)
	return nil
}

func HandlerFeeds(s *State, cmd Command) error {
	feeds, err := s.Db.GetFeeds(context.Background())
	if err != nil {
		return err
	}
	for _, feed := range feeds {
		userName, err := s.Db.GetUserName(context.Background(), feed.UserID)
		if err != nil {
			return err
		}
		fmt.Printf("Name of Feed: %s\nURL of Feed: %s\nUser of Feed: %s\n\n",
			feed.Name, feed.Url, userName)
	}
	return nil
}

func HandlerFollow(s *State, cmd Command, user database.User) error {
	if len(cmd.Arguments) == 0 {
		return errors.New("need url to follow")
	}

	feed, err := s.Db.GetFeedFromURL(context.Background(), cmd.Arguments[0])
	if err != nil {
		return err
	}

	_, err = s.Db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return errors.New("already added link for this user")
	}
	fmt.Printf("feed %s added for user %s\n", feed.Name, user.Name)
	return nil
}

func HandlerFollowing(s *State, cmd Command, user database.User) error {
	feeds, err := s.Db.GetFeedFollowForUser(context.Background(), user.ID)
	if err != nil {
		return err
	}
	fmt.Printf("%s is following:\n", s.Conf.CurrentUserName)
	for _, feed := range feeds {
		feed, err := s.Db.GetFeedFromID(context.Background(), feed.FeedID)
		if err != nil {
			return err
		}
		fmt.Println(feed.Name)
	}
	return nil
}

func HandlerUnfollow(s *State, cmd Command, user database.User) error {
	feed, err := s.Db.GetFeedFromURL(context.Background(), cmd.Arguments[0])
	if err != nil {
		return errors.New("could not find url")
	}
	err = s.Db.DeleteFollowRecord(context.Background(), database.DeleteFollowRecordParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return errors.New("could not delete url")
	}
	return nil
}

func (c *Commands) Run(s *State, cmd Command) error {
	if _, ok := c.CommandFunc[cmd.Name]; ok {
		err := c.CommandFunc[cmd.Name](s, cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Commands) Register(name string, f func(*State, Command) error) {
	c.CommandFunc[name] = f
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	var client http.Client
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "gator")
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var item RSSFeed
	err = xml.Unmarshal(data, &item)
	if err != nil {
		return nil, err
	}

	item.Channel.Title = html.UnescapeString(item.Channel.Title)
	item.Channel.Description = html.UnescapeString(item.Channel.Description)
	for _, r := range item.Channel.Item {
		r.Title = html.UnescapeString(r.Title)
		r.Description = html.UnescapeString(r.Description)
	}

	return &item, nil
}

func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		user, err := s.Db.GetUser(context.Background(), s.Conf.CurrentUserName)
		if err != nil {
			return errors.New("not logged in")
		}
		return handler(s, cmd, user)
	}
}
