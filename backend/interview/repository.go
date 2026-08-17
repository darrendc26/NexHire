package interview

import (
	"context"
	"errors"
	"sync"

	"nexhire/backend/models"
)

type MemoryRepository struct {
	mu         sync.RWMutex
	interviews map[string]*models.Interview
}

func NewRepository() Repository {
	return &MemoryRepository{
		interviews: make(map[string]*models.Interview),
	}
}

func (r *MemoryRepository) Create(ctx context.Context, interview *models.Interview) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.interviews[interview.ID]; exists {
		return errors.New("interview already exists")
	}

	r.interviews[interview.ID] = interview
	return nil
}

func (r *MemoryRepository) GetByID(ctx context.Context, id string) (*models.Interview, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	interview, exists := r.interviews[id]
	if !exists {
		return nil, errors.New("interview not found")
	}

	return interview, nil
}

func (r *MemoryRepository) GetByRecruiterID(ctx context.Context, recruiterID string) ([]models.Interview, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []models.Interview
	for _, interview := range r.interviews {
		if interview.RecruiterID == recruiterID {
			result = append(result, *interview)
		}
	}

	return result, nil
}

func (r *MemoryRepository) Update(ctx context.Context, interview *models.Interview) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.interviews[interview.ID]; !exists {
		return errors.New("interview not found")
	}

	r.interviews[interview.ID] = interview
	return nil
}

func (r *MemoryRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.interviews[id]; !exists {
		return errors.New("interview not found")
	}

	delete(r.interviews, id)
	return nil
}
