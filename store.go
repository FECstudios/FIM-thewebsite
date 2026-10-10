package main

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

type Section struct {
	ID        int       `json:"id"`
	ParentID  int       `json:"parent_id"`
	Prompt    string    `json:"prompt"`
	HTML      string    `json:"html"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) Latest() (Section, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.sections) == 0 {
		return Section{}, false
	}
	return s.sections[len(s.sections)-1], true
}

type Store struct {
	mu       sync.RWMutex
	path     string
	sections []Section
	nextID   int
}

func NewStore(path string) (*Store, error) {
	s := &Store{path: path, nextID: 1}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &s.sections); err != nil {
		return nil, err
	}
	for _, sec := range s.sections {
		if sec.ID >= s.nextID {
			s.nextID = sec.ID + 1
		}
	}

	return s, nil
}

func (s *Store) All() []Section {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Section, len(s.sections))
	copy(out, s.sections)
	return out
}

func (s *Store) Get(id int) (Section, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, sec := range s.sections {
		if sec.ID == id {
			return sec, true
		}
	}
	return Section{}, false
}

func (s *Store) Add(parentID int, prompt, html string) (Section, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sec := Section{
		ID:        s.nextID,
		ParentID:  parentID,
		Prompt:    prompt,
		HTML:      html,
		CreatedAt: time.Now(),
	}
	s.sections = append(s.sections, sec)
	s.nextID++

	if err := s.save(); err != nil {
		s.sections = s.sections[:len(s.sections)-1]
		s.nextID--
		return Section{}, err
	}
	return sec, nil
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.sections, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
