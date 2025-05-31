Hello!

You will need Postgres and Go installed to run this.

Please run:
```go install https://github.com/RobynIsBlue/borbAggieFRFR```
to use this repository.

To use this program and start configuring, please register or login with 
```gator register/login yourusername```

Here are a few commands, what they need, and what they do:


Users
None
Lists Users

Feeds
None (But must be logged in)
Lists all feeds

Following
None (But must be logged in)
Lists all followed feeds

AddFeed
name, url
Adds a feed and follows that feed automatically (must be logged in)

Follow
url (must be logged in)
Follows a feed

Unfollow
url (must be logged in)
Unfollows a feed

Agg
periodoftime (10s)
Scrapes all followed feeds and stores them as posts

Browse
amountofposts (optional, defaults to 2) (must be logged in)
Browses amount of specified posts sorted by recency
