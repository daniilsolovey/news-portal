package rest

//go:generate colgen -imports=github.com/daniilsolovey/news-portal/internal/newsportal
//colgen:News,Tag,Category
//colgen:News:Map(newsportal),Index(NewsID)
//colgen:Category:Map(newsportal),Index(CategoryID)
//colgen:Tag:Map(newsportal),Index(TagID)
