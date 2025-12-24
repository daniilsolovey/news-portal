package postgres

import (
	"context"
	"fmt"
)

func (r *Repository) attachTagsBatch(ctx context.Context, news []News) ([]News, error) {
	if len(news) == 0 {
		return news, nil
	}

	tagSet := make(map[int32]struct{})
	for i := range news {
		for _, id := range news[i].TagIds {
			tagSet[id] = struct{}{}
		}
	}

	if len(tagSet) == 0 {
		for i := range news {
			news[i].Tags = []Tag{}
		}
		return news, nil
	}

	allTagIDs := make([]int32, 0, len(tagSet))
	for id := range tagSet {
		allTagIDs = append(allTagIDs, id)
	}

	tags, err := r.loadTags(ctx, allTagIDs)
	if err != nil {
		return nil, fmt.Errorf("get tags by ids: %w", err)
	}

	tagsByID := make(map[int32]Tag, len(tags))
	for i := range tags {
		t := tags[i]
		tagsByID[int32(t.TagID)] = t
	}

	for i := range news {
		ids := news[i].TagIds
		if len(ids) == 0 {
			news[i].Tags = []Tag{}
			continue
		}

		out := make([]Tag, 0, len(ids))
		for _, id := range ids {
			if t, ok := tagsByID[id]; ok {
				out = append(out, t)
			}
		}
		news[i].Tags = out
	}

	return news, nil
}

func (r *Repository) loadTags(ctx context.Context, tagIDs []int32) ([]Tag, error) {
	if len(tagIDs) == 0 {
		return []Tag{}, nil
	}

	tags, err := r.getTagsByIDs(ctx, tagIDs)
	if err != nil {
		return nil, fmt.Errorf("get tags: %w", err)
	}

	return tags, nil
}
