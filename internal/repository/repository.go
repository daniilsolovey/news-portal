package repository

import (
	"github.com/daniilsolovey/news-portal/internal/repository/postgres"
)

type repo struct {
	pg postgres.IRepository
}

func New(pg postgres.IRepository) IRepository {
	return &repo{pg: pg}
}

func (r *repo) Postgres() postgres.IRepository {
	return r.pg
}
