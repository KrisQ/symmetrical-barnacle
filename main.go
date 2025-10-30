package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/KrisQ/symmetrical-barnacle/internal/config"
	"github.com/KrisQ/symmetrical-barnacle/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type commands struct {
	handlers map[string]func(*state, command) error
}

type command struct {
	name string
	args []string
}

func newCommands() *commands {
	return &commands{handlers: make(map[string]func(*state, command) error)}
}

func (c *commands) run(s *state, cmd command) error {
	h, ok := c.handlers[cmd.name]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}
	return h(s, cmd)
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.handlers[name] = f
}

func loginHandler(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("username required")
	}
	if _, err := s.db.GetUser(context.Background(), cmd.args[0]); err == sql.ErrNoRows {
		fmt.Println("user doesn't exists")
		os.Exit(1)
	}
	if err := s.cfg.SetUser(cmd.args[0]); err != nil {
		return err
	}
	fmt.Println("success")
	return nil
}

func registerHanlder(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("username required")
	}
	name := cmd.args[0]
	timestamp := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	if _, err := s.db.GetUser(context.Background(), name); err != sql.ErrNoRows {
		fmt.Println("user already exists")
		os.Exit(1)
	}
	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
		Name:      name,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	s.cfg.SetUser(name)
	fmt.Printf("user created %v", user.Name)
	return nil
}

func resetHandler(s *state, cmd command) error {
	if err := s.db.DeleteUsers(context.Background()); err != nil {
		fmt.Println("something went wrong, couldn't delete users")
		os.Exit(1)
	}
	fmt.Println("reset -> success")
	return nil
}

func getUsersHandler(s *state, cmd command) error {
	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Println("something went wrong, couldn't get users")
		os.Exit(1)
	}
	for _, user := range users {
		if user.Name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", user.Name)
			continue
		}
		fmt.Printf("* %s\n", user.Name)
	}
	return nil
}

func aggHandler(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("username required")
	}
	timeBetweenReqs, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		fmt.Println("provide valid time")
		os.Exit(1)
	}
	fmt.Printf("Collecting feeds every %s\n", timeBetweenReqs)

	ticker := time.NewTicker(timeBetweenReqs)
	for ; ; <-ticker.C {
		scrapeFeed(s)
	}
}

func addFeedHandler(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("username required")
	}
	name := cmd.args[0]
	url := cmd.args[1]
	timestamp := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
		Name:      name,
		Url:       url,
		UserID:    user.ID,
	})
	if err != nil {
		fmt.Println("couldn't create feed")
		os.Exit(1)
	}
	if _, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	}); err != nil {
		fmt.Println("couldn't create feed follow")
		os.Exit(1)
	}
	fmt.Printf("ID: %s\n", feed.ID)
	fmt.Printf("CreatedAt: %v\n", feed.CreatedAt)
	fmt.Printf("UpdatedAt: %v\n", feed.UpdatedAt)
	fmt.Printf("Name: %s\n", feed.Name)
	fmt.Printf("URL: %s\n", feed.Url)
	fmt.Printf("UserID: %s\n", feed.UserID)
	return nil
}

func feedsHandler(s *state, _ command) error {
	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		fmt.Println("cound't create feed")
		os.Exit(1)
	}
	for _, feed := range feeds {
		fmt.Printf("Name: %s\n", feed.Name)
		fmt.Printf("URL: %s\n", feed.Url)
		fmt.Printf("Username: %s\n", feed.Username)
	}
	return nil
}

func followHandler(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("url required")
	}
	url := cmd.args[0]
	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		fmt.Println("feed not found")
		os.Exit(0)
	}
	follow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		fmt.Println("womp womp didn't work")
		os.Exit(1)
	}
	fmt.Printf("ID: %s\n", follow.ID)
	fmt.Printf("CreatedAt: %v\n", follow.CreatedAt)
	fmt.Printf("UpdatedAt: %v\n", follow.UpdatedAt)
	fmt.Printf("User Name: %s\n", user.Name)
	fmt.Printf("Feed Name: %s\n", feed.Name)
	fmt.Printf("URL: %s\n", feed.Url)

	return nil
}

