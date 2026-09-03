// Package doc holds the document domain: the type, its error vocabulary, and
// the storage interface everything above it depends on. It knows nothing about
// HTTP, and nothing about who is allowed to touch a document — that decision
// belongs to package authz.
package doc

import "errors"

// Document is a stored document. OwnerID is the user who created it: the one
// attribute the authorization policy needs to answer ownership questions.
type Document struct {
	ID       string `json:"id"`
	OwnerID  string `json:"owner_id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Archived bool   `json:"archived"`
}

// Draft is the client-supplied part of a document. It deliberately has no
// owner field: ownership comes from the authenticated subject, never from the
// request body.
type Draft struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// ErrNotFound is the domain's word for "no such document". Every Store
// implementation contracts to return it (possibly wrapped) when the id is
// unknown.
var ErrNotFound = errors.New("document not found")

// Store is what the service needs from storage, declared on the consumer side
// as in S5's REST lesson.
type Store interface {
	Create(ownerID string, d Draft) (Document, error)
	Get(id string) (Document, error)
	List() ([]Document, error)
	Update(id string, d Draft) (Document, error)
	Delete(id string) error
	Archive(id string) error
}
