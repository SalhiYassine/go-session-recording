package session

import "time"

type Session struct {
	ID                string
	ClientId          string
	VisitorId         string
	LastEventTime     time.Time
	DurationInSeconds int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type SessionCreationParams struct {
	ClientId  string
	VisitorId string
}

type SessionUpdateParams struct {
	LastEventTime     time.Time
	DurationInSeconds int
}

type SessionRepository interface {
	FindById(id string) (*Session, error)
	FindAllByClientId(id string) ([]*Session, error)
	Store(session *SessionCreationParams) error
	Update(session *Session) error
	Delete(id string) error
}

type SessionService interface {
	FindById(id string) (*Session, error)
	FindAllByClientId(id string) ([]*Session, error)
	Store(session *Session) error
	Update(session *Session) error
	Delete(id string) error
}

type SessionServiceImpl struct {
	repo SessionRepository
}

func (s *SessionServiceImpl) FindById(id string) (*Session, error) {
	return s.repo.FindById(id)
}

func (s *SessionServiceImpl) FindAllByClientId(id string) ([]*Session, error) {
	return s.repo.FindAllByClientId(id)
}

func (s *SessionServiceImpl) Store(session *Session) error {
	return s.repo.Store(session)
}

func (s *SessionServiceImpl) Update(session *Session) error {
	return s.repo.Update(session)
}

func (s *SessionServiceImpl) Delete(id string) error {
	return s.repo.Delete(id)
}

func NewSessionService(repo SessionRepository) SessionService {
	return &SessionServiceImpl{repo}
}
