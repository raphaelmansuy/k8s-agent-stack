package idgen

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/raphaelmansuy/agentstack/internal/domain/evaluation"
)

type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

func (g *UUIDGenerator) Generate(prefix string) string {
	if prefix != "" {
		return fmt.Sprintf("%s_%s", prefix, uuid.New().String())
	}
	return uuid.New().String()
}

// Ensure UUIDGenerator implements evaluation.IDGenerator
var _ evaluation.IDGenerator = (*UUIDGenerator)(nil)
