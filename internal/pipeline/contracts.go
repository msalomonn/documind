// Package pipeline defines boundaries for later document slices; no adapter is wired yet.
package pipeline

import (
	"context"
	"io"
)

const (
	MaxUploadBytes    int64 = 10 << 20
	MaxPDFPages             = 30
	MaxTextCharacters       = 200000
)

type Identity struct{ UserID string }
type Sessions interface {
	Resolve(context.Context, string) (Identity, error)
}
type Store interface {
	Put(context.Context, string, io.Reader, int64, string) error
	Get(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}
type Parser interface {
	Text(context.Context, io.Reader, string) (string, error)
}
type Fact struct {
	Text          string `json:"text"`
	SourceExcerpt string `json:"sourceExcerpt"`
}
type Result struct {
	SchemaVersion int    `json:"schemaVersion"`
	Title         string `json:"title"`
	Summary       string `json:"summary"`
	KeyFacts      []Fact `json:"keyFacts"`
}
type Extractor interface {
	Extract(context.Context, string) (Result, error)
}
type Failure struct {
	Code      string
	Transient bool
}

func (f Failure) Error() string { return f.Code }
