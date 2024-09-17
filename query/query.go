package query

import (
	"gorm.io/gorm"

	"main/models"
)

type Query struct {
	db *gorm.DB
}

func NewQuery(db *gorm.DB) *Query {
	return &Query{db: db}
}

func (q *Query) Execute(userID models.ID) (*models.User, error) {
	var u *models.User
	if err := q.db.First(&u, userID).Error; err != nil {
		return nil, err
	}

	return u, nil
}
