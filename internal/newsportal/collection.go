package newsportal

//go:generate colgen -imports=github.com/daniilsolovey/news-portal/internal/db
//colgen:News,Tag,Category
//colgen:News:Map(db),UniqueTagIDs
//colgen:Category:Map(db)
//colgen:Tag:Map(db),Index(ID)

func (ll NewsList) SetTags(tags Tags) {
	tagIndex := tags.IndexByID()
	for i := range ll {
		ll[i].Tags = make([]Tag, len(ll[i].TagIDs))
		for j, tagID := range ll[i].TagIDs {
			ll[i].Tags[j] = tagIndex[tagID]
		}
	}
}
