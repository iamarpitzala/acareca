package subscription

import "time"

type Subscription struct {
	ID           int        `db:"id"`
	Name         string     `db:"name"`
	Description  *string    `db:"description"`
	Price        float64    `db:"price"`
	DurationDays int        `db:"duration_days"`
	IsActive     bool       `db:"is_active"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

type RqCreateSubscription struct {
	Name         string  `json:"name" validate:"required,max=255"`
	Description  *string `json:"description"`
	Price        float64 `json:"price" validate:"min=0"`
	DurationDays int     `json:"duration_days" validate:"required,min=1"`
	IsActive     *bool   `json:"is_active"`
}

func (r *RqCreateSubscription) ToSubscription() *Subscription {
	active := true
	if r.IsActive != nil {
		active = *r.IsActive
	}
	return &Subscription{
		Name:         r.Name,
		Description:  r.Description,
		Price:        r.Price,
		DurationDays: r.DurationDays,
		IsActive:     active,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

type RqUpdateSubscription struct {
	Name         *string  `json:"name" validate:"omitempty,max=255"`
	Description  *string  `json:"description"`
	Price        *float64 `json:"price" validate:"omitempty,min=0"`
	DurationDays *int     `json:"duration_days" validate:"omitempty,min=1"`
	IsActive     *bool    `json:"is_active"`
}

type RsSubscription struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	Description  *string   `json:"description,omitempty"`
	Price        float64   `json:"price"`
	DurationDays int       `json:"duration_days"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (s *Subscription) ToRs() *RsSubscription {
	return &RsSubscription{
		ID:           s.ID,
		Name:         s.Name,
		Description:  s.Description,
		Price:        s.Price,
		DurationDays: s.DurationDays,
		IsActive:     s.IsActive,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}
