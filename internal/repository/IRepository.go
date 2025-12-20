package repository

import (
	"github.com/daniilsolovey/news-portal/internal/repository/postgres"
)

type IRepository interface {
	Postgres() postgres.IRepository
}
