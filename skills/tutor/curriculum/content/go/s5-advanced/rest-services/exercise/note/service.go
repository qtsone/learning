package note

// Limits enforced by validation, counted in runes — "héllo" is five
// characters no matter how many bytes it takes.
const (
	MaxTitleLen   = 120
	MaxContentLen = 8000
)

// Service implements the business rules: normalization, validation, and
// ordering guarantees. It reaches storage only through the Store interface.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// Create normalizes and validates d, then stores it.
func (s *Service) Create(d Draft) (Note, error) {
	// TODO: trim leading/trailing whitespace from Title and Content,
	// validate the result (exact field messages are in LESSON.md's
	// acceptance criteria), and delegate to s.store.Create.
	return Note{}, nil
}

// Get returns the note with the given id.
func (s *Service) Get(id int64) (Note, error) {
	// TODO: delegate to the store.
	return Note{}, nil
}

// List returns all notes sorted by id, ascending. It never returns a nil
// slice — an empty service must encode as [] on the wire, not null.
func (s *Service) List() ([]Note, error) {
	// TODO: fetch from the store (which promises no particular order),
	// sort by id, and guarantee the result is non-nil.
	return nil, nil
}

// Update normalizes and validates d, then fully replaces the note with the
// given id.
func (s *Service) Update(id int64, d Draft) (Note, error) {
	// TODO: same normalization and validation as Create, then
	// s.store.Update.
	return Note{}, nil
}

// Delete removes the note with the given id.
func (s *Service) Delete(id int64) error {
	// TODO: delegate to the store.
	return nil
}
