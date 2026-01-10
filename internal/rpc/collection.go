package rpc

//go:generate colgen -imports=github.com/daniilsolovey/news-portal/internal/newsportal -funcpkg=newsportal
//colgen:News,Tag,Category,NewsSummary
//colgen:News:Map(newsportal),Index(NewsID)
//colgen:Category:Map(newsportal),Index(CategoryID)
//colgen:Tag:Map(newsportal),Index(TagID)
//colgen:NewsSummary:Map(newsportal.News)