func followingHandler(s *state, cmd command, user database.User) error {
	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		fmt.Println("womp womp didn't work")
		os.Exit(1)
	}
	for _, follow := range follows {
		fmt.Printf("ID: %s\n", follow.ID)
		fmt.Printf("CreatedAt: %v\n", follow.CreatedAt)
		fmt.Printf("UpdatedAt: %v\n", follow.UpdatedAt)
		fmt.Printf("User Name: %s\n", follow.UserName)
		fmt.Printf("Feed Name: %s\n", follow.FeedName)
	}
	return nil
}

func unfollowHanlder(s *state, cmd command, user database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("url required")
	}
	url := cmd.args[0]
	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		fmt.Println("feed not found")
		os.Exit(0)
	}
	err = s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		fmt.Println("couldn't delete feed follow")
		os.Exit(0)
	}
	return nil
}

func scrapeFeed(s *state) error {
	nextFeed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		fmt.Println("couldn't find next feed")
		os.Exit(0)
	}
	err = s.db.MarkFeedFetched(context.Background(), nextFeed.ID)
	if err != nil {
		fmt.Println("couldn't find next feed")
		os.Exit(0)
	}
	feed, err := fetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		fmt.Println("couldn't find next feed")
		os.Exit(0)
	}
	
	ctx := context.Background()
	now := time.Now()
	
	for _, item := range feed.Channel.Item {
		publishedAt, _ := parsePubDate(item.PubDate)
		
		description := sql.NullString{Valid: false}
		if item.Description != "" {
			description = sql.NullString{String: item.Description, Valid: true}
		}
		
		_, err := s.db.CreatePost(ctx, database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   now,
			UpdatedAt:   now,
			Title:       item.Title,
			Url:         item.Link,
			Description: description,
			PublishedAt: publishedAt,
			FeedID:      nextFeed.ID,
		})
		
		if err != nil {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				continue
			}
			log.Printf("Error saving post %s: %v", item.Title, err)
		}
	}
	return nil
}

func browseHandler(s *state, cmd command, user database.User) error {
	limit := int32(2)
	
	if len(cmd.args) > 0 {
		parsedLimit, err := strconv.ParseInt(cmd.args[0], 10, 32)
		if err != nil {
			return fmt.Errorf("invalid limit: %s", cmd.args[0])
		}
		limit = int32(parsedLimit)
	}
	
	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		fmt.Println("couldn't get posts")
		os.Exit(1)
	}
	
	if len(posts) == 0 {
		fmt.Println("No posts found")
		return nil
	}
	
	for _, post := range posts {
		fmt.Printf("\nTitle: %s\n", post.Title)
		fmt.Printf("URL: %s\n", post.Url)
		if post.Description.Valid {
			fmt.Printf("Description: %s\n", post.Description.String)
		}
		fmt.Printf("Feed: %s\n", post.FeedName)
		if post.PublishedAt.Valid {
			fmt.Printf("Published: %s\n", post.PublishedAt.Time.Format(time.RFC822))
		}
		fmt.Println()
	}
	
	return nil
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	s := &state{cfg: &cfg}

	cmds := newCommands()
	cmds.register("login", loginHandler)
	cmds.register("register", registerHanlder)
	cmds.register("reset", resetHandler)
	cmds.register("users", getUsersHandler)
	cmds.register("agg", aggHandler)
	cmds.register("addfeed", middlewareLoggedIn(addFeedHandler))
	cmds.register("feeds", feedsHandler)
	cmds.register("follow", middlewareLoggedIn(followHandler))
	cmds.register("following", middlewareLoggedIn(followingHandler))
	cmds.register("unfollow", middlewareLoggedIn(unfollowHanlder))
	cmds.register("browse", middlewareLoggedIn(browseHandler))

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "not enough arguments")
		os.Exit(1)
	}
	name := os.Args[1]
	args := os.Args[2:]
	cmd := command{name: name, args: args}

	db, err := sql.Open("postgres", s.cfg.DBURL)
	if err != nil {
		fmt.Println("couldn't start db")
		os.Exit(1)
	}

	s.db = database.New(db)

	if err := cmds.run(s, cmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
